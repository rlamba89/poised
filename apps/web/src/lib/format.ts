import { locale } from "./config";

// Dates are formatted in the deployment's locale (GET /api/config, A-20), e.g. en-GB or en-US.

/** A date and time, in the browser's time zone: "12/10/2026, 13:00" in en-GB. */
export const formatDateTime = (iso: string) =>
  new Intl.DateTimeFormat(locale(), { dateStyle: "short", timeStyle: "short" }).format(new Date(iso));

/** "11 December 1974" in en-GB, for a YYYY-MM-DD date such as a date of birth. */
export const formatDate = (ymd: string) =>
  new Intl.DateTimeFormat(locale(), { dateStyle: "long", timeZone: "UTC" }).format(new Date(ymd + "T00:00:00Z"));

/** A YYYY-MM-DD date in the short style ("03/11/2026" in en-GB), or "—" for none (S19's episode dates). */
export const formatOptionalDate = (ymd: string | null | undefined) =>
  ymd ? new Intl.DateTimeFormat(locale(), { dateStyle: "short", timeZone: "UTC" }).format(new Date(ymd + "T00:00:00Z")) : "—";

/** The day only, in the browser's time zone: "12/10/2026" in en-GB. */
export const formatShortDate = (iso: string) => new Date(iso).toLocaleDateString(locale());

/** The day in the medium style, in the browser's time zone: "3 Mar 2026" in en-GB (S25's pre-fill dates). */
export const formatDay = (iso: string) => new Intl.DateTimeFormat(locale(), { dateStyle: "medium" }).format(new Date(iso));

/** The month and year, in the browser's time zone: "March 2026" in en-GB (S26's "the answers you gave … in March 2026"). */
export const formatMonth = (iso: string) => new Intl.DateTimeFormat(locale(), { month: "long", year: "numeric" }).format(new Date(iso));
