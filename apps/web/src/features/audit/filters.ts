// The Audit screen's filters (S11). They live in the page's URL as days and text, so a reload, a
// deep link or the back button keeps them. auditQuery turns them into GET /api/o/<trust>/audit's
// query string.

export type AuditFilters = {
  /** A staff member's id, "patient" (the patient's link), or "" for everyone. */
  user: string;
  /** Days, YYYY-MM-DD, or "": from the start of `from` to the end of `to`. */
  from: string;
  to: string;
  /** Part of a patient's name, their hospital number or their record's id. */
  patient: string;
};

export function readFilters(params: URLSearchParams): AuditFilters {
  return {
    user: params.get("user") ?? "",
    from: params.get("from") ?? "",
    to: params.get("to") ?? "",
    patient: params.get("patient") ?? "",
  };
}

/** The browser's midnight at the start of a YYYY-MM-DD day, plus `days`; null if it isn't a real day. */
function midnight(day: string, days = 0): Date | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(day);
  if (!m) return null;
  const [y, mo, d] = [Number(m[1]), Number(m[2]) - 1, Number(m[3])];
  const date = new Date(y, mo, d);
  // new Date(2026, 1, 30) quietly becomes 2 March: refuse it instead.
  if (date.getFullYear() !== y || date.getMonth() !== mo || date.getDate() !== d) return null;
  return new Date(y, mo, d + days);
}

/**
 * The API's query for the filters. The days become the browser's own midnights (from: the start of
 * `from`; to: the start of the day after `to`), sent as exact times, so the code names no time zone.
 * The export adds format=csv and the browser's zone, so the CSV's times match the screen.
 */
export function auditQuery(f: AuditFilters, opts?: { csv: boolean; timeZone: string }): URLSearchParams {
  const q = new URLSearchParams();
  if (f.user) q.set("user", f.user);
  const from = f.from ? midnight(f.from) : null;
  const to = f.to ? midnight(f.to, 1) : null;
  if (from) q.set("from", from.toISOString());
  if (to) q.set("to", to.toISOString());
  if (f.patient.trim()) q.set("patient", f.patient.trim());
  if (opts?.csv) {
    q.set("format", "csv");
    q.set("tz", opts.timeZone);
  }
  return q;
}

/** Whether From or To is set but isn't a real day (e.g. 30 February in a typed or linked URL). */
export const badDays = (f: AuditFilters) => [f.from, f.to].some((d) => d !== "" && midnight(d) === null);

/**
 * A list screen's row can touch every patient in the trust, so the API names only the first ten
 * (S11 verification 1) and gives the count: "A, B and 330 more". The CSV names them all.
 */
export const namesOf = (shown: string[], count: number) =>
  shown.join(", ") + (count > shown.length ? ` and ${count - shown.length} more` : "");

/** The filters with any day that isn't real emptied: the form can't show 30 February anyway. */
export const withoutBadDays = (f: AuditFilters): AuditFilters => ({
  ...f,
  from: f.from && midnight(f.from) ? f.from : "",
  to: f.to && midnight(f.to) ? f.to : "",
});
