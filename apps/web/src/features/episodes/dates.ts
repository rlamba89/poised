// An episode's planned procedure and POA appointment dates (S19) are browser date inputs. A date
// typed only in part (e.g. "28/10/yyyy") has the value "", the same as no date, and only
// `validity.badInput` tells them apart (S19 test report 1, F1), so the page checks it before sending.

/** A date input as far as this check needs it (an HTMLInputElement, or null before it's drawn). */
export type DateInput = { validity: { badInput: boolean } } | null;

/** The message for the first field typed only in part, as the API words it, or "" if none is. */
export function partlyTypedDate(fields: { label: string; input: DateInput }[]): string {
  const bad = fields.find((f) => f.input?.validity.badInput);
  return bad ? `Enter the ${bad.label} as a date.` : "";
}
