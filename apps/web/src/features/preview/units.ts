// Shows each Number question's unit after its box (QT-06). Rendering only, so it lives here
// rather than in @poised/clinical, which has no DOM.
import type { SurveyModel } from "survey-core";

export function showUnits(model: SurveyModel): void {
  model.onAfterRenderQuestion.add((_, o) => {
    const unit = o.question.getPropertyValue("unit");
    const input = o.htmlElement.querySelector("input");
    if (!unit || !input?.parentElement || input.parentElement.querySelector(".sj-unit")) return;
    input.parentElement.classList.add("sj-has-unit");
    const span = document.createElement("span");
    span.className = "sj-unit";
    span.textContent = String(unit);
    input.insertAdjacentElement("afterend", span);
  });
}
