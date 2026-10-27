// Builds the evaluate service into one file, dist/evaluate.cjs (S41; S45's Lambda builds the same
// way). The evaluator's version is the git commit it was built from, "-dirty" with uncommitted
// changes, so a stored result names the code that worked it out.
import { execSync } from "node:child_process";
import { build } from "esbuild";

const git = (cmd) => {
  try {
    return execSync(`git ${cmd}`, { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
  } catch {
    return "";
  }
};
const commit = git("rev-parse --short=12 HEAD") || "unknown";
const dirty = git("status --porcelain -- :/packages/clinical :/apps/evaluate") ? "-dirty" : "";
const version = `clinical@${commit}${dirty}`;

await build({
  entryPoints: ["src/main.ts"],
  outfile: "dist/evaluate.cjs",
  bundle: true,
  platform: "node",
  target: "node22",
  format: "cjs",
  define: { __EVALUATOR_VERSION__: JSON.stringify(version) },
  logLevel: "warning",
});
console.log(`built dist/evaluate.cjs (evaluator ${version})`);
