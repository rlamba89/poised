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
import { ALL_ANSWERED, formatScore, type ScoreForm } from "./scores";

export type ReadyScore = {
  id: string;
  name: string;
  /** Which published version, and where it comes from. */
  source: string;
  /** Who owns it, and the licence its use needs. */
  licence: string;
  /** What the author must choose when inserting it. */
  needsBmi: boolean;
  /** What the author should know before inserting it, if anything. */
  note?: string;
  /** The Group as inserted, with the BMI it uses (`bmi`, when it needs one). */
  build: (bmi: string) => ElementJson;
};

export const STOP_BANG: ReadyScore = {
  id: "stopbang",
  name: "STOP-Bang",
  source:
    "STOP-Bang questionnaire, updated version (neck: shirt collar 17 inches / 43 cm for men, 16 inches / 41 cm for women). " +
    "Chung F, Abdullah HR, Liao P. STOP-Bang Questionnaire: a practical approach to screen for obstructive sleep apnea. Chest 2016;149(3):631–8.",
  licence: "© Dr Frances Chung and University Health Network (stopbang.ca). Use beyond education and research needs a licence from UHN.",
  needsBmi: true,
  build: (bmi) => stopBangGroup(bmi),
};

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

// ---------------------------------------------------------------- DASI (S48)

export const DASI: ReadyScore = {
  id: "dasi",
  name: "DASI",
  source:
    "Duke Activity Status Index: twelve items, each Yes its published weight (0–58.2); METs = (0.43 × DASI + 9.6) / 3.5. " +
    "Hlatky MA et al. A brief self-administered questionnaire to determine functional capacity. Am J Cardiol 1989;64(10):651–4. " +
    "Poor functional capacity at 34 or less: Wijeysundera DN et al. Br J Anaesth 2020;124(3):261–70.",
  licence: "© Mark Hlatky. Licensed by Greenlight Health (the exclusive licensor), by use case: a licence is needed before go-live.",
  needsBmi: false,
  build: () => dasiGroup(),
};

const DASI_ITEMS: [key: string, title: string, weight: number][] = [
  ["dasi_selfcare", "Can you take care of yourself, that is, eating, dressing, bathing or using the toilet?", 2.75],
  ["dasi_walk_indoors", "Can you walk indoors, such as around your house?", 1.75],
  ["dasi_walk_block", "Can you walk a block or two on level ground?", 2.75],
  ["dasi_stairs", "Can you climb a flight of stairs or walk up a hill?", 5.5],
  ["dasi_run", "Can you run a short distance?", 8],
  ["dasi_light_work", "Can you do light work around the house like dusting or washing dishes?", 2.7],
  ["dasi_moderate_work", "Can you do moderate work around the house like vacuuming, sweeping floors, or carrying in groceries?", 3.5],
  ["dasi_heavy_work", "Can you do heavy work around the house like scrubbing floors, or lifting or moving heavy furniture?", 8],
  ["dasi_yardwork", "Can you do yardwork like raking leaves, weeding or pushing a power mower?", 4.5],
  ["dasi_sexual", "Can you have sexual relations?", 5.25],
  ["dasi_moderate_recreation", "Can you participate in moderate recreational activities like golf, bowling, dancing, doubles tennis, or throwing a baseball or football?", 6],
  ["dasi_strenuous_sports", "Can you participate in strenuous sports like swimming, singles tennis, football, basketball or skiing?", 7.5],
];

/** DASI's keys: the twelve items in order, the score and the METs. */
export const DASI_KEYS = { group: "dasi", items: DASI_ITEMS.map(([k]) => k), score: "dasi_score", mets: "dasi_mets" };

/** A Yes/No question whose Yes scores `points` (and No 0). */
const yesNoPoints = (name: string, title: string, yes: number, no = 0): ElementJson => ({
  type: "radiogroup", name, title, yesNo: true, colCount: 0, isRequired: true,
  choices: [{ value: YES, text: "Yes", score: yes }, { value: "o_no", text: "No", score: no }],
});

/**
 * DASI as a Group: each Yes adds its weight (0–58.2), rounded to 2 places so the sum of decimals
 * never misses the cut-off; Incomplete while any item has no answer. Its METs are worked out from
 * the score, to one decimal place: (0.43 × DASI + 9.6) / 3.5. One band, "Poor functional capacity",
 * at 34 or less (S48 answers 2); above it, no band.
 */
export function dasiGroup(): ElementJson {
  const keys = DASI_KEYS.items.map((k) => `'${k}'`).join(", ");
  return {
    type: "panel",
    name: DASI_KEYS.group,
    title: "DASI",
    readyScore: DASI.id,
    elements: [
      ...DASI_ITEMS.map(([k, title, w]) => yesNoPoints(k, title, w)),
      {
        type: "expression",
        name: DASI_KEYS.score,
        title: "DASI score",
        clinicianOnly: true,
        calcKind: "score",
        expression: `iif(${ALL_ANSWERED}(${keys}), round(score(${keys}), 2), '')`,
        bands: [{ id: "b_poor", label: "Poor functional capacity", max: 34 }],
      },
      {
        type: "expression",
        name: DASI_KEYS.mets,
        title: "DASI METs",
        clinicianOnly: true,
        calcKind: "custom",
        expression: `iif({${DASI_KEYS.score}} notempty, round((0.43 * {${DASI_KEYS.score}} + 9.6) / 3.5, 1), '')`,
        format: "{0} METs",
        displayStyle: "decimal",
        minimumFractionDigits: 1,
        maximumFractionDigits: 1,
      },
    ],
  };
}

// ---------------------------------------------------------------- PRISMA-7 (S48)

export const PRISMA_7: ReadyScore = {
  id: "prisma7",
  name: "PRISMA-7",
  source:
    "PRISMA-7 questionnaire: one point per Yes, and older than 85 and male one point each (0–7); 3 or more is an increased risk of frailty. " +
    "Raîche M, Hébert R, Dubois MF. PRISMA-7: a case-finding tool to identify older adults with moderate to severe disabilities. Arch Gerontol Geriatr 2008;47(1):9–18. " +
    "Wording as BCGuidelines.ca prints it (Frailty in Older Adults, 2017).",
  licence: "© CESS (Centre d'expertise en santé de Sherbrooke). An electronic version needs its permission before go-live.",
  needsBmi: false,
  build: () => prisma7Group(),
};

const PRISMA7_ITEMS: [key: string, title: string][] = [
  ["prisma7_limit", "In general, do you have any health problems that require you to limit your activities?"],
  ["prisma7_help", "Do you need someone to help you on a regular basis?"],
  ["prisma7_home", "In general, do you have any health problems that require you to stay at home?"],
  ["prisma7_count_on", "If you need help, can you count on someone close to you?"],
  ["prisma7_aid", "Do you regularly use a stick, walker or wheelchair to move about?"],
];

/** PRISMA-7's keys: questions 3 to 7 in order (1 and 2 are the age and sex), and the score. */
export const PRISMA7_KEYS = { group: "prisma7", items: PRISMA7_ITEMS.map(([k]) => k), score: "prisma7_score" };

/**
 * PRISMA-7 as a Group: questions 3 to 7, each Yes 1 point (question 6 too, as published), plus 1
 * point for older than 85 and 1 for male, from the patient's record (0–7). 3 or more is "Increased
 * risk of frailty"; Incomplete while any of the seven has no answer (a sex of Other or Not known
 * can't decide the male point, as for STOP-Bang).
 */
export function prisma7Group(): ElementJson {
  const form: ScoreForm = {
    kind: "score",
    questions: PRISMA7_KEYS.items,
    lines: [
      { points: 1, rule: { left: "patientAge", op: "gt", value: 85 } },
      { points: 1, rule: { left: "patientSex", op: "eq", value: "male" } },
    ],
  };
  return {
    type: "panel",
    name: PRISMA7_KEYS.group,
    title: "PRISMA-7",
    readyScore: PRISMA_7.id,
    elements: [
      ...PRISMA7_ITEMS.map(([k, title]) => yesNoPoints(k, title, 1)),
      {
        type: "expression",
        name: PRISMA7_KEYS.score,
        title: "PRISMA-7 score",
        clinicianOnly: true,
        calcKind: "score",
        expression: formatScore(form),
        bands: [
          { id: "b_not_increased", label: "Not at increased risk", max: 2 },
          { id: "b_increased", label: "Increased risk of frailty" },
        ],
      },
    ],
  };
}

// ---------------------------------------------------------------- Apfel (S48)

export const APFEL: ReadyScore = {
  id: "apfel",
  name: "Apfel PONV",
  source:
    "Apfel simplified risk score for postoperative nausea and vomiting: female, non-smoker, a history of PONV or motion sickness, and post-op opioids, " +
    "1 point each; PONV in about 10%, 21%, 39%, 61% and 79% with 0 to 4. " +
    "Apfel CC, Läärä E, Koivuranta M, Greim CA, Roewer N. Anesthesiology 1999;91(3):693–700.",
  licence: "A published risk score: no licence found.",
  needsBmi: false,
  note: "In an HQ it's the patient's two questions only, with no score: the post-op opioids item is the clinician's, so the score is the Apfel assessment's, on the episode.",
  build: () => apfelGroup(),
};

/** Apfel's keys: the patient's two questions, the clinician's opioids item, and the score. */
export const APFEL_KEYS = { group: "apfel", smoking: "apfel_smoking", ponv: "apfel_ponv", opioids: "apfel_opioids", score: "apfel_score" };

/**
 * Apfel's patient part, as an HQ has it (S48 answers 4): smoking (a No is a non-smoker's point) and
 * a history of PONV or motion sickness. The sex is the patient's record's. No score: it needs the
 * clinician's post-op opioids item (apfelClinicianPart), so in an HQ it would never be complete.
 */
export function apfelGroup(): ElementJson {
  return {
    type: "panel",
    name: APFEL_KEYS.group,
    title: "Sickness after an anaesthetic",
    readyScore: APFEL.id,
    elements: [
      yesNoPoints(APFEL_KEYS.smoking, "Do you smoke?", 0, 1),
      yesNoPoints(APFEL_KEYS.ponv, "Have you ever felt or been sick after an anaesthetic, or do you get travel sickness (motion sickness)?", 1),
    ],
  };
}

/**
 * The rest of the Apfel score, for the assessment: the clinician's post-op opioids item and the
 * score (0–4), female 1 point, banded by the published incidence of PONV. Incomplete while an item
 * is missing, or the sex is Other or Not known (the female point can't be decided).
 */
export function apfelClinicianPart(): ElementJson[] {
  const form: ScoreForm = {
    kind: "score",
    questions: [APFEL_KEYS.smoking, APFEL_KEYS.ponv, APFEL_KEYS.opioids],
    lines: [{ points: 1, rule: { left: "patientSex", op: "eq", value: "female" } }],
  };
  return [
    yesNoPoints(APFEL_KEYS.opioids, "Will the patient be given opioids after surgery?", 1),
    {
      type: "expression",
      name: APFEL_KEYS.score,
      title: "Apfel score",
      calcKind: "score",
      expression: formatScore(form),
      bands: [
        { id: "b_10", label: "PONV risk about 10%", max: 0 },
        { id: "b_21", label: "PONV risk about 21%", max: 1 },
        { id: "b_39", label: "PONV risk about 39%", max: 2 },
        { id: "b_61", label: "PONV risk about 61%", max: 3 },
        { id: "b_79", label: "PONV risk about 79%" },
      ],
    },
  ];
}

/** The ready-made scores an author can insert. */
export const READY_SCORES: ReadyScore[] = [STOP_BANG, DASI, PRISMA_7, APFEL];

/**
 * The ready-made scores whose sex point a sex of Other can't decide (S44c), so it's Incomplete, as
 * Not known is: STOP-Bang's and PRISMA-7's male point, and Apfel's female point (S48).
 */
export const STRICT_SEX_SCORES = new Set([KEY.score, PRISMA7_KEYS.score, APFEL_KEYS.score]);

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
  const score = READY_SCORES.find((r) => r.id === id);
  if (!score) return { doc, name: "" };
  const group = score.build(bmi);
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
    if ((score.needsBmi && !bmi) || JSON.stringify(lockedPart(el)) !== JSON.stringify(lockedPart(score.build(bmi)))) {
      out.push(`${score.name} has been changed: a ready-made score can't be changed. Delete it and insert it again.`);
    }
  }
  return out;
}
