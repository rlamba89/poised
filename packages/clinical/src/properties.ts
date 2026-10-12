// SurveyJS custom properties for clinical metadata. All are non-translatable (CLN-08).
import { Serializer, type Base } from "survey-core";
import { registerQuestionTypes } from "./questionTypes";
import type { ClinicalOutput } from "./types";

const OUTPUTS = "clinicalOutputs";

/**
 * In memory, outputs are held in an immutable `{ items }` box; in JSON they are a
 * plain array. A bare array breaks redo: survey-core overwrites an existing array
 * value in place, which empties the array the undo stack recorded (spike 2).
 */
type OutputsBox = { items: ClinicalOutput[] };

/** Registers the properties once. Call before creating any survey or Creator. */
export function registerClinicalProperties(): void {
  registerQuestionTypes();
  if (Serializer.findProperty("question", "clinicianOnly")) return;

  for (const cls of ["question", "panel"]) {
    Serializer.addProperty(cls, {
      name: "clinicianOnly:boolean",
      displayName: "Clinician only",
      // First setting of the General tab. (In v3 each category is a separate tab,
      // and the first tab opens by default, so a one-checkbox "Clinical" tab first
      // would hide the question's title and description.)
      category: "general",
      visibleIndex: 0,
      default: false,
      isLocalizable: false,
    });
  }
  for (const cls of ["question", "itemvalue"]) {
    Serializer.addProperty(cls, {
      // A non-string type, so the undo stack doesn't merge quick edits into one step.
      name: `${OUTPUTS}:clinicaloutputs`,
      visible: false,
      isLocalizable: false,
      onSerializeValue: (obj: Base) => (obj.getPropertyValue(OUTPUTS) as OutputsBox | undefined)?.items,
      // From JSON the value is an array; from the Creator's undo/redo it is the box itself.
      onSetValue: (obj: Base, value: unknown) =>
        obj.setPropertyValue(OUTPUTS, Array.isArray(value) ? (value.length ? { items: value } : undefined) : value),
    });
  }
}

/** The outputs on a question or option (empty when none). */
export function outputsOf(obj: Base): ClinicalOutput[] {
  return ((obj.getPropertyValue(OUTPUTS) as OutputsBox | undefined)?.items ?? []).map(normalizeOutput);
}

/**
 * An output as saved may lack `codes` or its note's `category` (the e2e fixtures and older content
 * have `{id, note: {text}}`). Every reader goes through this, so `codes` is always a list and a
 * note always has `text` and `category` (S08 test reports 2 and 3).
 */
export function normalizeOutput(o: ClinicalOutput): ClinicalOutput {
  const partial = o as Partial<ClinicalOutput>;
  const out: ClinicalOutput = { ...o, codes: Array.isArray(partial.codes) ? partial.codes : [] };
  if (partial.note) out.note = { ...partial.note, text: partial.note.text ?? "", category: partial.note.category ?? "" };
  return out;
}

/** Replaces the outputs on a question or option. One call is one undo step in the Creator. */
export function setOutputs(obj: Base, items: ClinicalOutput[]): void {
  obj.setPropertyValue(OUTPUTS, items.length ? { items: [...items] } : undefined);
}

export function isClinicianOnly(obj: Base): boolean {
  return obj.getPropertyValue("clinicianOnly") === true;
}
