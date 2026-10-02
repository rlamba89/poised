// Marks answers the clinician has changed with what the patient said ("Patient answered: …"),
// under the question, as the clinician validates. Rendering only, so it lives here.
import type { Question, SurveyModel } from "survey-core";
import { describeAnswer, isClinicianOnlyInTree, sameAnswer, type Answers } from "@poised/clinical";

export function showCorrections(model: SurveyModel, patient: Answers): void {
  const rendered = new Map<string, HTMLElement>();
  const update = (q: Question) => {
    const el = rendered.get(q.name);
    if (!el || isClinicianOnlyInTree(q)) return; // the patient never saw it
    let note = el.querySelector<HTMLElement>(":scope .sj-patient-answer");
    if (sameAnswer(q.value, patient[q.name])) {
      note?.remove();
      return;
    }
    if (!note) {
      note = document.createElement("div");
      note.className = "sj-patient-answer";
      el.appendChild(note);
    }
    note.textContent = `Patient answered: ${describeAnswer(q, patient[q.name])}`;
  };
  model.onAfterRenderQuestion.add((_, o) => {
    rendered.set(o.question.name, o.htmlElement);
    update(o.question);
  });
  model.onValueChanged.add((_, o) => {
    const q = model.getQuestionByName(o.name);
    if (q) update(q);
  });
}
