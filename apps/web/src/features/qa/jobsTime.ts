// Times on the scheduled-jobs page (/qa/jobs, S16). The date-and-time box's own helpers are in
// lib/localTime.ts.

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

/**
 * Who runs the jobs by themselves (GET /api/qa/jobs `runner`): locally the API's timer (S16),
 * on AWS the jobs Lambda's schedule (S17), every intervalSeconds.
 */
export function whoRuns(runner: "timer" | "schedule", intervalSeconds: number): string {
  if (runner !== "schedule") return `the local server runs the jobs every ${intervalSeconds} seconds`;
  const minutes = Math.round(intervalSeconds / 60);
  return `a schedule on AWS runs the jobs every ${minutes === 1 ? "minute" : `${minutes} minutes`}`;
}
