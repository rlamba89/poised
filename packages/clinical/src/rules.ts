// An HQ's ASA rules (plan C6, session S40; ASA-02, ASA-03, ASA-04). A rule has a name, a "when"
// (the suggested ASA compared with a grade, plus optional extras) and a "then" (a fixed list of
// actions). They're stored with the questionnaire version (Go's internal/asarules checks their
// shape and holds the platform's defaults); which fire is worked out here, from the same result the
// outputs and the suggested ASA give, so the Preview, the test cases and (from S41) the server agree.
//
// Rules look at what the patient sends: the patient's answers, as the clinician's view shows them
// before any validation. They run once, when the patient sends the HQ (S40 answers 3); S42 and S43
// act on them.
import { ASA_GRADES, asaRank, type SuggestedAsa } from "./asa";
import { allElements, optionsOf, textOf, type ChapterJson, type ElementJson } from "./doc";
import { bandsOf, type Band } from "./logic";
import { computeOutputs, type ProducedOutput, type ScoreResult } from "./outputs";
import { normalizeOutput, registerClinicalProperties } from "./properties";
import { hqResult, type Answers } from "./respondent";
import { modelFor, type SamplePatient, type TestResult } from "./testcases";
import type { Earlier } from "./scores";
import type { ClinicalOutput, CodeRef } from "./types";

export type AsaOp = ">=" | "<=" | "=";
export type TriageTarget = "ready_for_admission" | "ready_for_poa" | "poa_complete";
export type SuggestableStatus = TriageTarget | "on_hold" | "not_ready";
export type RuleTeam = "nurse" | "anaesthetist" | "admin";

/** Where auto-triage may move an episode (A-8), as Go's asarules.Targets; the labels are Go's episode.Labels. */
export const TRIAGE_TARGETS: { value: TriageTarget; label: string }[] = [
  { value: "ready_for_admission", label: "Ready for admission" },
  { value: "ready_for_poa", label: "Ready for POA" },
  { value: "poa_complete", label: "POA complete" },
];
/** The statuses a rule may suggest (after the HQ review, before admission), as Go's asarules.Suggestable. */
export const SUGGESTABLE: { value: SuggestableStatus; label: string }[] = [
  { value: "ready_for_poa", label: "Ready for POA" },
  { value: "poa_complete", label: "POA complete" },
  { value: "on_hold", label: "On hold" },
  { value: "not_ready", label: "Not ready for admission" },
  { value: "ready_for_admission", label: "Ready for admission" },
];
/** A task's teams, as Go's episode.Teams (S21). */
export const RULE_TEAMS: { value: RuleTeam; label: string }[] = [
  { value: "nurse", label: "Nurse" },
  { value: "anaesthetist", label: "Anaesthetist" },
  { value: "admin", label: "Admin" },
];

export type RuleWhen = {
  /** The suggested ASA compared with a grade, 1–6 (I–VI). */
  asa: { op: AsaOp; grade: number };
  /** No disclosure produced has a red review flag. */
  noRedFlags?: boolean;
  /** A score (a Calculation's key) is in one of these bands (their ids). */
  band?: { score: string; bands: string[] };
  /** A disclosure produced has this code. */
  code?: CodeRef;
  /** The episode's procedure is one of these (the trust's procedure ids). */
  procedures?: string[];
};

export type RuleAction =
  | { kind: "autoTriage"; target: TriageTarget }
  | { kind: "task"; title: string; team: RuleTeam }
  | { kind: "flag"; name: string }
  | { kind: "requireAnaesthetistAsa" }
  | { kind: "suggestStatus"; status: SuggestableStatus };

export type AsaRule = { id: string; name: string; when: RuleWhen; then: RuleAction[] };

/** A whole-HQ test case (S40 answers 4): answers by question key, and the lines expected. */
export type HqTestCase = { id: string; name: string; patient: SamplePatient; procedureId?: string; answers: Answers; expected: string[] };

/**
 * What a version stores: its rules in order, whether a patient whose answers give no ASA grade can
 * be auto-triaged (S40 answers 2: they count as ASA I; on by default), and its whole-HQ test cases.
 */
export type RuleSet = { noGradeAutoTriage: boolean; rules: AsaRule[]; testCases: HqTestCase[] };

/** What the rules look at: the result computeOutputs (one set) or hqResult (the HQ) gives, and the episode's procedure. */
export type RuleInput = { suggestedAsa?: SuggestedAsa; outputs: ProducedOutput[]; scores?: ScoreResult[]; procedureId?: string };

/**
 * A rule that fires, with the actions to take. `held`: the rule auto-triages, but the patient's
 * answers give no ASA grade and the HQ's setting says such patients aren't auto-triaged, so that
 * action was taken out.
 */
export type FiredRule = { rule: AsaRule; actions: RuleAction[]; held: boolean };

const sameCode = (a: CodeRef, b: CodeRef) => a.set === b.set && a.code === b.code;

function matches(w: RuleWhen, input: RuleInput, grade: number): boolean {
  const { op, grade: limit } = w.asa;
  if (op === ">=" ? grade < limit : op === "<=" ? grade > limit : grade !== limit) return false;
  if (w.noRedFlags && input.outputs.some((o) => o.output.flag === "red")) return false;
  if (w.band) {
    const s = (input.scores ?? []).find((x) => x.questionId === w.band!.score);
    if (!s?.band || !w.band.bands.includes(s.band.id)) return false;
  }
  if (w.code && !input.outputs.some((o) => o.output.codes.some((c) => sameCode(c, w.code!)))) return false;
  if (w.procedures && !(input.procedureId && w.procedures.includes(input.procedureId))) return false;
  return true;
}

/**
 * The rules that fire, in order, with their actions. A patient whose answers give no ASA grade
 * counts as ASA I (S40 answers 2); with the HQ's setting off, they're never auto-triaged.
 */
export function firedRules(set: RuleSet, input: RuleInput): FiredRule[] {
  const grade = input.suggestedAsa ? asaRank(input.suggestedAsa.grade) : 1;
  const hold = !input.suggestedAsa && !set.noGradeAutoTriage;
  return set.rules
    .filter((r) => matches(r.when, input, grade))
    .map((rule) => {
      const held = hold && rule.then.some((a) => a.kind === "autoTriage");
      return { rule, held, actions: held ? rule.then.filter((a) => a.kind !== "autoTriage") : rule.then };
    });
}

/**
 * The rules that fire for one Question Set's answers alone (the Preview), worked out as the rules
 * see them on send: in the clinician's view, so a clinician-only score from the patient's answers
 * counts whichever view the Preview shows (test report 1, F-1). `earlier`: the earlier sets'
 * answers given in the Preview (S38).
 */
export function setFiredRules(set: RuleSet, content: ChapterJson, answers: Answers, patient: SamplePatient, earlier?: Earlier): FiredRule[] {
  registerClinicalProperties();
  const m = modelFor(content, "clinician", patient, earlier);
  m.data = answers;
  const r = computeOutputs(m);
  m.dispose();
  return firedRules(set, { suggestedAsa: r.suggestedAsa, outputs: r.outputs, scores: r.scores });
}

// ---------------------------------------------------------------- in words

/** Names for what rules refer to by id: scores (with their bands) and procedures. */
export type RuleNames = { scores: Map<string, { title: string; bands: Map<string, string> }>; procedures: Map<string, string> };

export function ruleNames(docs: ChapterJson[], procedures: { id: string; name: string }[] = []): RuleNames {
  return {
    scores: new Map(scoresWithBands(docs).map((s) => [s.key, { title: s.title, bands: new Map(s.bands.map((b) => [b.id, b.label])) }])),
    procedures: new Map(procedures.map((p) => [p.id, p.name])),
  };
}

const roman = (n: number) => ASA_GRADES[n - 1] ?? String(n);
const or = (xs: string[]) => (xs.length <= 1 ? (xs[0] ?? "") : `${xs.slice(0, -1).join(", ")} or ${xs.at(-1)}`);
const label = (list: { value: string; label: string }[], v: string) => list.find((x) => x.value === v)?.label ?? v;

/** "is I or II", "is I to IV", "is III or more", "is III" (test report 1, E-1: no "at most"). */
function asaWords({ op, grade }: RuleWhen["asa"]): string {
  if (op === "=") return `is ${roman(grade)}`;
  if (op === "<=") return grade === 1 ? "is I" : grade === 2 ? "is I or II" : `is I to ${roman(grade)}`;
  return grade === 1 ? "is I to VI" : grade === 6 ? "is VI" : grade === 5 ? "is V or VI" : `is ${roman(grade)} or more`;
}

/** The "when" in words, without "When". */
export function whenWords(w: RuleWhen, names: RuleNames): string {
  const parts = [`the suggested ASA ${asaWords(w.asa)}`];
  if (w.noRedFlags) parts.push("there are no red flags");
  if (w.band) {
    const s = names.scores.get(w.band.score);
    parts.push(`${s?.title ?? w.band.score} is ${or(w.band.bands.map((b) => s?.bands.get(b) ?? b))}`);
  }
  if (w.code) parts.push(`a disclosure has the code ${w.code.display} (${w.code.code})`);
  if (w.procedures) parts.push(`the procedure is ${or(w.procedures.map((p) => names.procedures.get(p) ?? p))}`);
  return parts.join(" and ");
}

export function actionWords(a: RuleAction): string {
  switch (a.kind) {
    case "autoTriage":
      return `auto-triage to ${label(TRIAGE_TARGETS, a.target)}`;
    case "task":
      return `add the task “${a.title}” for the ${label(RULE_TEAMS, a.team).toLowerCase()} team`;
    case "flag":
      return `flag “${a.name}”`;
    case "requireAnaesthetistAsa":
      return "require the anaesthetist's ASA before Ready for admission";
    case "suggestStatus":
      return `suggest ${label(SUGGESTABLE, a.status)}`;
  }
}

/** "When the suggested ASA is I or II and there are no red flags: auto-triage to Ready for admission". */
export const ruleWords = (r: AsaRule, names: RuleNames): string => `When ${whenWords(r.when, names)}: ${r.then.map(actionWords).join("; ")}`;

/** A rule that fires, as test cases write it: "Rule fires: Low risk (auto-triage to Ready for admission)". */
export function firedLine(f: FiredRule): string {
  const words = f.actions.map(actionWords);
  if (f.held) words.push("no auto-triage: no ASA grade");
  return `Rule fires: ${f.rule.name} (${words.join("; ")})`;
}

/** "ASA I–II, no red flags": a rule's "when", short. */
function shortWhen(w: RuleWhen, names: RuleNames): string {
  const { op, grade } = w.asa;
  const asa = op === "=" ? `ASA ${roman(grade)}`
    : op === "<=" ? (grade === 1 ? "ASA I" : `ASA I–${roman(grade)}`)
    : grade === 6 ? "ASA VI" : `ASA ${roman(grade)}–VI`;
  const parts = [asa];
  if (w.noRedFlags) parts.push("no red flags");
  if (w.band) {
    const s = names.scores.get(w.band.score);
    parts.push(`${s?.title ?? w.band.score} ${w.band.bands.map((b) => s?.bands.get(b) ?? b).join("/")}`);
  }
  if (w.code) parts.push(`code ${w.code.code}`);
  if (w.procedures) parts.push(w.procedures.map((p) => names.procedures.get(p) ?? p).join("/"));
  return parts.join(", ");
}

/** The Publish window's line: what this version will do automatically. */
export function autoTriageSummary(set: RuleSet, names: RuleNames): string {
  const triage = set.rules.flatMap((r) => r.then.filter((a) => a.kind === "autoTriage").map((a) => `${shortWhen(r.when, names)} → ${label(TRIAGE_TARGETS, a.target)}`));
  if (!triage.length) return "This version doesn't auto-triage anyone.";
  const none = set.noGradeAutoTriage ? "Patients with no ASA grade count as ASA I." : "Patients with no ASA grade are never auto-triaged.";
  return `This version auto-triages: ${triage.join("; ")}. ${none}`;
}

// ---------------------------------------------------------------- what an HQ offers

function outputsIn(el: ElementJson): ClinicalOutput[] {
  const out: ClinicalOutput[] = [...(el.clinicalOutputs ?? [])];
  for (const c of optionsOf(el)) out.push(...(c.clinicalOutputs ?? []));
  for (const b of bandsOf(el)) out.push(...(b.clinicalOutputs ?? []));
  const cells = (el.cellOutputs ?? {}) as Record<string, Record<string, ClinicalOutput[]>>;
  for (const row of Object.values(cells)) for (const list of Object.values(row)) out.push(...list);
  return out.map(normalizeOutput);
}

/** Every code on a disclosure in these sets, each once, in order: what a rule's code can be. */
export function codesIn(docs: ChapterJson[]): CodeRef[] {
  const out: CodeRef[] = [];
  for (const doc of docs) {
    for (const { el } of allElements(doc)) {
      for (const o of outputsIn(el)) for (const c of o.codes) if (!out.some((x) => sameCode(x, c))) out.push(c);
    }
  }
  return out;
}

/** Every Calculation with bands in these sets: what a rule's score can be. */
export function scoresWithBands(docs: ChapterJson[]): { key: string; title: string; bands: Band[] }[] {
  return docs.flatMap((doc) =>
    allElements(doc)
      .filter(({ el }) => el.type === "expression" && bandsOf(el).length > 0)
      .map(({ el }) => ({ key: el.name, title: textOf(el.title) || el.name, bands: bandsOf(el) })),
  );
}

// ---------------------------------------------------------------- the whole HQ

/** A Question Set as the rules need it; `audience` "patient" (or none) is one the patient fills in. */
export type HqSet = { id: string; name: string; content: ChapterJson; audience?: string };

export const isPatientSet = (s: { audience?: string }) => !s.audience || s.audience === "patient";

/**
 * Answers by question key (a test case's) split into each set's own (keys are unique in an HQ,
 * S23). Only the patient's sets get answers: the rules look at what the patient sends.
 */
export function answersBySet(sets: HqSet[], answers: Answers): Record<string, Answers> {
  const out: Record<string, Answers> = {};
  for (const s of sets) {
    const names = new Set(allElements(s.content).map(({ el }) => el.name));
    out[s.id] = isPatientSet(s) ? Object.fromEntries(Object.entries(answers).filter(([k]) => names.has(k))) : {};
  }
  return out;
}

/**
 * A whole-HQ test case's lines: "Suggested ASA II" or "No suggested ASA" first, then each rule that
 * fires, in the rules' order (test report 2, E-7). Compared as a set, so the order is for reading only.
 */
export function hqLines(asa: SuggestedAsa | undefined, fired: FiredRule[]): string[] {
  return [...new Set([asa ? `Suggested ASA ${asa.grade}${asa.emergency ? "E" : ""}` : "No suggested ASA", ...fired.map(firedLine)])];
}

/**
 * What the patient's answers (by key) give across the HQ, as the clinician's view shows them before
 * any validation (S41 works out the same on send): every set (a clinician set has no answers, but
 * its Calculations may use the patient's, S38), with the clinician-only scores worked out from them
 * (e.g. STOP-Bang). It gives the sets shown, the suggested ASA, the rules that fire and the test
 * case lines.
 */
export function hqRuleResult(sets: HqSet[], set: RuleSet, answers: Answers, patient: SamplePatient, procedureId?: string) {
  const r = hqResult(sets, answersBySet(sets, answers), "clinician", patient);
  const fired = firedRules(set, { suggestedAsa: r.suggestedAsa, outputs: r.outputs, scores: r.scores, procedureId });
  return { shown: r.shown.map((s) => s.id), asa: r.suggestedAsa, fired, lines: hqLines(r.suggestedAsa, fired) };
}

/** Runs a whole-HQ test case against the sets and rules as they are now. */
export function runHqTestCase(sets: HqSet[], set: RuleSet, tc: HqTestCase): TestResult {
  const actual = hqRuleResult(sets, set, tc.answers, tc.patient, tc.procedureId).lines;
  const missing = tc.expected.filter((x) => !actual.includes(x));
  const unexpected = actual.filter((x) => !tc.expected.includes(x));
  return { pass: missing.length === 0 && unexpected.length === 0, actual, missing, unexpected };
}

/**
 * Rules that can't work as written (publish problems): a score or band, or a code, that the HQ no
 * longer has, so the rule can never fire.
 */
export function ruleProblems(set: RuleSet, sets: HqSet[]): string[] {
  const docs = sets.map((s) => s.content);
  const scores = scoresWithBands(docs);
  const codes = codesIn(docs);
  const out: string[] = [];
  for (const r of set.rules) {
    const { band, code } = r.when;
    if (band) {
      const s = scores.find((x) => x.key === band.score);
      if (!s) out.push(`Rule “${r.name}”: its score isn't in this HQ any more`);
      else if (band.bands.some((b) => !s.bands.some((x) => x.id === b))) out.push(`Rule “${r.name}”: a band it tests isn't in “${s.title}” any more`);
    }
    if (code && !codes.some((c) => sameCode(c, code))) out.push(`Rule “${r.name}”: no disclosure in this HQ has the code ${code.code}, so the rule can't fire`);
  }
  return out;
}

/** A rule set a version may hold (null before S40): the stored JSON, or none. */
export const ruleSetOf = (raw: unknown): RuleSet | null =>
  raw && typeof raw === "object" && Array.isArray((raw as RuleSet).rules)
    ? { noGradeAutoTriage: (raw as RuleSet).noGradeAutoTriage !== false, rules: (raw as RuleSet).rules, testCases: (raw as RuleSet).testCases ?? [] }
    : null;
