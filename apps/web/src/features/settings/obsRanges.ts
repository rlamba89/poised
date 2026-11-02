// The observations' ranges setting (plan C7, session S46 answers 1): the soft range and hard limits
// per measure, which a trust's admins may change from Preopsis's defaults. Go checks them
// (internal/observation's CheckRanges); this only shapes the editor's value. The boxes keep exactly
// what's typed (a minus sign included: S46 test report 1, F-3); Save converts it.
import { parseNumber } from "@/features/episodes/observations";

export type Limit = "softMin" | "softMax" | "hardMin" | "hardMax";
export const LIMITS: Limit[] = ["softMin", "softMax", "hardMin", "hardMax"];
type Range = Record<Limit, number | string>;
export type RangesValue = Record<string, Range>;

const ordered = (r: Partial<Range>, fallback: Partial<Range>): Range =>
  Object.fromEntries(LIMITS.map((l) => [l, r[l] ?? fallback[l] ?? ""])) as Range;

/** The editor's value: every measure (missing ones from the defaults), in Go's order, so an unchanged table compares the same. */
export function rangesDraft(value: unknown, defaults: unknown): RangesValue {
  const own = (value ?? {}) as Record<string, Partial<Range>>;
  const base = (defaults ?? {}) as Record<string, Partial<Range>>;
  const keys = [...new Set([...Object.keys(base), ...Object.keys(own)])].sort();
  return Object.fromEntries(keys.map((k) => [k, ordered(own[k] ?? {}, base[k] ?? {})]));
}

/** The value with one limit changed. */
export const withLimit = (v: RangesValue, measure: string, limit: Limit, n: number | string): RangesValue => ({
  ...v,
  [measure]: { ...v[measure], [limit]: n },
});

/**
 * The value to save: each limit typed as a plain number becomes that number; anything else goes as
 * typed, for Go to refuse. Only the measures that differ from Preopsis's are sent, so a trust that
 * changes one keeps following the defaults for the rest (S46 verification).
 */
export function rangesForSave(v: RangesValue, defaults: unknown): RangesValue {
  const base = (defaults ?? {}) as Record<string, Partial<Range>>;
  const typed = Object.entries(v).map(([k, r]) => [k, Object.fromEntries(LIMITS.map((l) => {
    const x = r[l];
    if (typeof x === "number") return [l, x];
    const n = parseNumber(String(x ?? ""));
    return [l, Number.isFinite(n) ? n : x];
  })) as Range] as const);
  return Object.fromEntries(typed.filter(([k, r]) => LIMITS.some((l) => r[l] !== base[k]?.[l])));
}

/**
 * Whether the editor's value says the same as what's stored. Compared as Save would send each,
 * since the API returns the keys in the database's order: as text they never match, so an unchanged
 * table was sent, and a page left open undid another admin's limit (S46 verification 2, F-1).
 */
export function sameRanges(value: RangesValue, stored: unknown, defaults: unknown): boolean {
  const asSent = (v: RangesValue) => JSON.stringify(rangesForSave(v, defaults));
  return asSent(value) === asSent(rangesDraft(stored, defaults));
}
