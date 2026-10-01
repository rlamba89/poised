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
  for (const key of ["elements", "templateElements"]) {
    const elements = container[key] as Json[] | undefined;
    if (!Array.isArray(elements)) continue;
    container[key] = elements.filter((el) => el.clinicianOnly !== true);
    for (const el of container[key] as Json[]) stripElements(el);
  }
}

/**
 * The form as patients get it (VEW-01, STR-03): no clinician-only content, no test cases,
 * and each Section on its own screen.
 */
export function forPatient<T extends object>(json: T): T {
  const copy = stripClinicianOnly(json) as Json;
  delete copy.testCases;
  return patientScreens(copy) as T;
}

/**
 * Splits pages at Sections (STR-03): each Section becomes its own screen, and the questions
 * between Sections stay together. A Section heading is its title, or, when "Show as patient
 * page heading" is off, the first question's text (the Section's title is hidden).
 */
export function patientScreens<T extends object>(json: T): T {
  const copy = structuredClone(json) as Json;
  const pages = (copy.pages as Json[] | undefined) ?? [];
  copy.pages = pages.flatMap((page) => {
    const screens: Json[][] = [];
    let current: Json[] = [];
    for (const el of (page.elements as Json[] | undefined) ?? []) {
      if (el.type === "panel" && el.patientPage === true) {
        if (current.length) screens.push(current);
        screens.push([el.showAsHeading === false ? { ...el, title: undefined } : el]);
        current = [];
      } else current.push(el);
    }
    if (current.length) screens.push(current);
    if (screens.length <= 1) return [page];
    return screens.map((elements, i) => ({ ...page, name: i === 0 ? page.name : `${page.name}__${i}`, elements }));
  });
  return copy as T;
}

/** Sets {viewer} so conditions can test who is viewing (LOG-03, later). */
export function setViewer(model: SurveyModel, viewer: Viewer): void {
  model.setVariable("viewer", viewer);
}
