// Filling in an episode's HQ (plan-workflow.md Steps 3–4): the patient on their link, and the
// clinician validating. The same rules decide which Question Sets are shown and when one is done.
import type { ChapterJson } from "./doc";
import { registerClinicalProperties } from "./properties";
import { isChapterShown } from "./questionSets";
import { modelFor, type SamplePatient } from "./testcases";
import type { Viewer } from "./viewer";

/** The episode's patient as the API returns them. Made-up details only (NFR-03). */
export type EpisodePatient = { firstName: string; lastName: string; dateOfBirth: string; sex: string };

export type Answers = Record<string, unknown>;

/** Whole years from a YYYY-MM-DD date of birth to `today`. */
export function ageFrom(ymd: string, today = new Date()): number {
  const [y, m, d] = ymd.split("-").map(Number);
  const age = today.getFullYear() - y;
  return today.getMonth() + 1 < m || (today.getMonth() + 1 === m && today.getDate() < d) ? age - 1 : age;
}

/** The survey variables {patientName}, {patientAge} and {patientSex} for this patient. */
export function patientVariables(p: EpisodePatient, today = new Date()): SamplePatient {
  return { name: `${p.firstName} ${p.lastName}`, age: ageFrom(p.dateOfBirth, today), sex: p.sex };
}

/** Whether every required question the viewer can see is answered, without showing errors. */
export function isSetComplete(json: ChapterJson, data: Answers | undefined, viewer: Viewer, patient: SamplePatient): boolean {
  registerClinicalProperties();
  const m = modelFor(json, viewer, patient);
  m.data = data ?? {};
  const ok = m.validate(false);
  m.dispose();
  return ok;
}

/**
 * The Question Sets shown, in order (LOG-02). A set's condition sees the answers of the sets
 * shown before it, and the patient's age and sex.
 */
export function shownSets<T extends { id: string; content: ChapterJson }>(
  sets: T[], answers: Record<string, Answers | undefined>, viewer: Viewer, patient: SamplePatient,
): T[] {
  let values: Answers = { patientAge: patient.age, patientSex: patient.sex, viewer };
  return sets.filter((s) => {
    if (!isChapterShown(s.content, values)) return false;
    values = { ...values, ...answers[s.id] };
    return true;
  });
}
