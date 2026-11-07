// Writes each assessment's locked JSON for Go (apps/api/internal/assessment/<id>.json) from
// src/assessments.ts (S47), and the list of them, id and name in order (list.json, S48), so both
// sides hold one definition. assessments.test.ts fails until it's
// run after a change. Run: npm run assessments -w packages/clinical
import { build } from "esbuild";
import { writeFileSync } from "node:fs";

const src = new URL("../src/assessments.ts", import.meta.url).pathname;
const out = await build({ entryPoints: [src], bundle: true, write: false, format: "esm", platform: "node", logLevel: "error" });
const mod = await import(`data:text/javascript;base64,${Buffer.from(out.outputFiles[0].text).toString("base64")}`);
const write = (name, value) => {
  const file = new URL(`../../../apps/api/internal/assessment/${name}`, import.meta.url).pathname;
  writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`);
  console.log(`wrote ${file}`);
};
write("list.json", mod.ASSESSMENTS);
for (const { id } of mod.ASSESSMENTS) write(`${id}.json`, mod.assessmentContent(id));
