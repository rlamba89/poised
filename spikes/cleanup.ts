// Deletes the questionnaires and option lists the browser checks create ("Features test …", "Lung conditions <timestamp>").
import { chromium } from "playwright";
(async () => {
  const b = await chromium.launch({ executablePath: process.env.CHROME_PATH });
  const page = await b.newPage();
  await page.goto("http://localhost:3000/");
  await page.getByRole("button", { name: /Alex Author/ }).click();
  await page.waitForURL("**/questionnaires");
  const me = await (await page.request.get("http://localhost:3000/api/me")).json();
  const hid = me.hospitals[0].id;
  const raw = await (await page.request.get(`http://localhost:3000/api/h/${hid}/questionnaires`)).json();
  const list = (Array.isArray(raw) ? raw : Object.values(raw).find(Array.isArray)) as { id: string; name: string }[];
  for (const q of list.filter((x) => /^(Features test|Editor test|Dbg) /.test(x.name))) {
    const r = await page.request.delete(`http://localhost:3000/api/h/${hid}/questionnaires/${q.id}`);
    console.log(r.status(), q.name);
  }
  console.log("left:", list.filter((x) => !/^(Features test|Editor test|Dbg) /.test(x.name)).map((x) => x.name));
  const lists = (await (await page.request.get(`http://localhost:3000/api/h/${hid}/option-lists`)).json()) as { id: string; name: string }[];
  for (const l of lists.filter((x) => /^Lung conditions \d+$/.test(x.name))) {
    const r = await page.request.delete(`http://localhost:3000/api/h/${hid}/option-lists/${l.id}`);
    console.log(r.status(), "option list", l.name);
  }
  await b.close();
})();
