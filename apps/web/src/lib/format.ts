import { locale } from "./config";

// Dates are formatted in the deployment's locale (GET /api/config, A-20), e.g. en-GB or en-US.

/** A date and time, in the browser's time zone: "12/10/2026, 13:00" in en-GB. */
export const formatDateTime = (iso: string) =>
  new Intl.DateTimeFormat(locale(), { dateStyle: "short", timeStyle: "short" }).format(new Date(iso));

/** "11 December 1974" in en-GB, for a YYYY-MM-DD date such as a date of birth. */
export const formatDate = (ymd: string) =>
  new Intl.DateTimeFormat(locale(), { dateStyle: "long", timeZone: "UTC" }).format(new Date(ymd + "T00:00:00Z"));

/** The day only, in the browser's time zone: "12/10/2026" in en-GB. */
export const formatShortDate = (iso: string) => new Date(iso).toLocaleDateString(locale());
