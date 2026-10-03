import { LazyStore } from "@tauri-apps/plugin-store";
import type { Breakpoint, Theme } from "@/types";
import { getRandomId } from "@/utils";
import { reactive, watch } from "vue";

export interface Settings {
  thresholdEyesClosed: number;
  thresholdEyesOpened: number;
  blinkTimeout: number;
  notifyBeforeSecond: number;
  breakpoints: Breakpoint[];
  theme: Theme;
  autoStart: boolean;
}

export const DEFAULT_SETTINGS: Settings = {
  thresholdEyesClosed: 0.5,
  thresholdEyesOpened: 0.4,
  blinkTimeout: 1.0,
  notifyBeforeSecond: 20,
  breakpoints: [
    {
      id: getRandomId(),
      interval: 20,
      intervalUnit: "minute",
      duration: 2,
      durationUnit: "minute",
      enabled: true,
    },
  ],
  theme: "system",
  autoStart: false,
};

const STORE_KEY = "settings";

export async function loadSettings(): Promise<Settings> {
  try {
    const store = new LazyStore("settings.json");
    const storedSettings = await store.get(STORE_KEY);

    if (storedSettings && typeof storedSettings === "object") {
      const settings = storedSettings as Partial<Settings>;
      return {
        ...DEFAULT_SETTINGS,
        ...settings,
        breakpoints: (settings.breakpoints ?? DEFAULT_SETTINGS.breakpoints).map((bp) => ({
          ...bp,
          enabled: bp.enabled ?? true,
        })),
      };
    }

    return DEFAULT_SETTINGS;
  } catch (error) {
    console.warn("Failed to load settings from store, using defaults:", error);
    return DEFAULT_SETTINGS;
  }
}

export async function saveSettings(settings: Settings): Promise<void> {
  try {
    const store = new LazyStore("settings.json");
    await store.set(STORE_KEY, settings);
    await store.save();
  } catch (error) {
    console.error("Failed to save settings:", error);
  }
}

export async function resetToDefaults(): Promise<Settings> {
  try {
    const store = new LazyStore("settings.json");
    const freshDefaults: Settings = {
      ...DEFAULT_SETTINGS,
      breakpoints: DEFAULT_SETTINGS.breakpoints.map((bp) => ({
        ...bp,
        id: getRandomId(),
      })),
    };
    await store.set(STORE_KEY, freshDefaults);
    await store.save();
    return freshDefaults;
  } catch (error) {
    console.error("Failed to reset settings:", error);
    return DEFAULT_SETTINGS;
  }
}

export const settings = reactive<Settings>(structuredClone(DEFAULT_SETTINGS));

/**
 * Watch for changes to settings and save them to the store (debounced)
 */
let saveTimeoutId: ReturnType<typeof setTimeout> | null = null;

watch(
  settings,
  (newSettings) => {
    if (saveTimeoutId) {
      clearTimeout(saveTimeoutId);
    }

    saveTimeoutId = setTimeout(() => {
      saveSettings(newSettings);
    }, 500); // Debounce saves by 500ms
  },
  { deep: true },
);

export async function hydrateSettings(loadedSettings: Settings) {
  Object.assign(settings, loadedSettings);
}
