// Test cases (PRV-05/06/07/08): a preview's answers saved with the outputs they should produce.
// They are stored in the chapter JSON under `testCases`, so they carry over to new versions,
// and forPatient() removes them before a patient gets the form.
import { Model } from "survey-core";
import { edit, type ChapterJson } from "./doc";
import { computeOutputs, type ComputedOutputs } from "./outputs";
import { forPatient, setViewer, type Viewer } from "./viewer";
import { withoutSurveyHtml } from "./information";
import { applyEarlier, type Earlier } from "./scores";
import { givenEarlier } from "./questionSets";

/** Made-up patient details only (PRV-08). */
export type SamplePatient = { name: string; age?: number; sex?: string };

export type TestCase = {
  id: string;
  name: string;
  viewer: Viewer;
  patient: SamplePatient;
  answers: Record<string, unknown>;
  /**
   * Answers to earlier Question Sets' questions, by key (S38), for a score that uses them. The preview
   * can't see earlier sets, so the author gives them; without them they count as unanswered.
   */
  earlier?: Record<string, unknown>;
  /** The outputs expected, as outputLines() writes them. */
  expected: string[];
};

export type TestResult = { pass: boolean; actual: string[]; missing: string[]; unexpected: string[] };

/** Outputs as plain, sorted lines: what test cases store and compare. */
export function outputLines(r: ComputedOutputs): string[] {
  const lines: string[] = [];
  for (const o of r.outputs) {
    for (const c of o.output.codes) lines.push(`Code ${c.set} ${c.code} ${c.display}`);
    // A note saved without a category has no brackets (S08 test report 3).
    if (o.output.note && o.noteText) lines.push(o.output.note.category ? `Note (${o.output.note.category}): ${o.noteText}` : `Note: ${o.noteText}`);
    if (o.output.flag) lines.push(`Flag ${o.output.flag}: ${o.questionTitle}: ${o.answerLabel}`);
  }
  if (r.suggestedAsa) lines.push(`Suggested ASA ${r.suggestedAsa.grade}${r.suggestedAsa.emergency ? "E" : ""}`);
  return [...new Set(lines)].sort();
}

/**
 * A survey as the viewer gets it, for the sample patient. `earlier` gives it the earlier Question
 * Sets' answers, for scores that use them (S38; respondent.ts earlierContexts builds it for an HQ).
 */
export function modelFor(json: ChapterJson, viewer: Viewer, patient: SamplePatient, earlier?: Earlier): Model {
  const clean = withoutSurveyHtml(json);
  const m = new Model(viewer === "patient" ? forPatient(clean) : clean);
  setViewer(m, viewer);
  m.setVariable("patientName", patient.name);
  m.setVariable("patientAge", patient.age);
  m.setVariable("patientSex", patient.sex);
  applyEarlier(m, earlier);
  return m;
}

/**
 * Runs one case against the chapter as it is now (PRV-06). `earlier` is the earlier Question Sets,
 * for the points and bands of the earlier answers the case gives (S38); a set the case's patient
 * wouldn't see gives none (givenEarlier).
 */
export function runTestCase(json: ChapterJson, tc: TestCase, earlier: { name: string; doc: ChapterJson }[] = []): TestResult {
  const m = modelFor(json, tc.viewer, tc.patient, tc.earlier ? givenEarlier(earlier, tc.earlier, tc.patient, tc.viewer).earlier : undefined);
  m.data = tc.answers;
  const actual = outputLines(computeOutputs(m));
  const missing = tc.expected.filter((x) => !actual.includes(x));
  const unexpected = actual.filter((x) => !tc.expected.includes(x));
  return { pass: missing.length === 0 && unexpected.length === 0, actual, missing, unexpected };
}

export const testCasesOf = (doc: ChapterJson): TestCase[] => (Array.isArray(doc.testCases) ? (doc.testCases as TestCase[]) : []);

export function setTestCases(doc: ChapterJson, cases: TestCase[]): ChapterJson {
  return edit(doc, (d) => {
    if (cases.length) d.testCases = cases;
    else delete d.testCases;
  });
}
