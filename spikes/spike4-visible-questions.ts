// Spike 4: does model.getAllQuestions(true) exclude questions inside hidden panels?
// Run: npx tsx spike4-visible-questions.ts
import { Model } from "survey-core";

const json = {
  pages: [
    {
      name: "p1",
      elements: [
        { type: "boolean", name: "smoke", title: "Do you smoke?" },
        {
          type: "panel",
          name: "smokingPanel",
          visibleIf: "{smoke} = true",
          elements: [
            { type: "text", inputType: "number", name: "perDay", title: "How many per day?" },
            {
              type: "panel",
              name: "nestedPanel",
              elements: [{ type: "text", name: "nestedQ", title: "Nested question" }],
            },
          ],
        },
        { type: "text", name: "directlyHidden", visibleIf: "{smoke} = true" },
        { type: "text", name: "alwaysVisible" },
      ],
    },
    {
      name: "p2",
      visibleIf: "{smoke} = true",
      elements: [{ type: "text", name: "onHiddenPage" }],
    },
  ],
};

const results: { check: string; pass: boolean; detail?: string }[] = [];
const check = (name: string, pass: boolean, detail?: string) => results.push({ check: name, pass, detail });
const visibleNames = (m: Model) => m.getAllQuestions(true).map((q) => q.name).sort();

const m = new Model(json);
// Answer smoke = true first so hidden questions get values, then hide them.
m.setValue("smoke", true);
m.setValue("perDay", 20);
m.setValue("nestedQ", "x");
m.setValue("directlyHidden", "y");
m.setValue("onHiddenPage", "z");
check(
  "smoke=true: all questions visible",
  JSON.stringify(visibleNames(m)) ===
    JSON.stringify(["alwaysVisible", "directlyHidden", "nestedQ", "onHiddenPage", "perDay", "smoke"]),
  visibleNames(m).join(","),
);

m.setValue("smoke", false);
const vis = visibleNames(m);
check("smoke=false: question in hidden panel excluded", !vis.includes("perDay"), vis.join(","));
check("smoke=false: question in nested panel of hidden panel excluded", !vis.includes("nestedQ"));
check("smoke=false: directly hidden question excluded", !vis.includes("directlyHidden"));
check("smoke=false: question on hidden page excluded", !vis.includes("onHiddenPage"));
check("smoke=false: unrelated question still included", vis.includes("alwaysVisible") && vis.includes("smoke"));

// Known trap from plan 12.2, recorded for the record (not a pass/fail of the spike).
const perDay = m.getQuestionByName("perDay");
console.log(`note: perDay.isVisible inside hidden panel = ${perDay.isVisible}, isParentVisible = ${perDay.isParentVisible}`);
console.log(`note: survey.data after hiding = ${JSON.stringify(m.data)}`);

let failed = 0;
for (const r of results) {
  if (!r.pass) failed++;
  console.log(`${r.pass ? "PASS" : "FAIL"}  ${r.check}${r.detail ? `  [${r.detail}]` : ""}`);
}
process.exit(failed ? 1 : 0);
