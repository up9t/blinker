import { useEffect, useState } from "react";
import { WailsStatisticsService } from "../lib/adapters/wails_statistics.ts";
import { formatDuration } from "../lib/duration.ts";
import type { IStatBucket, IStatPeriodSet, IStatistics } from "../lib/ports/statistics.ts";

const statisticsService = new WailsStatisticsService();

const hourFormat = new Intl.DateTimeFormat(undefined, { hour: "numeric" });
const dayFormat = new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric" });

const formatHour = (start: number) => hourFormat.format(new Date(start * 1000));
const formatDay = (start: number) => dayFormat.format(new Date(start * 1000));

export function StatisticsPanel() {
  const [stats, setStats] = useState<IStatistics | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    void refresh();
  }, []);

  async function refresh() {
    setError("");

    try {
      setStats(await statisticsService.statistics());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <>
      <button type="button" onClick={() => void refresh()}>
        Refresh
      </button>
      {error !== "" && <p role="alert">{error}</p>}
      {stats === null && error === "" && <p>Loading…</p>}

      {stats !== null && (
        <>
          <PeriodSection title="Blinks" periodSet={stats.blinks} value={String} />
          <PeriodSection title="Screen time" periodSet={stats.screenTime} value={formatDuration} />
        </>
      )}
    </>
  );
}

interface PeriodSectionProps {
  title: string;
  periodSet: IStatPeriodSet;
  value(value: number): string;
}

function PeriodSection({ title, periodSet, value }: PeriodSectionProps) {
  return (
    <section>
      <h2>{title}</h2>
      <p>Today: {value(periodSet.today.total)}</p>
      <p>This week: {value(periodSet.week.total)}</p>
      <p>This month: {value(periodSet.month.total)}</p>

      <Graph
        caption="Today, per hour"
        buckets={periodSet.today.buckets}
        label={formatHour}
        value={value}
      />
      <Graph
        caption="This week, per day"
        buckets={periodSet.week.buckets}
        label={formatDay}
        value={value}
      />
      <Graph
        caption="This month, per day"
        buckets={periodSet.month.buckets}
        label={formatDay}
        value={value}
      />
    </section>
  );
}

interface GraphProps {
  caption: string;
  buckets: IStatBucket[] | null;
  label(start: number): string;
  value(value: number): string;
}

function Graph({ caption, buckets, label, value }: GraphProps) {
  return (
    <table>
      <caption>{caption}</caption>
      <thead>
        <tr>
          <th scope="col">Time</th>
          <th scope="col">Value</th>
        </tr>
      </thead>
      <tbody>
        {(buckets ?? []).map((bucket) => (
          <tr key={bucket.start}>
            <th scope="row">{label(bucket.start)}</th>
            <td>{value(bucket.value)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
