// The clinical metadata stored inside a chapter's SurveyJS JSON (plan section 6).

export type CodeSet = "SNOMED" | "ICD10";

/** A code picked from the library. `display` is a copy so the form stands alone. */
export type CodeRef = { set: CodeSet; code: string; display: string };

export type AsaGrade = "I" | "II" | "III" | "IV" | "V" | "VI";

export type ReviewFlag = "amber" | "red";

/** One output on an option or a question. Needs at least one code or a note (CLN-06). */
export type ClinicalOutput = {
  id: string;
  codes: CodeRef[];
  /** `text` may contain {answer}. `category` is the category name, copied from the library. */
  note?: { text: string; category: string };
  asa?: { grade: AsaGrade; emergency: boolean };
  flag?: ReviewFlag;
};
