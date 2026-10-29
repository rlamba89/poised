// Builds the evaluate service into one file with esbuild (S41), from the same sources two ways:
//   node build.mjs                 dist/evaluate.cjs: the local HTTP service (npm run evaluate)
//   node build.mjs --lambda <dir>  <dir>/index.js: the Lambda (S45; make build-lambdas writes
//                                  apps/api/dist/evaluate/, which infra's ApiStack deploys)
// The evaluator's version is the git commit it was built from, "-dirty" with uncommitted changes,
// so a stored result names the code that worked it out. In the pipeline (CodeBuild has no .git)
// it's the commit CodeBuild was given, CODEBUILD_RESOLVED_SOURCE_VERSION.
import { execSync } from "node:child_process";
import { join } from "node:path";
import { build } from "esbuild";

const git = (cmd) => {
  try {
    return execSync(`git ${cmd}`, { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
  } catch {
    return "";
  }
};
const commit = git("rev-parse --short=12 HEAD") || (process.env.CODEBUILD_RESOLVED_SOURCE_VERSION ?? "").slice(0, 12) || "unknown";
const dirty = git("status --porcelain -- :/packages/clinical :/apps/evaluate") ? "-dirty" : "";
const version = `clinical@${commit}${dirty}`;

const at = process.argv.indexOf("--lambda");
const lambda = at >= 0;
if (lambda && !process.argv[at + 1]) {
  console.error("usage: node build.mjs [--lambda <dir>]");
  process.exit(2);
}
const outfile = lambda ? join(process.argv[at + 1], "index.js") : "dist/evaluate.cjs";
// The Lambda runs on Node 24 (infra's nodejs24.x; Rahul, 28 Oct: AWS drops Node 22 Lambdas around
// April 2027); the local service runs on the repo's Node 22 until the repo moves to 24 before go-live.
const target = lambda ? "node24" : "node22";

await build({
  entryPoints: [lambda ? "src/lambda.ts" : "src/main.ts"],
  outfile,
  bundle: true,
  platform: "node",
  target,
  format: "cjs", // index.js with no package.json beside it is CommonJS to Node, as Lambda loads it
  define: { __EVALUATOR_VERSION__: JSON.stringify(version) },
  logLevel: "warning",
});
console.log(`built ${outfile} for ${target} (evaluator ${version})`);
