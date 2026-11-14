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

// ---------------------------------------------------------------- MUST (S49)

export const MUST: ReadyScore = {
  id: "must",
  name: "MUST",
  source:
    "'Malnutrition Universal Screening Tool' ('MUST'), BAPEN: Step 1 BMI (>20 = 0, 18.5–20 = 1, <18.5 = 2), Step 2 unplanned weight loss in the past 3–6 months " +
    "(<5% = 0, 5–10% = 1, >10% = 2), Step 3 acute disease effect (2), Step 4 overall risk: 0 Low, 1 Medium, 2 or more High. " +
    "The 'Malnutrition Universal Screening Tool' ('MUST') is reproduced here with the kind permission of BAPEN (British Association for Parenteral and Enteral Nutrition). " +
    "For further information on 'MUST' see www.bapen.org.uk.",
  licence: "© BAPEN. Commercial use (an electronic patient record or app) needs BAPEN's permission, a signed licence and a fee before go-live.",
  needsBmi: false,
  note: "In an HQ it's the patient's weight-loss questions only, with no score: the BMI and today's weight are the observations', and the acute disease effect the clinician's, so the score is the MUST assessment's, on the episode.",
  build: () => mustGroup(),
};

/** MUST's keys: the patient's two questions, the assessment's items, the weight lost as a percentage, and the score. */
export const MUST_KEYS = {
  group: "must", lost: "must_wtloss", lostKg: "must_wtloss_kg", bmi: "must_bmi", weight: "must_weight", acute: "must_acute",
  pct: "must_wtloss_pct", score: "must_score",
};

const number = (name: string, title: string, min: number, max: number, more: Partial<ElementJson> = {}): ElementJson => ({
  type: "text", name, title, inputType: "number", min, max, isRequired: true, ...more,
});

/**
 * MUST's patient part, as an HQ has it (S49 answers 3): whether they've lost weight without trying
 * in the last 3–6 months and, if so, about how much. No score: the BMI and today's weight are the
 * observations' and the acute disease effect the clinician's (mustClinicianPart).
 */
export function mustGroup(): ElementJson {
  return {
    type: "panel",
    name: MUST_KEYS.group,
    title: "Weight loss",
    readyScore: MUST.id,
    elements: [
      { ...yesNoPoints(MUST_KEYS.lost, "Have you lost weight without trying in the last 3 to 6 months?", 0), alwaysFresh: true },
      number(MUST_KEYS.lostKg, "About how much weight have you lost without trying (kg)?", 0, 100, { visibleIf: `{${MUST_KEYS.lost}} = 'o_yes'`, alwaysFresh: true }),
    ],
  };
}

/**
 * The MUST assessment's items and score: the BMI and today's weight (the observations' or the
 * patient's, S47's precedence), the weight lost in kg (the HQ's: 0 when they said No), and the
 * clinician's acute disease effect, worded as BAPEN requires. The weight lost is a percentage of the
 * weight before (today's plus what was lost), to one decimal place, and scored as shown. Incomplete
 * while any of the four is missing.
 */
export function mustClinicianPart(): ElementJson[] {
  const { bmi, weight, lostKg, acute } = MUST_KEYS;
  const pct = `round({${lostKg}} * 100 / ({${weight}} + {${lostKg}}), 1)`;
  return [
    number(bmi, "BMI (kg/m²)", 5, 150),
    number(weight, "Weight now (kg)", 1, 400),
    number(lostKg, "Unplanned weight loss in the past 3–6 months (kg)", 0, 100),
    yesNoPoints(acute, "Acute disease effect: is the patient acutely ill and there has been or is likely to be no nutritional intake for >5 days?", 2),
    {
      type: "expression",
      name: MUST_KEYS.pct,
      title: "Weight lost",
      calcKind: "custom",
      expression: `iif({${weight}} notempty and {${lostKg}} notempty, ${pct}, '')`,
      format: "{0}% of body weight lost",
    },
    {
      type: "expression",
      name: MUST_KEYS.score,
      title: "MUST score",
      calcKind: "score",
      expression:
        `iif(${ALL_ANSWERED}('${bmi}', '${weight}', '${lostKg}', '${acute}'), ` +
        `iif({${bmi}} < 18.5, 2, iif({${bmi}} <= 20, 1, 0)) + iif(${pct} > 10, 2, iif(${pct} >= 5, 1, 0)) + score('${acute}'), '')`,
      bands: [
        { id: "b_low", label: "Low risk", max: 0 },
        { id: "b_medium", label: "Medium risk", max: 1 },
        { id: "b_high", label: "High risk" },
      ],
    },
  ];
}

// ---------------------------------------------------------------- Morse Fall Scale (S49)

export const MORSE: ReadyScore = {
  id: "morse",
  name: "Morse Fall Scale",
  source:
    "Morse Fall Scale: history of falling 25, secondary diagnosis 15, ambulatory aid 0/15/30, IV or heparin lock 20, gait 0/10/20, mental status 0/15 (0–125). " +
    "Morse JM, Morse RM, Tylko SJ. Development of a scale to identify the fall-prone patient. Can J Aging 1989;8(4):366–77; " +
    "as in Dykes PC et al. JAMA 2010;304(17):1912–8. Bands: low 0–24, moderate 25–44, high 45 or more (Morse leaves the cut-offs to each hospital).",
  licence: "© Janice Morse. Free of charge, but the author's permission is needed before go-live.",
  needsBmi: false,
  note: "In an HQ it's the patient's two questions only (a fall, a walking aid), with no score: the other four items are the clinician's, so the score is the Morse assessment's, on the episode.",
  build: () => morseGroup(),
};

/** The Morse Fall Scale's keys, in the published order of its items, and the score. */
export const MORSE_KEYS = {
  group: "morse", falls: "morse_falls", secondary: "morse_secondary", aid: "morse_aid", iv: "morse_iv", gait: "morse_gait",
  mental: "morse_mental", score: "morse_score",
};

const choiceOf = (name: string, title: string, choices: [value: string, text: string, score: number][]): ElementJson => ({
  type: "radiogroup", name, title, colCount: 0, isRequired: true, choices: choices.map(([value, text, score]) => ({ value, text, score })),
});

/** Morse's patient part, as an HQ has it: a fall in the last 3 months, and what they use to walk. No score. */
export function morseGroup(): ElementJson {
  return {
    type: "panel",
    name: MORSE_KEYS.group,
    title: "Falls",
    readyScore: MORSE.id,
    elements: [
      { ...yesNoPoints(MORSE_KEYS.falls, "Have you had a fall in the last 3 months?", 25), alwaysFresh: true },
      choiceOf(MORSE_KEYS.aid, "Do you use anything to help you walk?", [
        ["o_aid_none", "Nothing", 0],
        ["o_aid_walker", "Crutches, a stick or a walking frame", 15],
        ["o_aid_furniture", "I hold on to furniture", 30],
      ]),
    ],
  };
}

/**
 * The rest of the Morse Fall Scale, for the assessment: the clinician's four items and the score
 * (0–125). Incomplete while any of the six is unanswered.
 */
export function morseClinicianPart(): ElementJson[] {
  const { falls, secondary, aid, iv, gait, mental } = MORSE_KEYS;
  return [
    yesNoPoints(secondary, "Secondary diagnosis: does the patient have more than one medical diagnosis?", 15),
    yesNoPoints(iv, "IV therapy: does the patient have an IV line or a heparin (saline) lock?", 20),
    choiceOf(gait, "Gait", [["o_gait_normal", "Normal, on bed rest or in a wheelchair", 0], ["o_gait_weak", "Weak", 10], ["o_gait_impaired", "Impaired", 20]]),
    choiceOf(mental, "Mental status", [["o_mental_oriented", "Knows their own ability", 0], ["o_mental_forgets", "Overestimates or forgets their limits", 15]]),
    {
      type: "expression",
      name: MORSE_KEYS.score,
      title: "Morse Fall Scale score",
      calcKind: "score",
      expression: formatScore({ kind: "score", questions: [falls, secondary, aid, iv, gait, mental] }),
      bands: [
        { id: "b_low", label: "Low risk", max: 24 },
        { id: "b_moderate", label: "Moderate risk", max: 44 },
        { id: "b_high", label: "High risk" },
      ],
    },
  ];
}

// ---------------------------------------------------------------- VTE (S49)

export const VTE: ReadyScore = {
  id: "vte",
  name: "VTE risk",
  source:
    "Department of Health, Risk assessment for venous thromboembolism (VTE), 2010 (NICE NG89's resources), for surgical patients: " +
    "each thrombosis and bleeding risk factor ticked; any tick for thrombosis risk should prompt thromboprophylaxis according to NICE guidance, and any tick for " +
    "bleeding risk should prompt clinical staff to consider if bleeding risk is sufficient to preclude pharmacological intervention.",
  licence: "© Crown copyright 2010; reusable free under the Open Government Licence (attribution).",
  needsBmi: false,
  note: "In an HQ it's the patient's own risk factors only, with no result: the age and BMI come from the record and the observations, and the admission's risk factors are the clinician's, on the episode.",
  build: () => vteGroup(),
};

// The tool's risk factors, in its order: its key, its title, the short label the card and Go's
// messages use, and whether the patient answers it in the HQ (our wording, for a patient; otherwise
// the clinician's, worded as the form words them). Age > 60 and obesity are their own items.
type VteFactor = [key: string, title: string, label: string, patient?: true];
const VTE_THROMBOSIS_PATIENT: VteFactor[] = [
  ["vte_cancer", "Do you have cancer, or are you having treatment for cancer?", "Active cancer or cancer treatment", true],
  ["vte_dehydration", "Dehydration", "Dehydration"],
  ["vte_thrombophilia", "Have you been told you have a condition that makes your blood clot more easily (thrombophilia)?", "Known thrombophilias", true],
  ["vte_comorbidity", "One or more significant medical comorbidities (eg heart disease; metabolic, endocrine or respiratory pathologies; acute infectious diseases; inflammatory conditions)", "Significant medical comorbidities"],
  ["vte_history", "Have you, or a parent, brother, sister or child of yours, ever had a blood clot in a leg or lung (DVT or pulmonary embolism)?", "Personal or family history of VTE", true],
  ["vte_hrt", "Are you taking hormone replacement therapy (HRT)?", "Hormone replacement therapy", true],
  ["vte_contraceptive", "Are you using a contraceptive that contains oestrogen (such as the combined pill, patch or ring)?", "Oestrogen-containing contraceptive", true],
  ["vte_varicose", "Do you have varicose veins that have been red, hot and painful (phlebitis)?", "Varicose veins with phlebitis", true],
  ["vte_pregnancy", "Are you pregnant, or have you had a baby in the last 6 weeks?", "Pregnancy or under 6 weeks post partum", true],
];
const VTE_THROMBOSIS_ADMISSION: VteFactor[] = [
  ["vte_mobility", "Significantly reduced mobility for 3 days or more", "Reduced mobility for 3 days or more"],
  ["vte_hip_knee", "Hip or knee replacement", "Hip or knee replacement"],
  ["vte_hip_fracture", "Hip fracture", "Hip fracture"],
  ["vte_time_90", "Total anaesthetic + surgical time > 90 minutes", "Anaesthetic + surgical time over 90 minutes"],
  ["vte_pelvis_60", "Surgery involving pelvis or lower limb with a total anaesthetic + surgical time > 60 minutes", "Pelvis or lower limb surgery over 60 minutes"],
  ["vte_acute_surgical", "Acute surgical admission with inflammatory or intra-abdominal condition", "Acute surgical admission (inflammatory or intra-abdominal)"],
  ["vte_critical_care", "Critical care admission", "Critical care admission"],
  ["vte_surgery_mobility", "Surgery with significant reduction in mobility", "Surgery with significant reduction in mobility"],
];
const VTE_BLEEDING: VteFactor[] = [
  ["vte_active_bleeding", "Active bleeding", "Active bleeding"],
  ["vte_acquired_bleeding", "Acquired bleeding disorders (such as acute liver failure)", "Acquired bleeding disorders"],
  ["vte_anticoagulant", "Do you take medicine to thin your blood (an anticoagulant, such as warfarin, apixaban or rivaroxaban)?", "Anticoagulants", true],
  ["vte_stroke", "Acute stroke", "Acute stroke"],
  ["vte_platelets", "Thrombocytopaenia (platelets < 75×10⁹/l)", "Thrombocytopaenia"],
  ["vte_hypertension", "Uncontrolled systolic hypertension (230/120 mmHg or higher)", "Uncontrolled systolic hypertension"],
  ["vte_inherited_bleeding", "Do you have an inherited bleeding disorder, such as haemophilia or von Willebrand disease?", "Inherited bleeding disorders", true],
  ["vte_neuro_spinal_eye", "Neurosurgery, spinal surgery or eye surgery", "Neurosurgery, spinal or eye surgery"],
  ["vte_high_bleeding", "Other procedure with high bleeding risk", "Other procedure with high bleeding risk"],
  ["vte_lp_next_12h", "Lumbar puncture/epidural/spinal anaesthesia expected within the next 12 hours", "Spinal or epidural in the next 12 hours"],
  ["vte_lp_last_4h", "Lumbar puncture/epidural/spinal anaesthesia within the previous 4 hours", "Spinal or epidural in the previous 4 hours"],
];
const VTE_ALL = [...VTE_THROMBOSIS_PATIENT, ...VTE_THROMBOSIS_ADMISSION, ...VTE_BLEEDING];

/** The questions the HQ doesn't ask a patient recorded as male (S49 verification 1, Rahul's answer 2). */
const VTE_NOT_MALE = new Set(["vte_hrt", "vte_contraceptive", "vte_pregnancy"]);

/** The question about a recent time, asked afresh, never copied from an earlier HQ (S49 verification 2, Rahul). */
const VTE_FRESH = new Set(["vte_pregnancy"]);

/**
 * The VTE assessment's keys: the thrombosis risk factors (besides the age and BMI), the bleeding
 * ones, those the patient answers in the HQ, the BMI item, the score and the bleeding count.
 */
export const VTE_KEYS = {
  group: "vte",
  thrombosis: [...VTE_THROMBOSIS_PATIENT, ...VTE_THROMBOSIS_ADMISSION].map(([k]) => k),
  bleeding: VTE_BLEEDING.map(([k]) => k),
  patient: VTE_ALL.filter(([, , , p]) => p).map(([k]) => k),
  bmi: "vte_bmi",
  score: "vte_score",
  bleedingCount: "vte_bleeding",
};

/**
 * The VTE items in the form's order, each [key, short label]: thrombosis (the age second and the
 * BMI fifth, where the form has them), then bleeding. One list, so the card can't drift from the keys.
 */
export const VTE_ITEMS: { thrombosis: [string, string][]; bleeding: [string, string][] } = {
  // The age goes after active cancer and the BMI after known thrombophilias, as on the form.
  thrombosis: [...VTE_THROMBOSIS_PATIENT, ...VTE_THROMBOSIS_ADMISSION].flatMap(([k, , label]): [string, string][] => [
    [k, label],
    ...(k === "vte_cancer" ? [["patientAge", "Age over 60"] as [string, string]] : []),
    ...(k === "vte_thrombophilia" ? [[VTE_KEYS.bmi, "Obesity (BMI over 30)"] as [string, string]] : []),
  ]),
  bleeding: VTE_BLEEDING.map(([k, , label]): [string, string] => [k, label]),
};

/**
 * VTE's patient part, as an HQ has it: their own risk factors, Yes or No. No result. Pregnancy, an
 * oestrogen contraceptive and HRT aren't asked of a patient recorded as male (verification 1); the
 * assessment asks the clinician them (its copy has no such conditions: `withoutConditions` in assessments.ts).
 */
export function vteGroup(): ElementJson {
  return {
    type: "panel",
    name: VTE_KEYS.group,
    title: "Blood clots",
    readyScore: VTE.id,
    elements: VTE_ALL.filter(([, , , p]) => p).map(([k, title]) => ({
      ...yesNoPoints(k, title, 1),
      ...(VTE_NOT_MALE.has(k) ? { visibleIf: "{patientSex} <> 'male'" } : {}),
      ...(VTE_FRESH.has(k) ? { alwaysFresh: true } : {}),
    })),
  };
}

/**
 * The rest of the VTE assessment: the BMI, the clinician's risk factors, and the result: the number
 * of thrombosis ticks (age over 60 and a BMI over 30 included), any one of them "Thrombosis risk:
 * consider thromboprophylaxis", and the number of bleeding ticks. Incomplete while any is missing.
 */
export function vteClinicianPart(): ElementJson[] {
  const form: ScoreForm = {
    kind: "score",
    questions: VTE_KEYS.thrombosis,
    lines: [
      { points: 1, rule: { left: "patientAge", op: "gt", value: 60 } },
      { points: 1, rule: { left: VTE_KEYS.bmi, op: "gt", value: 30 } },
    ],
  };
  const bleeding = VTE_KEYS.bleeding.map((k) => `'${k}'`).join(", ");
  return [
    number(VTE_KEYS.bmi, "BMI (kg/m²)", 5, 150),
    ...VTE_ALL.filter(([, , , p]) => !p).map(([k, title]) => yesNoPoints(k, title, 1)),
    {
      type: "expression",
      name: VTE_KEYS.score,
      title: "Thrombosis risk factors",
      calcKind: "score",
      expression: formatScore(form),
      bands: [
        { id: "b_none", label: "No thrombosis risk factors ticked", max: 0 },
        { id: "b_risk", label: "Thrombosis risk: consider thromboprophylaxis" },
      ],
    },
    {
      type: "expression",
      name: VTE_KEYS.bleedingCount,
      title: "Bleeding risk factors",
      calcKind: "custom",
      expression: `iif(${ALL_ANSWERED}(${bleeding}), score(${bleeding}), '')`,
      format: "Bleeding risk factors: {0}",
    },
  ];
}

// ---------------------------------------------------------------- Rockwood CFS (S49)

// The Clinical Frailty Scale (an assessment only: never in an HQ, so no ReadyScore): version 2.0
// (2020), nine levels, 1 Very Fit to 9 Terminally Ill, scored from the person's baseline (about two
// weeks before). Rockwood K et al. CMAJ 2005;173(5):489–95; Geriatric Medicine Research, Dalhousie
// University. © Dalhousie University: a software vendor may need its licence (asked through its
// permission portal) before go-live.

/** The Clinical Frailty Scale's keys: the level and its score. */
export const CFS_KEYS = { level: "cfs_level", score: "cfs_score" };

const CFS_LEVELS = [
  "Very Fit", "Fit", "Managing Well", "Living with Very Mild Frailty", "Living with Mild Frailty", "Living with Moderate Frailty",
  "Living with Severe Frailty", "Living with Very Severe Frailty", "Terminally Ill",
];

/**
 * The Clinical Frailty Scale (Rockwood): the clinician picks the level, a clinical judgement, so it
 * is never in an HQ (it isn't in READY_SCORES). The score is the level, banded by its name.
 */
export function cfsElements(): ElementJson[] {
  return [
    choiceOf(CFS_KEYS.level, "Clinical Frailty Scale level, from the patient's usual state (about two weeks before)",
      CFS_LEVELS.map((label, i) => [`o_cfs_${i + 1}`, `${i + 1}: ${label}`, i + 1])),
    {
      type: "expression",
      name: CFS_KEYS.score,
      title: "Clinical Frailty Scale",
      calcKind: "score",
      expression: formatScore({ kind: "score", questions: [CFS_KEYS.level] }),
      bands: CFS_LEVELS.map((label, i) => ({ id: `b_cfs_${i + 1}`, label, ...(i < 8 ? { max: i + 1 } : {}) })),
    },
  ];
}

/** The ready-made scores an author can insert (MUST, Morse and VTE as the patient's parts only, S49). */
export const READY_SCORES: ReadyScore[] = [STOP_BANG, DASI, PRISMA_7, APFEL, MUST, MORSE, VTE];

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
