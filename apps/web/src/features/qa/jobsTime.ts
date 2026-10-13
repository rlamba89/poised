// Times on the scheduled-jobs page (/qa/jobs, S16). A date-and-time box holds "YYYY-MM-DDTHH:mm"
// in the browser's time zone; the API takes an exact time (ISO 8601).

const two = (n: number) => String(n).padStart(2, "0");

/** The value for a date-and-time box: d in the browser's zone, to the minute. */
export function toLocalInput(d: Date): string {
  return `${d.getFullYear()}-${two(d.getMonth() + 1)}-${two(d.getDate())}T${two(d.getHours())}:${two(d.getMinutes())}`;
}

/** The box's value as an exact time for the API, or null if it isn't a date and time. */
export function fromLocalInput(value: string): string | null {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) return null;
  const d = new Date(value); // a date-time without a zone is read in the browser's zone
  return Number.isNaN(d.getTime()) ? null : d.toISOString();
}

/**
 * d rounded up to the next whole minute (unchanged if it's already whole). The box holds minutes
 * only, so "now + 2 days" written down would otherwise be up to a minute early, and miss a
 * reminder due two days after an invite sent seconds ago.
 */
export function ceilMinute(d: Date): Date {
  const out = new Date(d);
  if (out.getSeconds() || out.getMilliseconds()) out.setSeconds(60, 0);
  return out;
}

/** d moved on by whole calendar days, keeping its time of day. */
export function addDays(d: Date, days: number): Date {
  const out = new Date(d);
  out.setDate(out.getDate() + days);
  return out;
}

/** The timer's offset in minutes for days and hours ahead, or null outside 0 to 90 days (the API's limit). */
export function offsetMinutes(days: number, hours: number): number | null {
  if (!Number.isInteger(days) || !Number.isInteger(hours) || days < 0 || hours < 0) return null;
  const m = days * 24 * 60 + hours * 60;
  return m > 90 * 24 * 60 ? null : m;
}
