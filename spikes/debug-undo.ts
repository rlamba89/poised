import { chromium } from "playwright";
const code = `(() => {
  const s = window.__spike;
  const m = s.creator.undoRedoManager;
  const q = s.creator.survey.getQuestionByName("q_smoke1");
  const cnt = () => [s.outputsOf(q.choices[0]).length, s.outputsOf(q).length].join(",");
  const dump = () => m._transactions.map(t => t.actions.map(a => a._sender.getType() + ":" + a._propertyName + ":" + JSON.stringify(a._oldValue) + "->" + JSON.stringify(a._newValue)).join(" | "));
  const set = (o, v) => s.setOutputs(o, v);
  const out = { propType: s.creator.survey.getQuestionByName("q_smoke1").getPropertyByName("clinicalOutputs").type };
  set(q.choices[0], [{ id: "a" }]);
  set(q.choices[0], [{ id: "a" }, { id: "b" }]);
  set(q, [{ id: "c" }]);
  out.transactions = dump(); out.counts = [cnt()];
  const st = (l) => out.counts.push(l + " " + cnt() + " idx=" + m._currentTransactionIndex + " n=" + m._transactions.length + " qOwn=" + Object.prototype.hasOwnProperty.call(q, "clinicalOutputs"));
  m.undo(); st("undo");
  m.undo(); st("undo");
  m.redo(); st("redo");
  const t = m._transactions[1]; out.t1 = t && t.actions.map(a => [a._sender === q, JSON.stringify(a._newValue)]);
  m.redo(); st("redo");
  
  return out;
})()`;
(async () => {
  const browser = await chromium.launch({ executablePath: process.env.CHROME_PATH });
  const page = await browser.newPage();
  await page.goto("http://localhost:3100/spike2" + (process.env.FIX ? "?fix=1" : ""));
  await page.waitForSelector(".svc-creator");
  console.log(JSON.stringify(await page.evaluate(code), null, 1));
  await browser.close();
})();
