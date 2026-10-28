// The anaesthetist's ASA grade as a history (plan C6, session S44b; Rahul, 27 Oct). A grade, once
// given, is never changed or cleared: a later one is added on top (with a reason: every grade
// after the episode's first), and a mistake is marked "entered in error" with a reason. It stays in the list, struck
// through, and the one before it counts again. Go keeps the rules (asa_grades.go); this file only
// words them for the POA Summary.
import { formatDateTime } from "@/lib/format";

/** One grade (Go: ListAnaesthetistGrades), oldest first in GET …/episodes/:id's anaesthetistAsaGrades. */
export type AnaesthetistGrade = {
  id: string;
  grade: number;
  givenAt: string;
  /** Null for a grade carried over from before the history (migration 00035). */
  givenByName: string | null;
  /** Why a later grade was given; null for the first. */
  reason: string | null;
  /** Set once the grade is marked entered in error (it never comes off). */
  errorAt: string | null;
  errorByName: string | null;
  errorReason: string | null;
};

const ASA = ["I", "II", "III", "IV", "V", "VI"];
export const asaLabel = (n: number | null) => (n ? `ASA ${ASA[n - 1]}` : "Not recorded");

/** The grade that counts: the latest not entered in error (as Go's episode.anaesthetistAsa), or null. */
export function countingGrade(grades: AnaesthetistGrade[]): AnaesthetistGrade | null {
  return [...grades].reverse().find((g) => !g.errorAt) ?? null;
}

export const newestFirst = (grades: AnaesthetistGrade[]) => [...grades].reverse();

/** "Given by Cara Clinician on …", with the reason for a re-grade. */
export function givenLine(g: AnaesthetistGrade): string {
  const line = g.givenByName ? `Given by ${g.givenByName} on ${formatDateTime(g.givenAt)}` : `Recorded before the history (${formatDateTime(g.givenAt)})`;
  return g.reason ? `${line}: ${g.reason}` : line;
}

/** Who marked it entered in error, when and why; "" for a grade that isn't. */
export function errorLine(g: AnaesthetistGrade): string {
  if (!g.errorAt) return "";
  return `Entered in error: marked by ${g.errorByName ?? "someone"} on ${formatDateTime(g.errorAt)}: ${g.errorReason}`;
}

/** Every grade after the episode's first needs a reason, even after the earlier were entered in error (S44b answers 2). */
export const needsReason = (grades: AnaesthetistGrade[]) => grades.length > 0;

/** The Add grade form's request, or what's missing. `reasonNeeded`: needsReason of the history. */
export function addGradeRequest(grade: string | null, reason: string, reasonNeeded: boolean, revision: number):
  { error: string } | { body: { grade: number; reason: string; revision: number } } {
  if (!grade) return { error: "Choose an ASA grade." };
  if (reasonNeeded && !reason.trim()) return { error: "Say why the grade is changed." };
  return { body: { grade: Number(grade), reason: reason.trim(), revision } };
}
