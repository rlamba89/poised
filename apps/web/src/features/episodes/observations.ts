// The Observations panel's small rules (plan C7, session S46; OBS-01–03): how a reading reads, its
// history lines, the soft warning, and the form turned into a request. Go checks the hard limits
// (internal/observation); which readings count and the hospital BMI are @poised/clinical's.
import { byTime, hospitalBmi, outsideUsual, type ObservationRanges, type Reading } from "@poised/clinical";
import { observationUnit } from "@/lib/config";
import { formatDateTime } from "@/lib/format";
import { fromLocalInput, toLocalInput } from "@/lib/localTime";

export type ObservationKind = { key: string; label: string; measures: string[]; note: boolean };
export type ObservationMeasure = { key: string; label: string; unit: string; decimals: number };

/** GET …/observations. */
export type ObservationsView = {
  readings: Reading[];
  kinds: ObservationKind[];
  measures: ObservationMeasure[];
  ranges: ObservationRanges;
  maxNote: number;
};

const measureOf = (view: ObservationsView, key: string) => view.measures.find((m) => m.key === key);
export const kindOf = (view: ObservationsView, key: string) => view.kinds.find((k) => k.key === key);

/** The unit a measure is shown in: the country's for glucose and temperature (GET /api/config, A-20), else the stored one. */
export const unitOf = (measure: string, stored: string) => observationUnit(measure) ?? stored;

/** "82.5 kg", "128/82 mmHg", or an ECG's note. */
export function valueText(r: Reading, view: ObservationsView): string {
  if (r.value === null) return r.note ?? "";
  const kind = kindOf(view, r.kind);
  const keys = kind?.measures ?? [r.kind];
  const fixed = (v: number, key: string) => {
    const m = measureOf(view, key);
    return m ? v.toFixed(m.decimals) : String(v);
  };
  const first = measureOf(view, keys[0]);
  const unit = unitOf(keys[0], first?.unit ?? r.unit ?? "");
  const shown = r.value2 !== null && keys[1] ? `${fixed(r.value, keys[0])}/${fixed(r.value2, keys[1])}` : fixed(r.value, keys[0]);
  return unit ? `${shown} ${unit}` : shown;
}

/** The hospital BMI line (OBS-04). */
export function hospitalBmiText(readings: Reading[]): string {
  const b = hospitalBmi(readings);
  return b ? `${b.bmi} (measured ${formatDateTime(b.at)})` : "Not measured yet (needs a height and a weight)";
}

/**
 * The POA Summary's Hospital BMI row, or null for no row (observations off, or still loading). A
 * failed load says so: a BMI that exists but didn't load mustn't look like observations being off.
 */
export function hospitalBmiRow(view: ObservationsView | "off" | null, error: string): string | null {
  if (view === "off") return null;
  if (view) return hospitalBmiText(view.readings);
  return error ? "Couldn't be loaded" : null;
}

/** Who entered it and when it was measured; a correction's reason. */
export const historyLine = (r: Reading) =>
  `Measured ${formatDateTime(r.takenAt)} · ${r.corrects ? `corrected by ${r.enteredByName}: ${r.reason}` : `entered by ${r.enteredByName}`}`;

/** For a reading entered in error: who marked it, when and why. */
export const errorLine = (r: Reading) => `Entered in error: marked by ${r.errorByName} on ${formatDateTime(r.errorAt ?? "")}: ${r.errorReason}`;

/** Newest measured first (then newest entered). */
export const newestFirst = (rs: Reading[]) => [...rs].sort((a, b) => byTime(b, a));

/** OBS-03's soft warning for values about to be saved, or "" inside the usual range. Names the first value outside it. */
export function warningText(view: ObservationsView, kind: string, values: (number | undefined)[]): string {
  const k = kindOf(view, kind);
  if (!k || k.note || !k.measures) return "";
  const out = outsideUsual(view.ranges, k.measures, values)[0];
  if (!out) return "";
  const m = measureOf(view, out);
  const r = view.ranges[out];
  const v = values[k.measures.indexOf(out)];
  return `${m?.label ?? out} ${v} ${unitOf(out, m?.unit ?? "")} is outside the usual range (${r.softMin} to ${r.softMax}).`;
}

/** A number typed in a box ("82.5", "-1" or "82,5"), or NaN: only plain decimals, never 1e2 or 0x10. */
export function parseNumber(s: string): number {
  const t = s.trim().replace(",", ".");
  return /^-?(\d+\.?\d*|\.\d+)$/.test(t) ? Number(t) : Number.NaN;
}

type Values = { value?: number; value2?: number; note?: string };

/** The values from the form, or what's missing. */
function valuesOf(view: ObservationsView, kind: string, inputs: string[], note: string): { values: Values } | { error: string } {
  const k = kindOf(view, kind);
  if (!k) return { error: "Choose what was measured." };
  if (k.note) {
    const text = note.trim();
    if (!text) return { error: `Enter the ${k.label} note.` };
    // By characters, as Go counts them. Said here, as the box no longer cuts a paste off (test report 4).
    if ([...text].length > view.maxNote) return { error: `The note is too long (${view.maxNote} characters at most).` };
    return { values: { note: text } };
  }
  const out: number[] = [];
  for (const [i, key] of k.measures.entries()) {
    const m = measureOf(view, key);
    const label = m?.label ?? key;
    const s = inputs[i] ?? "";
    if (!s.trim()) return { error: `Enter the ${label.toLowerCase()}.` };
    const v = parseNumber(s);
    if (!Number.isFinite(v)) return { error: `${label}: enter a number.` };
    out.push(v);
  }
  if (out.length === 2) {
    // Go refuses it too; said here first, so no "outside the usual range" warning comes before it (F-2).
    if (out[1] >= out[0]) return { error: "The diastolic must be lower than the systolic." };
    return { values: { value: out[0], value2: out[1] } };
  }
  return { values: { value: out[0] } };
}

/** POST …/observations's body, or what to fix first. takenAtLocal is the date-time box's value. */
export function readingRequest(view: ObservationsView, kind: string, inputs: string[], note: string, takenAtLocal: string):
  { body: Values & { kind: string; takenAt: string } } | { error: string } {
  const v = valuesOf(view, kind, inputs, note);
  if ("error" in v) return v;
  const takenAt = fromLocalInput(takenAtLocal);
  if (!takenAt) return { error: "When was it measured? Enter the date and time." };
  return { body: { kind, ...v.values, takenAt } };
}

/** POST …/observations/{id}/correct's body (the kind is the reading's; its time stays), or what to fix. */
export function correctionRequest(view: ObservationsView, kind: string, inputs: string[], note: string, reason: string):
  { body: Values & { reason: string } } | { error: string } {
  const v = valuesOf(view, kind, inputs, note);
  if ("error" in v) return v;
  if (!reason.trim()) return { error: "Say why the reading is corrected." };
  return { body: { ...v.values, reason: reason.trim() } };
}

/** The Measured at time to send: what the clinician typed, else the time of Save (not when the form opened). */
export const measuredAt = (touched: boolean, typed: string, now: Date) => (touched ? typed : toLocalInput(now));
