// The date of birth the patient types on their link's page (S14): three boxes, day, month and
// year, as GOV.UK's date pattern does (easier on a phone than a date picker).

/** The date as the API takes it, YYYY-MM-DD, or null unless it's a whole, real date. */
export function dobValue(day: string, month: string, year: string): string | null {
  const [d, m, y] = [day, month, year].map((v) => v.trim());
  if (!/^\d{1,2}$/.test(d) || !/^\d{1,2}$/.test(m) || !/^\d{4}$/.test(y)) return null;
  const [dn, mn, yn] = [Number(d), Number(m), Number(y)];
  // A real date: Date rolls 31 February over into March, so check it comes back the same.
  const date = new Date(Date.UTC(yn, mn - 1, dn));
  if (date.getUTCFullYear() !== yn || date.getUTCMonth() !== mn - 1 || date.getUTCDate() !== dn) return null;
  return `${y}-${String(mn).padStart(2, "0")}-${String(dn).padStart(2, "0")}`;
}
