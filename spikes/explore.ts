import { chromium } from "playwright";
const page_ = process.argv[2] ?? "spike1";
(async () => {
  const browser = await chromium.launch({ executablePath: process.env.CHROME_PATH });
  const page = await browser.newPage({ viewport: { width: 1500, height: 950 } });
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push("pageerror: " + e.message));
  page.on("console", (m) => { if (m.type() === "error" || m.type() === "warning") errors.push(m.type() + ": " + m.text().slice(0, 200)); });
  await page.goto(`http://localhost:3100/${page_}`);
  await page.waitForSelector(".svc-creator", { timeout: 20000 });
  await page.waitForTimeout(1500);
  await page.screenshot({ path: `shots/${page_}.png` });
  console.log("errors:", errors);
  console.log("tabs:", await page.locator(".svc-tabbed-menu-item").allInnerTexts());
  console.log("toolbox:", (await page.locator(".svc-toolbox__item").allInnerTexts()).slice(0, 25));
  const html = await (await fetch(`http://localhost:3100/${page_}`)).text();
  console.log("SSR html contains svc-creator:", html.includes("svc-creator"));
  await browser.close();
})();
