// Spike 2 follow-up: an exclusive regular choice as "None of these" keeps custom properties.
import { Model, Serializer } from "survey-core";
Serializer.addProperty("itemvalue", { name: "clinicalOutputs", visible: false, isLocalizable: false });
const out = [{ id: "out_1", note: { text: "No conditions", category: "Unassigned" } }];
const m = new Model({ elements: [{ type: "checkbox", name: "q_c", choices: [
  { value: "o_asth", text: "Asthma" },
  { value: "o_none", text: "None of these", isExclusive: true, clinicalOutputs: out },
  { value: "o_dk", text: "Don't know", isExclusive: true } ] }] });
const q: any = m.getQuestionByName("q_c");
q.value = ["o_asth"];
q.clickItemHandler(q.choices[1], true);
console.log(`${JSON.stringify(q.value) === '["o_none"]' ? "PASS" : "FAIL"}  selecting exclusive clears others -> ${JSON.stringify(q.value)}`);
q.clickItemHandler(q.choices[0], true);
console.log(`${JSON.stringify(q.value) === '["o_asth"]' ? "PASS" : "FAIL"}  selecting another clears exclusive -> ${JSON.stringify(q.value)}`);
const saved = JSON.stringify(m.toJSON());
console.log(`${saved.includes('"isExclusive":true') && saved.includes("No conditions") ? "PASS" : "FAIL"}  exclusive choice round-trips with clinicalOutputs`);
console.log(`class of exclusive item: ${q.choices[1].getType()}`);
