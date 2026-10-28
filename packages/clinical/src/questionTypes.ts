// The incumbent's question types SurveyJS doesn't have, built from its own questions so the
// preview and the patient runtime render them (plan-redesign.md, step 5).
import { ComponentCollection, FunctionFactory, Serializer, settings, SurveyModel, type Question } from "survey-core";
import { ALL_ANSWERED, bandIn, bandWhenInputs, earlierPoints, earlierQuestion, inputAnswered, inputHidden, isGuardedScore, scoreAsShown } from "./scores";
import { registerVideoType } from "./video";
import { STOP_BANG_KEYS } from "./ready";

/** Registers Medication, Admissions, BMI, Profile and Video (S34), and the flags the editor stores. Safe to call twice. */
export function registerQuestionTypes(): void {
  if (ComponentCollection.Instance.getCustomQuestionByName("medication")) return;

  // A repeating list (incumbent "DynamicAdd"): each row is one medicine.
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
  // "Always ask fresh" (S23): never pre-filled from an earlier HQ. Answer-holding questions only.
  Serializer.addProperty("question", { name: "alwaysFresh:boolean", default: false, ...hidden });
  // "Same as…" (S24): where an adopted key came from ({ questionnaire, title }), for information only.
  Serializer.addProperty("question", { name: "linkedFrom", ...hidden });

  registerClinicalFunctions();
  registerVideoType();
  // Information (S35): the author's text with simple marks, translatable (information.ts).
  Serializer.addProperty("html", { name: "markdown:text", visible: false, isLocalizable: true });

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
 * - `allAnswered('q_a', 'patientAge', …)` (S38): whether every input of a score has an answer (one a
 *   condition hides counts, adding 0: S44a)
 * score() and band() also find questions in earlier Question Sets (scores.ts, applyEarlier).
 */
function registerClinicalFunctions(): void {
  // SurveyJS skips a Calculation whose `{…}` variables didn't change. A score names its questions as
  // text ('q_a'), so a score with a line testing {patientAge} would skip a change to q_a (S38 test
  // report 1, F-1). Calculations always run again instead, as before SurveyJS tracked them.
  settings.expressionQuestionTrackDependencies = false;
  // A score's "Add N points when…" line that tests a hidden input doesn't fire (S44a, scoreAsShown).
  // SurveyJS passes every expression through beforeExpressionRunning (the onExpressionRunning
  // event) just before it runs; doing it here, once, covers every survey any view builds.
  const proto = SurveyModel.prototype as unknown as { beforeExpressionRunning(obj: unknown, prop: string, expr: string): string };
  const before = proto.beforeExpressionRunning;
  proto.beforeExpressionRunning = function (this: SurveyModel, obj: unknown, prop: string, expr: string) {
    const e = before.call(this, obj, prop, expr);
    return prop === "expression" && e ? scoreAsShown(this, e) : e;
  };
  FunctionFactory.Instance.register("score", function (this: { survey?: SurveyModel }, params: unknown[]) {
    let total = 0;
    for (const name of params) {
      const q = this.survey?.getQuestionByName(String(name)) as (Question & { choices?: { value: unknown; score?: number }[] }) | null;
      if (!q) {
        const earlier = earlierQuestion(this.survey, String(name));
        if (earlier) total += earlierPoints(earlier.el, earlier.value);
        continue;
      }
      if (q.isEmpty() || inputHidden(this.survey, q.name)) continue; // hidden on purpose: 0 (S44a)
      const chosen: unknown[] = Array.isArray(q.value) ? q.value : [q.value];
      for (const c of q.choices ?? []) if (chosen.includes(c.value)) total += Number(c.score ?? 0);
    }
    return total;
  });
  FunctionFactory.Instance.register("band", function (this: { survey?: SurveyModel }, params: unknown[]) {
    const q = this.survey?.getQuestionByName(String(params[0]));
    if (!q) {
      const earlier = earlierQuestion(this.survey, String(params[0]));
      return (earlier && bandIn(this.survey, earlier.el.bands, earlier.value)?.id) ?? "";
    }
    if (q.isEmpty()) return "";
    return bandIn(this.survey, q.getPropertyValue("bands"), q.value)?.id ?? "";
  });
  FunctionFactory.Instance.register(ALL_ANSWERED, function (this: { survey?: SurveyModel; question?: Question }, params: unknown[]) {
    // A band's "also when" inputs count too (S44a): without them the band can't be decided.
    const q = this.question;
    const also = q && q.getType() === "expression" ? bandWhenInputs(q.name, q.getPropertyValue("bands")) : [];
    // STOP-Bang (S44c): a patient whose sex is "Other" makes it Incomplete, as "Not known" does.
    const strictSex = q?.name === STOP_BANG_KEYS.score;
    return [...params.map(String), ...also].every((name) => inputAnswered(this.survey, name, strictSex));
  });
  FunctionFactory.Instance.register("bmi", (params: unknown[]) => {
    const [h, w] = params.map(Number);
    return h > 0 && w > 0 ? Math.round((w / ((h / 100) * (h / 100))) * 10) / 10 : null;
  });
}

/**
 * Shows a Calculation's band after its value, e.g. "5 (High)" (CAL-02), and "Incomplete" for a score
 * with an input not answered yet (S38). Call once per survey model.
 */
export function showBands(model: SurveyModel): void {
  model.onGetExpressionDisplayValue.add((_, o) => {
    if ((o.value === null || o.value === undefined || o.value === "") && isGuardedScore(o.question.getPropertyValue("expression"))) {
      o.displayValue = INCOMPLETE;
      return;
    }
    const band = bandIn(model, o.question.getPropertyValue("bands"), o.value);
    if (band) o.displayValue = `${o.displayValue} (${band.label})`;
  });
}

/** What a score with a missing input shows (S38). */
export const INCOMPLETE = "Incomplete";
