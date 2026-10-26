// Ready-made scores (S38): a validated score inserted whole, with its published wording, points and
// bands, which authors can't change. The first is STOP-Bang (Rahul, 26 Oct); the others (Apfel and
// the rest) come with plan C7. Built from the same parts as an author's score (scores.ts): yes/no
// questions with points, "Add points when…" lines for age, sex and BMI, and bands, one with an
// "also when" condition. Its published scoring is checked by ready.test.ts.
//
// Only the Group's condition and the disclosures (on its questions' options and its bands) may
// change: they're the trust's response to an answer or a result, not the score. Anything else is a
// publish problem (readyScoreProblems).
import { allElements, edit, findElement, pagesOf, textOf, type ChapterJson, type ElementJson, type Target } from "./doc";
import { formatScore, type ScoreForm } from "./scores";

export type ReadyScore = {
  id: string;
  name: string;
  /** Which published version, and where it comes from. */
  source: string;
  /** Who owns it, and the licence its use needs. */
  licence: string;
  /** What the author must choose when inserting it. */
  needsBmi: boolean;
};

export const STOP_BANG: ReadyScore = {
  id: "stopbang",
  name: "STOP-Bang",
  source:
    "STOP-Bang questionnaire, updated version (neck: shirt collar 17 inches / 43 cm for men, 16 inches / 41 cm for women). " +
    "Chung F, Abdullah HR, Liao P. STOP-Bang Questionnaire: a practical approach to screen for obstructive sleep apnea. Chest 2016;149(3):631–8.",
  licence: "© Dr Frances Chung and University Health Network (stopbang.ca). Use beyond education and research needs a licence from UHN.",
  needsBmi: true,
};

export const READY_SCORES: ReadyScore[] = [STOP_BANG];

const KEY = {
  group: "stopbang",
  snoring: "stopbang_snoring",
  tired: "stopbang_tired",
  observed: "stopbang_observed",
  pressure: "stopbang_pressure",
  neck: "stopbang_neck",
  score: "stopbang_score",
};
/** The STOP-Bang questions' keys, in order, for checks elsewhere. */
export const STOP_BANG_KEYS = KEY;

const YES = "o_yes";
const yesNo = (name: string, title: string): ElementJson => ({
  type: "radiogroup", name, title, yesNo: true, colCount: 0, isRequired: true,
  choices: [{ value: YES, text: "Yes", score: 1 }, { value: "o_no", text: "No", score: 0 }],
});

/**
 * STOP-Bang as a Group, scored with the patient's age and sex and the BMI from `bmi` (a BMI
 * question's `<key>.bmi`, or a BMI Calculation's key, in this Question Set or an earlier one).
 * - Each Yes is 1 point; over 50, male, and a BMI over 35 are 1 point each (0–8).
 * - Low risk 0–2, Intermediate 3–4, High 5–8; also High with 2 or more of the 4 STOP questions and
 *   male sex, a BMI over 35, or a large neck.
 * - Incomplete while any of the eight has no answer.
 */
export function stopBangGroup(bmi: string): ElementJson {
  const stop = [KEY.snoring, KEY.tired, KEY.observed, KEY.pressure];
  const form: ScoreForm = {
    kind: "score",
    questions: [...stop, KEY.neck],
    lines: [
      { points: 1, rule: { left: bmi, op: "gt", value: 35 } },
      { points: 1, rule: { left: "patientAge", op: "gt", value: 50 } },
      { points: 1, rule: { left: "patientSex", op: "eq", value: "male" } },
    ],
  };
  const stopPart = `score(${stop.map((k) => `'${k}'`).join(", ")}) >= 2`;
  return {
    type: "panel",
    name: KEY.group,
    title: "STOP-Bang",
    readyScore: STOP_BANG.id,
    readyScoreBmi: bmi,
    elements: [
      yesNo(KEY.snoring, "Do you snore loudly (loud enough to be heard through closed doors, or your bed partner elbows you for snoring at night)?"),
      yesNo(KEY.tired, "Do you often feel tired, fatigued or sleepy during the daytime (such as falling asleep while driving or talking to someone)?"),
      yesNo(KEY.observed, "Has anyone observed you stop breathing, or choking or gasping, during your sleep?"),
      yesNo(KEY.pressure, "Do you have, or are you being treated for, high blood pressure?"),
      yesNo(KEY.neck, "Is your neck large? For a man: is your shirt collar 17 inches (43 cm) or larger? For a woman: is your shirt collar 16 inches (41 cm) or larger?"),
      {
        type: "expression",
        name: KEY.score,
        title: "STOP-Bang score",
        clinicianOnly: true,
        calcKind: "score",
        expression: formatScore(form),
        bands: [
          { id: "b_low", label: "Low risk", max: 2 },
          { id: "b_intermediate", label: "Intermediate risk", max: 4 },
          { id: "b_high", label: "High risk", whenText: "Yes to 2 or more of the 4 STOP questions, and male, a BMI over 35 or a large neck", when: `${stopPart} and ({patientSex} = 'male' or {${bmi}} > 35 or {${KEY.neck}} = '${YES}')` },
        ],
      },
    ],
  };
}

/** The ready-made score a Group is (its `readyScore`), if any. */
export const readyScoreOf = (el: ElementJson | undefined): ReadyScore | undefined =>
  el && el.type === "panel" ? READY_SCORES.find((r) => r.id === el.readyScore) : undefined;

/** The ready-made score an element belongs to: the Group itself, or the Group it sits in. */
export function readyScoreAround(doc: ChapterJson, name: string): { group: ElementJson; score: ReadyScore } | undefined {
  const loc = findElement(doc, name);
  for (const el of [loc?.el, loc?.parent]) {
    const score = readyScoreOf(el);
    if (el && score) return { group: el, score };
  }
  return undefined;
}

/** Inserts a ready-made score as a Group at `target` (a page or a Group). */
export function insertReadyScore(doc: ChapterJson, target: Target, id: string, bmi: string): { doc: ChapterJson; name: string } {
  if (id !== STOP_BANG.id) return { doc, name: "" };
  const group = stopBangGroup(bmi);
  return {
    doc: edit(doc, (d) => {
      const parent = target.parent ? findElement(d, target.parent)?.el : undefined;
      const list = parent ? (parent.elements ??= []) : (pagesOf(d).find((p) => p.name === target.page)!.elements ??= []);
      list.splice(target.index ?? list.length, 0, group);
    }),
    name: group.name,
  };
}

/** A copy without what an author may change: the Group's condition and the disclosures; text in English only. */
function lockedPart(el: ElementJson): unknown {
  const copy = structuredClone(el) as Record<string, unknown>;
  delete copy.visibleIf;
  const strip = (x: Record<string, unknown>) => {
    delete x.clinicalOutputs;
    for (const k of ["title", "description"]) if (x[k] !== undefined) x[k] = textOf(x[k]);
    if (Array.isArray(x.choices)) x.choices = (x.choices as Record<string, unknown>[]).map(({ clinicalOutputs: _drop, ...c }) => ({ ...c, text: textOf(c.text) }));
    if (Array.isArray(x.bands)) x.bands = (x.bands as Record<string, unknown>[]).map(({ clinicalOutputs: _drop, ...b }) => b);
    if (Array.isArray(x.elements)) (x.elements as Record<string, unknown>[]).forEach(strip);
  };
  strip(copy);
  return canonical(copy);
}

const canonical = (v: unknown): unknown =>
  Array.isArray(v) ? v.map(canonical)
    : v && typeof v === "object" ? Object.fromEntries(Object.keys(v as object).sort().map((k) => [k, canonical((v as Record<string, unknown>)[k])]))
      : v;

/**
 * Ready-made scores that were changed (in the Code tab, or by the API): they block publishing. A
 * changed one has to be deleted and inserted again. A second copy in the HQ is refused by the key
 * checks (its keys are fixed).
 */
export function readyScoreProblems(doc: ChapterJson): string[] {
  const out: string[] = [];
  for (const { el } of allElements(doc)) {
    if (el.type !== "panel" || el.readyScore === undefined) continue;
    const score = readyScoreOf(el);
    if (!score) {
      out.push(`The ready-made score “${textOf(el.title) || el.name}” isn't one Preopsis has`);
      continue;
    }
    const bmi = typeof el.readyScoreBmi === "string" ? el.readyScoreBmi : "";
    if (!bmi || JSON.stringify(lockedPart(el)) !== JSON.stringify(lockedPart(stopBangGroup(bmi)))) {
      out.push(`${score.name} has been changed: a ready-made score can't be changed. Delete it and insert it again.`);
    }
  }
  return out;
}
