// The clinician's pre-fill markers (S26, PRE-04). Under each pre-filled question SurveyJS draws (so
// only questions shown: a hidden one isn't drawn), where the answer came from and how it matched
// ("Pre-filled from Demo pre-op HQ, 3 Mar 2026 (matched by link)", with the hospital when it's
// another trust's), and "Changed by patient" with the pre-filled answer where the patient's final
// answer differs, as corrections show what the patient said (corrections.ts). Rendering only; the
// rules are prefillMark in @poised/clinical.
import type { SurveyModel } from "survey-core";
import { prefillMark, prefillSourceText, type Answers, type PrefillEntry } from "@poised/clinical";
import { formatDay } from "@/lib/format";

/** `entries`: the set's pre-filled answers still in use (prefilledAnswers: none after Start fresh, S27). */
export function showPrefillOrigins(model: SurveyModel, entries: Record<string, PrefillEntry>, patient: Answers): void {
  model.onAfterRenderQuestion.add((_, o) => {
    const mark = prefillMark(o.question, entries, patient);
    o.htmlElement.querySelector(":scope > .sj-prefill-origin")?.remove();
    if (!mark) return;
    const box = document.createElement("div");
    box.className = "sj-prefill-origin";
    box.dataset.testid = "prefill-origin";
    const source = document.createElement("div");
    source.textContent = prefillSourceText(mark.entry, formatDay);
    box.appendChild(source);
    if (mark.changed) {
      const changed = document.createElement("div");
      changed.className = "sj-prefill-changed";
      changed.dataset.testid = "prefill-changed";
      changed.textContent = `Changed by patient: pre-filled “${mark.was}”`;
      box.appendChild(changed);
    }
    o.htmlElement.appendChild(box);
  });
}
