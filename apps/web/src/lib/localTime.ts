// Times in a date-and-time box, which holds "YYYY-MM-DDTHH:mm" in the browser's time zone; the API
// takes an exact time (ISO 8601). The box can hold what no date can be, such as a five-digit year.

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
