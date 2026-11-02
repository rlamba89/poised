// Observations (plan C7, session S46; OBS-01–04): the rules the web app needs for a clinician's
// readings. Go stores them and enforces the hard limits (internal/observation); here are which
// readings count, the hospital BMI, and the soft warning shown before a reading is saved.
// S47's assessments pre-fill from these (obs.*).

/** A reading as GET …/observations sends it. value is in the kind's stored unit; null for a note. */
export type Reading = {
  id: string;
  kind: string;
  value: number | null;
  value2: number | null;
  unit: string | null;
  note: string | null;
  takenAt: string;
  enteredAt: string;
  enteredByName: string;
  /** The reading this one corrects, with why. */
  corrects: string | null;
  reason: string | null;
  /** The reading that corrected this one: it no longer counts. */
  supersededBy: string | null;
  /** Entered in error: who, when and why. It no longer counts. */
  errorAt: string | null;
  errorByName: string | null;
  errorReason: string | null;
};

export type ObservationRange = { softMin: number; softMax: number; hardMin: number; hardMax: number };
export type ObservationRanges = Record<string, ObservationRange>;

/** The body mass index from a height in cm and a weight in kg, to one decimal place (the HQ's bmi() too). */
export function bmiOf(heightCm: number | undefined, weightKg: number | undefined): number | undefined {
  if (heightCm === undefined || weightKg === undefined || !(heightCm > 0) || !(weightKg > 0)) return undefined;
  return Math.round((weightKg / (heightCm / 100) ** 2) * 10) / 10;
}

/** The readings that count: not corrected, not entered in error. */
export const counting = (readings: Reading[]) => readings.filter((r) => !r.supersededBy && !r.errorAt);

/** Oldest first, by when it was measured, then when it was entered: as times, since the API's strings differ in offset and fraction. */
export const byTime = (a: Reading, b: Reading) =>
  Date.parse(a.takenAt) - Date.parse(b.takenAt) || Date.parse(a.enteredAt) - Date.parse(b.enteredAt);

/** The newest reading of a kind that counts, by when it was measured (then when it was entered). */
export function latest(readings: Reading[], kind: string): Reading | undefined {
  return counting(readings).filter((r) => r.kind === kind).sort(byTime).at(-1);
}

/**
 * The hospital BMI (OBS-04): from the latest height and the latest weight that count on the episode,
 * with when the later of the two was measured. Undefined without both.
 */
export function hospitalBmi(readings: Reading[]): { bmi: number; at: string } | undefined {
  const h = latest(readings, "height");
  const w = latest(readings, "weight");
  const bmi = bmiOf(h?.value ?? undefined, w?.value ?? undefined);
  if (bmi === undefined || !h || !w) return undefined;
  return { bmi, at: Date.parse(h.takenAt) > Date.parse(w.takenAt) ? h.takenAt : w.takenAt };
}

/**
 * The measures whose value is outside its usual range (OBS-03's soft warning): the reading can still
 * be saved. None when any value is outside its hard limits: the server refuses that, with its reason.
 */
export function outsideUsual(ranges: ObservationRanges, measures: string[], values: (number | undefined)[]): string[] {
  const refused = measures.some((m, i) => {
    const v = values[i];
    const r = ranges[m];
    return r !== undefined && v !== undefined && Number.isFinite(v) && (v < r.hardMin || v > r.hardMax);
  });
  if (refused) return [];
  return measures.filter((m, i) => {
    const v = values[i];
    const r = ranges[m];
    return r !== undefined && v !== undefined && Number.isFinite(v) && (v < r.softMin || v > r.softMax);
  });
}
