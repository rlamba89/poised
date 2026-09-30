// Spike 3: onQuestionAdded / onPanelAdded / onItemValueAdded set generated name/value
// without clobbering the default label, including on copy. Run with the spike app on :3100.
import { chromium, Page } from "playwright";

const results: { check: string; pass: boolean; detail?: string }[] = [];
const check = (check: string, pass: boolean, detail?: unknown) =>
  results.push({ check, pass, detail: detail === undefined ? undefined : JSON.stringify(detail) });
const S = (page: Page, body: string) => page.evaluate(`(() => { const s = window.__spike; ${body} })()`) as Promise<any>;
const questions = (page: Page) =>
  S(page, `return s.creator.survey.getAllQuestions().map(q => ({ name: q.name, title: q.title, type: q.getType(),
    choices: (q.choices || []).map(c => ({ value: c.value, text: c.text })) }))`);

(async () => {
  const browser = await chromium.launch({ executablePath: process.env.CHROME_PATH });
  const page = await browser.newPage({ viewport: { width: 1500, height: 1100 } });
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto("http://localhost:3100/spike3");
  await page.waitForSelector(".svc-creator");
  await page.waitForTimeout(500);

  // 1. Click a toolbox item.
  await page.locator(".svc-toolbox__item", { hasText: "Radio Button Group" }).first().click();
  await page.waitForTimeout(500);
  let qs = await questions(page);
  const q1 = qs[0];
  check("toolbox add: name is generated q_xxxxxx", /^q_[a-z0-9]{6}$/.test(q1?.name), q1);
  check("toolbox add: default label kept (title = question1)", q1?.title === "question1", q1);
  check("toolbox add: default choices get o_xxxx values, labels kept",
    q1?.choices.length === 3 && q1.choices.every((c: any, i: number) => /^o_[a-z0-9]{4}$/.test(c.value) && c.text === `Item ${i + 1}`), q1?.choices);
  const canvasText = await page.locator(".svc-question__content").first().innerText();
  check("toolbox add: canvas shows labels, not IDs", canvasText.includes("question1") && canvasText.includes("Item 1") && !canvasText.includes(q1?.name),
    canvasText.replace(/\s+/g, " ").slice(0, 200));

  // 2. Add a choice on the canvas with the "+" next to the placeholder item.
  await page.locator(".svc-question__content").first().click({ position: { x: 20, y: 20 } });
  await page.waitForTimeout(300);
  await page.locator(".svc-item-value--new .svc-item-value-controls__add").first().click();
  await page.waitForTimeout(400);
  qs = await questions(page);
  const added = qs[0].choices[3];
  check("canvas add choice: value generated o_xxxx", /^o_[a-z0-9]{4}$/.test(added?.value), qs[0].choices);
  check("canvas add choice: default label kept (Item 4)", added?.text === "Item 4", added);

  // 3. Duplicate the question with the adorner's Duplicate action.
  await page.locator(".svc-question__content").first().locator('.sv-action--duplicate button').first().click();
  await page.waitForTimeout(500);
  qs = await questions(page);
  const copy = qs[1];
  const copyEvent = (await S(page, "return s.events")).find((e: any) => e.ev === "questionAdded" && e.reason === "ELEMENT_COPIED");
  check("copy: onQuestionAdded fires with reason ELEMENT_COPIED", !!copyEvent, await S(page, "return s.events"));
  check("copy: new generated name, different from original", /^q_[a-z0-9]{6}$/.test(copy?.name) && copy.name !== q1.name, copy);
  check("copy: label kept (title = question1)", copy?.title === "question1", copy);
  check("copy: choices and labels copied", JSON.stringify(copy?.choices) === JSON.stringify(qs[0].choices), { orig: qs[0].choices, copy: copy?.choices });

  // 4. Panel from toolbox.
  await page.locator(".svc-toolbox__item", { hasText: /^Panel$/ }).first().click();
  await page.waitForTimeout(400);
  const panel = await S(page, `const p = s.creator.survey.getAllPanels()[0]; return p && { name: p.name, title: p.title }`);
  check("panel add: name generated g_xxxxxx, label kept (panel1)", /^g_[a-z0-9]{6}$/.test(panel?.name) && panel.title === "panel1", panel);

  // 5. Undo of the toolbox add removes the question in one step (IDs set inside the add transaction).
  const before = (await questions(page)).length;
  await page.locator('[title="Undo"]').first().click();
  await page.waitForTimeout(300);
  const afterPanelUndo = await S(page, "return s.creator.survey.getAllPanels().length");
  check("one undo removes the added panel (rename is not a separate undo step)", afterPanelUndo === 0, { afterPanelUndo, before });

  await page.screenshot({ path: "shots/spike3.png" });
  check("no page errors", errors.length === 0, errors);
  console.log("events:", JSON.stringify(await S(page, "return s.events")));
  let failed = 0;
  for (const r of results) {
    if (!r.pass) failed++;
    console.log(`${r.pass ? "PASS" : "FAIL"}  ${r.check}${r.detail && !r.pass ? `  ${r.detail.slice(0, 600)}` : ""}`);
  }
  await browser.close();
  process.exit(failed ? 1 : 0);
})();
