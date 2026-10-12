// The deployment's public per-country values (A-20), from GET /api/config. The web app has
// none of its own: the same bundle serves every country. main.tsx loads them once, before the
// first page renders.
import { api } from "./api";

export type AppConfig = { locale: string; timeZone: string };

let current: AppConfig | undefined;

export async function loadConfig(): Promise<AppConfig> {
  current = await api<AppConfig>("/config");
  return current;
}

/** The locale for dates, or undefined (the browser's own) if the config couldn't be loaded. */
export const locale = () => current?.locale;

export function resetConfigForTests() {
  current = undefined;
}
