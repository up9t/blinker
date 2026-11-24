import { useEffect, useState } from "react";
import { Events } from "@wailsio/runtime";
import { BreakStart } from "../../bindings/events/index.js";
import { WailsBreakpointService } from "../lib/adapters/wails_breakpoint.ts";
import { POSTPONE_DELAY_MS, formatCountdown, remainingMs } from "../lib/breakpoint.ts";
import type { IBreakInfo } from "../lib/ports/breakpoint.ts";

const breakpointService = new WailsBreakpointService();
const TICK_MS = 250;

export function BreakOverlay() {
  const [info, setInfo] = useState<IBreakInfo | null>(null);
  const [endsAt, setEndsAt] = useState(0);
  const [now, setNow] = useState(() => Date.now());
  const [postponing, setPostponing] = useState(false);

  const show = (next: IBreakInfo) => {
    const startedAt = Date.now();

    setInfo(next);
    setEndsAt(startedAt + next.remainingMs);
    setNow(startedAt);
  };

  useEffect(() => {
    // first load: the backend already started the break before showing this page
    breakpointService
      .activeBreak()
      .then((active) => {
        if (active !== null) {
          show(active);
        }
      })
      .catch((err) => console.error("Failed to load the running break", err));

    // a queued break starts while this page is already mounted
    return Events.On(BreakStart, (event) => show(event.data));
  }, []);

  useEffect(() => {
    if (info === null) {
      return;
    }

    const timer = window.setInterval(() => setNow(Date.now()), TICK_MS);

    return () => window.clearInterval(timer);
  }, [info]);

  const postpone = async () => {
    setPostponing(true);

    try {
      await breakpointService.postpone(POSTPONE_DELAY_MS);
      setInfo(null);
    } catch (err) {
      console.error("Failed to postpone the break", err);
    } finally {
      setPostponing(false);
    }
  };

  if (info === null) {
    return <p>No break.</p>;
  }

  return (
    <main>
      <h1>{info.name}</h1>

      <p role="timer" aria-label="Time left">
        {formatCountdown(remainingMs(endsAt, now))}
      </p>

      <p>
        Current time:{" "}
        {new Date(now).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" })}
      </p>

      {/* a postponed break can't be snoozed again, the backend rejects it too */}
      {!info.postponed && (
        <button type="button" onClick={postpone} disabled={postponing}>
          Postpone {POSTPONE_DELAY_MS / 1000}s
        </button>
      )}
    </main>
  );
}
