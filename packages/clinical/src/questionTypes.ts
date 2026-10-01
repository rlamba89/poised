// The Lifebox question types SurveyJS doesn't have, built from its own questions so the
// preview and the patient runtime render them (plan-redesign.md, step 5).
import { ComponentCollection, FunctionFactory, Serializer, type Question, type SurveyModel } from "survey-core";
import { bandFor } from "./logic";

/** Registers Medication, Admissions, BMI and Profile, and the flags the editor stores. Safe to call twice. */
export function registerQuestionTypes(): void {
  if (ComponentCollection.Instance.getCustomQuestionByName("medication")) return;

  // A repeating list (Lifebox "DynamicAdd"): each row is one medicine.
  ComponentCollection.Instance.add({
    name: "medication",
    title: "Medication",
    questionJSON: {
      type: "matrixdynamic",
      rowCount: 0,
      addRowText: "Add medication",
      noRowsText: "No medication added yet",
      columns: [
        { name: "name", title: "Medication name", cellType: "text", isRequired: true },
        { name: "dosage", title: "Dosage", cellType: "text", placeholder: "e.g. 500mg" },
        { name: "frequency", title: "Frequency", cellType: "text", placeholder: "e.g. twice daily" },
      ],
    },
  });

  ComponentCollection.Instance.add({
    name: "admissions",
    title: "Admissions",
    questionJSON: {
      type: "matrixdynamic",
      rowCount: 0,
      addRowText: "Add admission",
      noRowsText: "No admission added yet",
      columns: [
        { name: "reason", title: "Reason for admission", cellType: "text", isRequired: true },
        { name: "hospital", title: "Hospital", cellType: "text" },
        { name: "year", title: "Year", cellType: "text", inputType: "number", min: 1900 },
        {
          name: "anaesthetic",
          title: "Anaesthetic type",
          cellType: "dropdown",
          choices: ["General", "Local", "Sedation", "Spinal/Epidural", "Other", "Not applicable"],
        },
      ],
    },
  });

  // Height and weight, and the BMI worked out from them.
  ComponentCollection.Instance.add({
    name: "bmi",
    title: "BMI",
    elementsJSON: [
      { type: "text", name: "height", title: "What is your height?", description: "cm", inputType: "number", min: 50, max: 250 },
      { type: "text", name: "weight", title: "What is your weight?", description: "kg", inputType: "number", min: 2, max: 400 },
      {
        type: "expression",
        name: "bmi",
        title: "Your BMI is:",
        expression: "iif({composite.height} > 0 and {composite.weight} > 0, round({composite.weight} / (({composite.height} / 100) * ({composite.height} / 100)), 1), '')",
      },
    ],
  });

  // Read-only patient details. The preview fills them from the sample patient.
  ComponentCollection.Instance.add({
    name: "profile",
    title: "Profile",
    questionJSON: {
      type: "html",
      html:
        "<div class='sj-profile'><p><b>Name</b> {patientName}</p><p><b>Age</b> {patientAge}</p><p><b>Sex</b> {patientSex}</p></div>" +
        "<p class='sj-profile-note'>If any of these details are incorrect, please contact your hospital to make changes to your profile. In the meantime, you can continue your Health Questionnaire.</p>",
    },
  });

  // Flags the editor stores in the JSON. Hidden: there is no Creator property grid any more.
  const hidden = { visible: false, isLocalizable: false };
  Serializer.addProperty("radiogroup", { name: "yesNo:boolean", default: false, ...hidden });
  Serializer.addProperty("panel", { name: "patientPage:boolean", default: false, ...hidden });
  Serializer.addProperty("panel", { name: "showAsHeading:boolean", default: true, ...hidden });
  Serializer.addProperty("page", { name: "clinicalSummary:boolean", default: false, ...hidden });
  Serializer.addProperty("itemvalue", { name: "score:number", ...hidden });
  Serializer.addProperty("itemvalue", { name: "special", ...hidden });
  Serializer.addProperty("expression", { name: "bands", ...hidden });
  Serializer.addProperty("expression", { name: "calcKind", ...hidden });
  Serializer.addProperty("matrix", { name: "cellOutputs", ...hidden });
  Serializer.addProperty("text", { name: "dateFormat", ...hidden });
  Serializer.addProperty("text", { name: "unit", ...hidden });
  Serializer.addProperty("text", { name: "textFormat", ...hidden });
  Serializer.addProperty("matrixdynamic", { name: "dateList:boolean", default: false, ...hidden });
  Serializer.addProperty("survey", { name: "testCases", ...hidden });
  Serializer.addProperty("surveytrigger", { name: "page", ...hidden });
  Serializer.addProperty("survey", { name: "chapterVisibleIf", ...hidden });

  registerClinicalFunctions();

  Serializer.addProperty("medication", {
    name: "medicationType",
    default: "prescribed",
    choices: ["prescribed", "non_prescribed", "recreational"],
    ...hidden,
  });
}

/**
 * Functions for conditions and calculations (CAL-01/02/03), besides SurveyJS's own (age, dateDiff…):
 * - `score('q_a', 'q_b')`: the total of the chosen options' scores
 * - `band('q_calc')`: the ID of the band a Calculation's value falls in
 * - `bmi(heightCm, weightKg)`: the body mass index, to one decimal place
 */
function registerClinicalFunctions(): void {
  FunctionFactory.Instance.register("score", function (this: { survey?: SurveyModel }, params: unknown[]) {
    let total = 0;
    for (const name of params) {
      const q = this.survey?.getQuestionByName(String(name)) as (Question & { choices?: { value: unknown; score?: number }[] }) | null;
      if (!q || q.isEmpty()) continue;
      const chosen: unknown[] = Array.isArray(q.value) ? q.value : [q.value];
      for (const c of q.choices ?? []) if (chosen.includes(c.value)) total += Number(c.score ?? 0);
    }
    return total;
  });
  FunctionFactory.Instance.register("band", function (this: { survey?: SurveyModel }, params: unknown[]) {
    const q = this.survey?.getQuestionByName(String(params[0]));
    if (!q || q.isEmpty()) return "";
    return bandFor(q.getPropertyValue("bands"), q.value)?.id ?? "";
  });
  FunctionFactory.Instance.register("bmi", (params: unknown[]) => {
    const [h, w] = params.map(Number);
    return h > 0 && w > 0 ? Math.round((w / ((h / 100) * (h / 100))) * 10) / 10 : null;
  });
}

/** Shows a Calculation's band after its value, e.g. "5 (High)" (CAL-02). Call once per survey model. */
export function showBands(model: SurveyModel): void {
  model.onGetExpressionDisplayValue.add((_, o) => {
    const band = bandFor(o.question.getPropertyValue("bands"), o.value);
    if (band) o.displayValue = `${o.displayValue} (${band.label})`;
  });
}
