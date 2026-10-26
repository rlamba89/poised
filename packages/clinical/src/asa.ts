// The ASA grade (S39, ASA-01, CLN-17): the ASA guide's choices in the editor, and the suggested ASA
// for one Question Set (the preview) or the whole HQ (respondent.ts hqSuggestedAsa: the POA Summary,
// the episode page), with its reasons. The suggestion is only that: the nurse and the anaesthetist give the real grade.
// Only types are imported, so outputs.ts and respondent.ts can both use it.
import type { ProducedOutput } from "./outputs";
import type { AsaGrade, ClinicalOutput } from "./types";

export const ASA_GRADES: AsaGrade[] = ["I", "II", "III", "IV", "V", "VI"];

/** A grade's number: I is 1 … VI is 6. */
export const asaRank = (grade: AsaGrade): number => ASA_GRADES.indexOf(grade) + 1;

/**
 * One example from the ASA guide (Go's internal/asaguide), as GET /api/codes sends it with each
 * code: the grade, the ASA's own words, and whose example it is (Adults, Children or Pregnancy).
 */
export type AsaChoice = { grade: AsaGrade; example: string; group: string };

type Asa = NonNullable<ClinicalOutput["asa"]>;

const sameChoice = (a: AsaChoice, b: AsaChoice) => a.grade === b.grade && a.example === b.example && a.group === b.group;

/**
 * After a code is added to a disclosure in the editor (S39 answers 1): `guide` is every example the
 * disclosure's added codes offer, each once. A disclosure that has no grade yet takes the only
 * example's grade, with its words as the reason; when there are several (the grade depends on
 * severity or control), nothing is filled in and the author must choose (`mustChoose`). A grade the
 * disclosure already has is kept. Once the author has chosen "No ASA grade" (`declined`), adding a
 * code fills in nothing and asks nothing: its examples join the box, and No ASA grade stands
 * (verification 1, F1: a declined grade never comes back by itself).
 */
export function afterCodeAdded(
  asa: Asa | undefined, guide: AsaChoice[], added: AsaChoice[], declined = false,
): { asa: Asa | undefined; guide: AsaChoice[]; mustChoose: boolean } {
  const all = [...guide];
  for (const c of added) if (!all.some((x) => sameChoice(x, c))) all.push(c);
  if (asa || declined || all.length === 0) return { asa, guide: all, mustChoose: false };
  if (all.length === 1) return { asa: { grade: all[0].grade, emergency: false, reason: all[0].example }, guide: all, mustChoose: false };
  return { asa: undefined, guide: all, mustChoose: true };
}

/**
 * After a code is removed from a disclosure in the editor (test report 1, F1): a grade taken from
 * one of that code's examples (`removed`) goes with it, unless a code left still offers the same
 * example; the codes left (`left`, their examples) then decide as when adding: one example fills
 * in, several must be chosen; the Emergency (E) tick stays on the grade that replaces it
 * (verification 1, F2). Any other grade (the author's own, or another code's) is kept.
 */
export function afterCodeRemoved(asa: Asa | undefined, removed: AsaChoice[], left: AsaChoice[]): { asa: Asa | undefined; mustChoose: boolean } {
  const from = (list: AsaChoice[]) => !!asa?.reason && list.some((c) => c.grade === asa.grade && c.example === asa.reason);
  if (!asa || !from(removed) || from(left)) return { asa, mustChoose: false };
  const next = afterCodeAdded(undefined, [], left);
  return { asa: next.asa && { ...next.asa, emergency: asa.emergency }, mustChoose: next.mustChoose };
}

/**
 * The disclosure's ASA when the author picks a grade (from the guide or the grade list): the guide's
 * words go with it when exactly one example has that grade; otherwise it's the author's own grade,
 * with no reason. No grade: none.
 */
export function gradePicked(grade: AsaGrade | null, emergency: boolean, guide: AsaChoice[]): Asa | undefined {
  if (!grade) return undefined;
  const at = guide.filter((c) => c.grade === grade);
  return at.length === 1 ? { grade, emergency, reason: at[0].example } : { grade, emergency };
}

/** One reason for the suggested ASA: a disclosure in words, and where it came from (its set and question). */
export type AsaReason = { text: string; questionId: string; setId?: string };

/** The suggested ASA: the highest grade, E when any disclosure counted is an emergency, and why. */
export type SuggestedAsa = { grade: AsaGrade; emergency: boolean; reasons: AsaReason[] };

/**
 * A disclosure in words: its codes ("Heart failure unspecified (I50.9)"), else its note, else the
 * question and answer; with the ASA guide's example when it has one ("– Moderate reduction of
 * ejection fraction").
 */
function words(o: ProducedOutput): string {
  const what = o.output.codes.length
    ? o.output.codes.map((c) => `${c.display} (${c.code})`).join(", ")
    : o.noteText || `${o.questionTitle}: ${o.answerLabel}`;
  return o.output.asa?.reason ? `${what} – ${o.output.asa.reason}` : what;
}

/**
 * The suggested ASA from the disclosures produced (CLN-17): the highest grade, with every disclosure
 * at that grade as a reason (each once). E (emergency) is kept from any disclosure that has it,
 * whatever its grade, and one below the top grade is listed with "(emergency)". None without a grade.
 */
export function suggestAsa(outputs: (ProducedOutput & { setId?: string })[]): SuggestedAsa | undefined {
  const graded = outputs.filter((o) => o.output.asa && asaRank(o.output.asa.grade) > 0);
  if (!graded.length) return undefined;
  const top = Math.max(...graded.map((o) => asaRank(o.output.asa!.grade)));
  const reasons: AsaReason[] = [];
  const add = (o: ProducedOutput & { setId?: string }, text: string) => {
    if (!reasons.some((r) => r.text === text)) reasons.push({ text, questionId: o.questionId, ...(o.setId ? { setId: o.setId } : {}) });
  };
  for (const o of graded) if (asaRank(o.output.asa!.grade) === top) add(o, words(o));
  for (const o of graded) if (o.output.asa!.emergency && asaRank(o.output.asa!.grade) < top) add(o, `${words(o)} (emergency)`);
  return { grade: ASA_GRADES[top - 1], emergency: graded.some((o) => o.output.asa!.emergency), reasons };
}

/** "Suggested ASA III: Heart failure (I50); Angina", "Suggested ASA IIE: …", or "No suggested ASA". */
export function suggestedAsaText(s: SuggestedAsa | undefined): string {
  if (!s) return "No suggested ASA";
  return `${asaHeading(s)}: ${s.reasons.map((r) => r.text).join("; ")}`;
}

/** "Suggested ASA III", or "Suggested ASA IIIE" (E: emergency), as test cases write it. */
export const asaHeading = (s: SuggestedAsa): string => `Suggested ASA ${s.grade}${s.emergency ? "E" : ""}`;
