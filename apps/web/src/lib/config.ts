// The deployment's public per-country values (A-20), from GET /api/config. The web app has
// none of its own: the same bundle serves every country. main.tsx loads them once, before the
// first page renders.
import { api } from "./api";
import type { Cognito } from "./signin";

export type AppConfig = {
  locale: string;
  timeZone: string;
  /** The deployment's patient identifier, e.g. { type: "nhs", label: "NHS number" } (S13). */
  nationalId: { type: string; label: string };
  /** How staff sign in (S12): the dev login (local only), Cognito's managed login, or both locally. */
  staffSignIn?: { devLogin: boolean; cognito: Cognito | null };
  /** The estimated time's seconds for each question the patient sees (S36, PX-10; Go's SECONDS_PER_QUESTION). */
  secondsPerQuestion?: number;
};

let current: AppConfig | undefined;

// The first page waits for the config, so a request that hangs (e.g. a Lambda cold start gone
// wrong) is given up after this long; the pages then render with the browser's locale.
const TIMEOUT_MS = 5000;

export async function loadConfig(): Promise<AppConfig> {
  const abort = new AbortController();
  const timer = setTimeout(() => abort.abort(new Error("GET /api/config took more than 5 seconds")), TIMEOUT_MS);
  try {
    current = await api<AppConfig>("/config", { signal: abort.signal });
    return current;
  } finally {
    clearTimeout(timer);
  }
}

/** The locale for dates, or undefined (the browser's own) if the config couldn't be loaded. */
export const locale = () => current?.locale;

/** The deployment's default time zone, or undefined if the config couldn't be loaded. */
export const deploymentTimeZone = () => current?.timeZone;

/** What staff call the patient identifier ("NHS number"), or a neutral name if the config couldn't be loaded. */
export const nationalIdLabel = () => current?.nationalId?.label ?? "National ID";

/**
 * The estimated time's seconds per question (S36), or undefined if the config couldn't be loaded:
 * then no estimate is shown. The number is the deployment's setting, never one of the web app's.
 */
export const secondsPerQuestion = () => current?.secondsPerQuestion;

/** How staff sign in. Without the config (it couldn't be loaded), the dev login is tried. */
export const staffSignIn = (): { devLogin: boolean; cognito: Cognito | null } =>
  current?.staffSignIn ?? { devLogin: true, cognito: null };

export function resetConfigForTests() {
  current = undefined;
}
