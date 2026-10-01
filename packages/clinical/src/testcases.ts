// Test cases (PRV-05/06/07/08): a preview's answers saved with the outputs they should produce.
// They are stored in the chapter JSON under `testCases`, so they carry over to new versions,
// and forPatient() removes them before a patient gets the form.
import { Model } from "survey-core";
import { edit, type ChapterJson } from "./doc";
import { computeOutputs, type ComputedOutputs } from "./outputs";
import { forPatient, setViewer, type Viewer } from "./viewer";

/** Made-up patient details only (PRV-08). */
export type SamplePatient = { name: string; age?: number; sex?: string };

export type TestCase = {
  id: string;
  name: string;
  viewer: Viewer;
  patient: SamplePatient;
  answers: Record<string, unknown>;
  /** The outputs expected, as outputLines() writes them. */
  expected: string[];
};

export type TestResult = { pass: boolean; actual: string[]; missing: string[]; unexpected: string[] };

/** Outputs as plain, sorted lines: what test cases store and compare. */
export function outputLines(r: ComputedOutputs): string[] {
  const lines: string[] = [];
  for (const o of r.outputs) {
    for (const c of o.output.codes) lines.push(`Code ${c.set} ${c.code} ${c.display}`);
    if (o.output.note && o.noteText) lines.push(`Note (${o.output.note.category}): ${o.noteText}`);
    if (o.output.flag) lines.push(`Flag ${o.output.flag}: ${o.questionTitle}: ${o.answerLabel}`);
  }
  if (r.suggestedAsa) lines.push(`Suggested ASA ${r.suggestedAsa.grade}${r.suggestedAsa.emergency ? "E" : ""}`);
  return [...new Set(lines)].sort();
}

/** A survey as the viewer gets it, for the sample patient. */
export function modelFor(json: ChapterJson, viewer: Viewer, patient: SamplePatient): Model {
  const m = new Model(viewer === "patient" ? forPatient(json) : json);
  setViewer(m, viewer);
  m.setVariable("patientName", patient.name);
  m.setVariable("patientAge", patient.age);
  m.setVariable("patientSex", patient.sex);
  return m;
}

/** Runs one case against the chapter as it is now (PRV-06). */
export function runTestCase(json: ChapterJson, tc: TestCase): TestResult {
  const m = modelFor(json, tc.viewer, tc.patient);
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
