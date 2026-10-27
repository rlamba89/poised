// The evaluate service's two operations (plan C6, session S41; D-1). Go never evaluates SurveyJS
// (CLAUDE.md's key rule); this is the one agreed exception: when an HQ is sent, Go sends the
// published version, the patient's answers and the patient context here, and stores what comes back.
// When a version is published, Go asks here for the browser's publish checks, so the API alone can't
// publish a version whose test cases fail.
//
// Both are pure functions of their input: no database, no state, nothing logged. apps/evaluate
// serves them over HTTP (npm run evaluate); S45 runs the same functions in a Lambda. The shared
// fixtures in apps/api/internal/evaluate/testdata keep this file and Go's client in step.
import type { SuggestedAsa } from "./asa";
import type { ChapterJson } from "./doc";
import type { ProducedOutput, ScoreResult } from "./outputs";
import { publishProblems, type PublishProblem } from "./publish";
import { hqResult, hqSets, isSetComplete, type Answers } from "./respondent";
import { firedLine, firedRules, isPatientSet, ruleSetOf, type RuleAction } from "./rules";
import type { SamplePatient } from "./testcases";
import type { ReviewFlag } from "./types";

/** A Question Set of the published version, as stored (Go sends it whole, clinician-only parts too). */
export type EvaluateSet = { id: string; name: string; audience: string; content: ChapterJson };

/** What Go sends when an HQ is sent. */
export type EvaluateInput = {
  /** The episode form's id: the only thing an error log may name. */
  formId?: string;
  /** Every Question Set of the published version, in order. */
  sets: EvaluateSet[];
  /** The version's ASA rules as stored (S40); null for a version published before rules existed. */
  rules: unknown;
  /** The patient's answers, by set id. Only the patient's sets count. */
  answers: Record<string, Answers>;
  /** The patient's age on the day they sent their answers, and sex. No name or other details. */
  patient: { age?: number; sex?: string };
  /** The episode's procedure (a trust procedure id), if it has one. */
  procedureId?: string | null;
};

/** A disclosure with a review flag, in words ("Do you get chest pain?: Yes"). */
export type EvaluateFlag = { level: ReviewFlag; setId: string; questionId: string; text: string };

/** A rule that fires: its actions (auto-triage taken out when `held`), and the line test cases write. */
export type EvaluateRule = { id: string; name: string; actions: RuleAction[]; held: boolean; line: string };

/** What comes back: the same result the clinician's view shows for the patient's answers, before any validation. */
export type EvaluateResult = {
  /** Which code worked it out (apps/evaluate's build). */
  evaluator: string;
  /** The Question Sets shown (the clinician's view), in order. */
  shown: string[];
  /** Each shown set's disclosures, and its scores with their bands. */
  sets: { id: string; outputs: ProducedOutput[]; scores: ScoreResult[] }[];
  /** The suggested ASA with its reasons (S39), or null for none. */
  suggestedAsa: SuggestedAsa | null;
  /** Every red and amber flag, in order. */
  flags: EvaluateFlag[];
  /** The rules that fire, in order (S40). */
  rules: EvaluateRule[];
  /** The sets the patient's own form shows, and whether each is complete (S44 refuses an incomplete send). */
  patientSets: { id: string; complete: boolean }[];
};

/** An input that isn't what Go sends: the service answers 400. Its message never holds an answer. */
export class EvaluateInputError extends Error {}

const isObject = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null && !Array.isArray(v);

function checkSets(sets: unknown, needId: boolean): void {
  if (!Array.isArray(sets)) throw new EvaluateInputError("sets must be a list");
  for (const s of sets) {
    if (!isObject(s) || (needId && typeof s.id !== "string") || typeof s.name !== "string" || typeof s.audience !== "string" || !isObject(s.content)) {
      throw new EvaluateInputError("each set needs its id, name, audience and content");
    }
  }
}

function checkInput(input: EvaluateInput): void {
  if (!isObject(input)) throw new EvaluateInputError("the input must be an object");
  checkSets(input.sets, true);
  if (!isObject(input.answers) || !Object.values(input.answers).every(isObject)) throw new EvaluateInputError("answers must be an object of objects, by set id");
  const p = input.patient;
  if (!isObject(p) || (p.age !== undefined && typeof p.age !== "number") || (p.sex !== undefined && typeof p.sex !== "string")) {
    throw new EvaluateInputError("patient must give a number age and a text sex");
  }
  if (input.procedureId != null && typeof input.procedureId !== "string") throw new EvaluateInputError("procedureId must be text");
}

/**
 * What a sent HQ's answers give, worked out as the clinician's view does before any validation
 * (rules.ts hqRuleResult, S40): every set, with answers only in the patient's sets, so a clinician
 * score worked out from them (STOP-Bang) counts and a clinician set's own questions never do. Then,
 * in the patient's own view, which of their sets are shown and complete.
 */
export function evaluate(input: EvaluateInput, version: string): EvaluateResult {
  checkInput(input);
  const patient: SamplePatient = { name: "", age: input.patient.age, sex: input.patient.sex };
  const answers: Record<string, Answers> = {};
  for (const s of input.sets) if (isPatientSet(s)) answers[s.id] = input.answers[s.id] ?? {};

  const r = hqResult(input.sets, answers, "clinician", patient);
  const rules = ruleSetOf(input.rules);
  const fired = rules
    ? firedRules(rules, { suggestedAsa: r.suggestedAsa, outputs: r.outputs, scores: r.scores, procedureId: input.procedureId ?? undefined })
    : [];
  const flags: EvaluateFlag[] = r.outputs
    .filter((o) => o.output.flag)
    .map((o) => ({ level: o.output.flag!, setId: o.setId, questionId: o.questionId, text: `${o.questionTitle}: ${o.answerLabel}` }));

  const own = input.sets.filter(isPatientSet);
  const view = hqSets(own, answers, "patient", patient);
  const patientSets = view.shown.map((s) => ({ id: s.id, complete: isSetComplete(s.content, answers[s.id], "patient", patient, view.earlier.get(s.id)) }));

  return {
    evaluator: version,
    shown: r.shown.map((s) => s.id),
    sets: r.sets,
    suggestedAsa: r.suggestedAsa ?? null,
    flags,
    rules: fired.map((f) => ({ id: f.rule.id, name: f.rule.name, actions: f.actions, held: f.held, line: firedLine(f) })),
    patientSets,
  };
}

/** What Go sends when a version is published: its Question Sets in order (with their audience) and its rules. */
export type PublishInput = { sets: Omit<EvaluateSet, "id">[]; rules: unknown };

/** The browser's publish checks (publish.ts), run on the server: each set's audience counts (S39). */
export function checkPublish(input: PublishInput): { problems: PublishProblem[] } {
  if (!isObject(input)) throw new EvaluateInputError("the input must be an object");
  checkSets(input.sets, false);
  const sets = input.sets.map((s) => ({ name: s.name, doc: s.content, audience: s.audience }));
  return { problems: publishProblems(sets, ruleSetOf(input.rules)) };
}
