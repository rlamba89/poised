// The New episode page's rules that don't need React (S13), unit-tested in newEpisode.test.ts.
import { ApiError } from "@/lib/api";
import type { FoundPatient } from "./types";

/**
 * The search's query string, or null while the term is too broad to send. It mirrors the API's
 * rule (patient.ParseSearch in Go): at least 3 characters, or 1 with a date of birth.
 */
export function searchQuery(term: string, dob: string): string | null {
  const q = term.trim();
  const n = [...q].length;
  if (n === 0 || (n < 3 && !dob)) return null;
  const params = new URLSearchParams({ q });
  if (dob) params.set("dob", dob);
  return params.toString();
}

/**
 * Why creating an episode for typed details was refused with a 409 the page handles itself:
 * - "mismatch": the NHS number is the trust's, but the date of birth or surname differ;
 * - "possible_duplicate": the same name and date of birth. `matches` are this hospital's own
 *   patients (to use instead); a patient known only elsewhere is never shown, so it may be empty.
 */
export type Conflict = { code: "mismatch" | "possible_duplicate"; message: string; matches: FoundPatient[] };

export function conflictOf(e: unknown): Conflict | null {
  if (!(e instanceof ApiError) || e.status !== 409) return null;
  const body = e.body as { code?: string; matches?: FoundPatient[] } | null;
  if (body?.code !== "mismatch" && body?.code !== "possible_duplicate") return null;
  return { code: body.code, message: e.message, matches: body.matches ?? [] };
}

/**
 * The Episode card's hospital number once a patient is chosen: the patient's number here, unless
 * staff already typed one. (Change patient clears it: it belonged to that patient.)
 */
export function hospitalNumberOnChoose(typed: string, patients: string): string {
  return typed || patients;
}

/** A patient's newest hospital number here, from their episodes here (newest first), or "". */
export function latestHospitalNumber(episodes: { hospitalNumber: string }[]): string {
  return episodes.find((e) => e.hospitalNumber)?.hospitalNumber ?? "";
}
