// Who is viewing (VEW-01/02). Patients never receive clinician-only content: it is
// removed from the JSON before the survey is built, not just hidden.
import type { SurveyModel } from "survey-core";

export type Viewer = "patient" | "clinician";

type Json = Record<string, unknown>;

/** A copy of the survey JSON without clinician-only questions and groups (at any depth). */
export function stripClinicianOnly<T extends object>(json: T): T {
  const copy = structuredClone(json) as Json;
  for (const page of (copy.pages as Json[] | undefined) ?? []) stripElements(page);
  stripElements(copy); // a survey can also have top-level "elements"
  return copy as T;
}

function stripElements(container: Json) {
  const elements = container.elements as Json[] | undefined;
  if (!Array.isArray(elements)) return;
  container.elements = elements.filter((el) => el.clinicianOnly !== true);
  for (const el of container.elements as Json[]) stripElements(el);
}

/** Sets {viewer} so conditions can test who is viewing (LOG-03, later). */
export function setViewer(model: SurveyModel, viewer: Viewer): void {
  model.setVariable("viewer", viewer);
}
