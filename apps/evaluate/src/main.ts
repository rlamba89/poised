// npm run evaluate: builds this (build.mjs) and runs it. The port is EVALUATE_URL's, the address the
// Go API calls (default http://127.0.0.1:8090); the version is the build's (the git commit).
import { portOf } from "./handler";
import { startServer } from "./server";

declare const __EVALUATOR_VERSION__: string;
const version = typeof __EVALUATOR_VERSION__ === "string" ? __EVALUATOR_VERSION__ : "dev";
const port = portOf(process.env.EVALUATE_URL);

startServer({ port, version }).then(
  () => console.log(`evaluate listening on 127.0.0.1:${port} (evaluator ${version})`),
  (e: Error) => {
    console.error(`evaluate: can't listen on 127.0.0.1:${port}: ${e.message}`);
    process.exit(1);
  },
);
