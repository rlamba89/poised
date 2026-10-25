// Question keys (plan C4, session S23): what the editor needs to draw a question's pre-fill badge.
// Go does the rest (apps/api/internal/keys): the matches behind "Pre-fills from N other HQs", and
// the key checks Publish runs (a key used twice in the HQ; a published question whose answer
// changed), which the editor shows in its Publish window before the author clicks.
import { SUMMARY_TYPE, type ChapterJson, type ElementJson } from "./doc";

/**
 * Whether an element can be pre-filled, so may have "Always ask fresh" and a pre-fill badge: an
 * answer not marked clinician-only. Never statements, calculations, groups, the profile or summaries (no
 * answer), nor signatures, uploads and the BMI calculator (one-off answers Go never copies).
 */
export function canAskFresh(el: ElementJson): boolean {
  if (el.clinicianOnly) return false;
  return !["panel", "html", "expression", "image", "profile", "bmi", "signaturepad", "file", SUMMARY_TYPE].includes(el.type);
}

/**
 * Where a question sits, for its pre-fill label: inside a repeating group (its answer travels in the
 * group's rows, so it has no label of its own), and clinician-only through a group or page around it.
 */
export function keyPlace(doc: ChapterJson, name: string): { inList: boolean; clinicianOnly: boolean } {
  type Place = { inList: boolean; clinicianOnly: boolean };
  const find = (els: ElementJson[] | undefined, at: Place): Place | undefined => {
    for (const el of els ?? []) {
      const here = { inList: at.inList, clinicianOnly: at.clinicianOnly || !!el.clinicianOnly };
      if (el.name === name) return here;
      const inner = find(el.type === "paneldynamic" ? el.templateElements : el.elements, { ...here, inList: here.inList || el.type === "paneldynamic" });
      if (inner) return inner;
    }
    return undefined;
  };
  for (const page of doc.pages ?? []) {
    const found = find(page.elements, { inList: false, clinicianOnly: page.clinicianOnly === true });
    if (found) return found;
  }
  return { inList: false, clinicianOnly: false };
}
