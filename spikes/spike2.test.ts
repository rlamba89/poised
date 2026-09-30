// Spike 2: clinicalOutputs set from our React modal marks Creator modified,
// autosaves, and undo/redo work. Run with the spike app on :3100.
import { chromium, Page } from "playwright";

const results: { check: string; pass: boolean; detail?: string }[] = [];
const check = (check: string, pass: boolean, detail?: unknown) =>
  results.push({ check, pass, detail: detail === undefined ? undefined : JSON.stringify(detail) });

const S = (page: Page, fn: string) => page.evaluate(`(() => { const s = window.__spike; ${fn} })()`) as Promise<any>;
const q = (name: string) => `s.creator.survey.getQuestionByName("${name}")`;

(async () => {
  const browser = await chromium.launch({ executablePath: process.env.CHROME_PATH });
  const page = await browser.newPage({ viewport: { width: 1500, height: 1100 } });
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto("http://localhost:3100/spike2" + (process.env.FIX ? "?fix=1" : ""));
  console.log("variant:", process.env.FIX ? "fix=1 (boxed value)" : "as planned (plain array)");
  await page.waitForSelector(".svc-creator");
  await page.waitForTimeout(800);

  check("initial state is clean", (await S(page, "return s.creator.state")) === "", await S(page, "return s.creator.state"));

  // Select the question on the canvas so its adorner actions show.
  await page.getByText("Do you smoke?", { exact: true }).first().click();
  await page.waitForTimeout(400);
  const badge = page.locator(".svc-question__content", { hasText: "Do you smoke?" }).getByText(/^Outputs \(\d+\)$/);
  check("adorner 'Outputs (0)' visible on selected question", (await badge.count()) > 0 && (await badge.first().innerText()) === "Outputs (0)",
    await badge.allInnerTexts());
  await page.screenshot({ path: "shots/spike2-selected.png" });

  // Open our modal from the adorner and add an output to option "Yes" (row 1).
  await badge.first().click();
  await page.getByTestId("outputs-modal").waitFor();
  await page.getByTestId("add-1").click();
  const stateAfterAdd = await S(page, "return s.creator.state");
  check("state becomes modified/saving after setPropertyValue", ["modified", "saving"].includes(stateAfterAdd), stateAfterAdd);
  check("onModified raised PROPERTY_CHANGED clinicalOutputs",
    (await S(page, "return s.events")).some((e: any) => e.type === "PROPERTY_CHANGED" && e.name === "clinicalOutputs"),
    await S(page, "return s.events"));
  await page.waitForTimeout(1000);
  const saves1 = await S(page, "return s.saves");
  check("autosave called saveSurveyFunc", saves1.length === 1, saves1.length);
  const yesOutputsSaved = saves1.at(-1)?.json.pages[0].elements[0].choices[0].clinicalOutputs;
  check("saved JSON contains outputs on option Yes", Array.isArray(yesOutputsSaved) && yesOutputsSaved.length === 1, yesOutputsSaved);
  check("state is 'saved' after callback", (await S(page, "return s.creator.state")) === "saved", await S(page, "return s.creator.state"));

  // Second output on the same option, plus one on the question itself.
  await page.getByTestId("add-1").click();
  await page.getByTestId("add-0").click();
  await page.getByTestId("close-modal").click();
  await page.waitForTimeout(1000);
  check("badge title updates live to 'Outputs (3)'", (await badge.first().innerText()) === "Outputs (3)", await badge.allInnerTexts());

  // Undo via the Creator's own toolbar button.
  const undoBtn = page.locator('[title="Undo"]').first();
  await undoBtn.click();
  await page.waitForTimeout(200);
  check("undo removes last change (question-level output)", (await S(page, `return s.outputsOf(${q("q_smoke1")}).length`)) === 0);
  await undoBtn.click();
  await page.waitForTimeout(200);
  check("second undo restores option to 1 output", (await S(page, `return s.outputsOf(${q("q_smoke1")}.choices[0]).length`)) === 1);
  await page.waitForTimeout(1000);
  check("badge reflects undo: 'Outputs (1)'", (await badge.first().innerText()) === "Outputs (1)", await badge.allInnerTexts());
  const savesAfterUndo = await S(page, "return s.saves");
  const lastSavedYes = savesAfterUndo.at(-1).json.pages[0].elements[0].choices[0].clinicalOutputs;
  check("undo triggers autosave with the undone JSON", savesAfterUndo.length > saves1.length && lastSavedYes.length === 1,
    { saves: savesAfterUndo.length, lastSavedYes });

  const redoBtn = page.locator('[title="Redo"]').first();
  const undoState = () => S(page, "return { canUndo: s.creator.undoRedoManager?.canUndo?.(), canRedo: s.creator.undoRedoManager?.canRedo?.() }");
  const beforeRedo = await undoState();
  await redoBtn.click();
  await page.waitForTimeout(300);
  const afterRedo1 = await undoState();
  const afterRedo1Counts = await S(page, `return [s.outputsOf(${q("q_smoke1")}.choices[0]).length, s.outputsOf(${q("q_smoke1")}).length]`);
  await redoBtn.click();
  await page.waitForTimeout(1000);
  console.log("redo debug:", JSON.stringify({ beforeRedo, afterRedo1, afterRedo1Counts, afterRedo2: await undoState() }));
  check("redo restores both changes", (await S(page, `return [s.outputsOf(${q("q_smoke1")}.choices[0]).length, s.outputsOf(${q("q_smoke1")}).length]`)).join() === "2,1");
  check("badge reflects redo: 'Outputs (3)'", (await badge.first().innerText()) === "Outputs (3)", await badge.allInnerTexts());

  // "None of these" item on a checkbox question.
  await page.getByText("Which conditions?", { exact: true }).first().click();
  await page.waitForTimeout(300);
  await page.locator(".svc-question__content", { hasText: "Which conditions?" }).getByText(/^Outputs \(\d+\)$/).first().click();
  await page.getByTestId("outputs-modal").waitFor();
  const noneRowLabel = await page.getByTestId("row-2").innerText();
  await page.getByTestId("add-2").click();
  await page.getByTestId("close-modal").click();
  await page.waitForTimeout(1000);
  const json = await S(page, "return s.creator.JSON");
  const cond = json.pages[0].elements[2];
  check("exclusive None-of-these option output is serialized into the chapter JSON", JSON.stringify(cond).includes("clinicalOutputs"), { noneRowLabel, cond });

  // JSON round trip: a fresh runtime survey sees the same outputs.
  const roundTrip = await page.evaluate((j) => {
    const { Model } = (window as any).__survey ?? {};
    return j;
  }, json);
  const creator2 = await S(page, `const c = new s.creator.constructor({}); c.JSON = s.creator.JSON;
    const qq = c.survey.getQuestionByName("q_smoke1");
    return { yes: s.outputsOf(qq.choices[0]).length, q: s.outputsOf(qq).length, none: s.outputsOf(c.survey.getQuestionByName("q_condit").choices[1]).length }`);
  check("JSON reload restores outputs (option, question, None-of-these)", creator2.yes === 2 && creator2.q === 1 && creator2.none === 1, creator2);
  void roundTrip;

  // Logic tab shows titles, not IDs (useElementTitles).
  await page.getByText("Logic", { exact: true }).first().click();
  await page.waitForTimeout(800);
  const logicText = await page.locator(".svc-creator-tab").last().innerText();
  await page.screenshot({ path: "shots/spike2-logic.png" });
  check("Logic tab shows question titles, not IDs", logicText.includes("Do you smoke?") && !logicText.includes("q_smoke1"),
    logicText.replace(/\s+/g, " ").slice(0, 300));

  check("no page errors", errors.length === 0, errors);

  let failed = 0;
  for (const r of results) {
    if (!r.pass) failed++;
    console.log(`${r.pass ? "PASS" : "FAIL"}  ${r.check}${r.detail && !r.pass ? `  ${r.detail.slice(0, 600)}` : ""}`);
  }
  await browser.close();
  process.exit(failed ? 1 : 0);
})();
