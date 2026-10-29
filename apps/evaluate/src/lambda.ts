// The evaluate Lambda (plan C6, S45; D-1, D-4): the same operations the local HTTP service serves
// (handler.ts), as one synchronous invoke from the Go API (internal/evaluate's Lambda):
//
//   { "op": "evaluate",         "input": <a sent HQ> }               → { status: 200, body: <its result> }
//   { "op": "publish-problems", "input": <a version's sets, rules> }  → { status: 200, body: { problems } }
//   { "op": "shown",            "input": <an HQ under review> }      → { status: 200, body: { shown } }
//   { "op": "health" }                                               → { status: 200, body: { ok, evaluator } }
//
// The status is as the HTTP service's (400 for input that isn't what the op takes, 500 when the
// work throws); Go counts anything but 200 as a failure. It never throws, so Lambda never reports a
// function error with a message in it. No database, no network, no environment settings.
//
// Its log: one line a call, with the op, the episode form's id (when the input names one), the
// status and the milliseconds; and handler.ts's error line. Never an answer, a name or a date of birth.
import { formIdOf, health, isOpName, run, type Ops, type Reply } from "./handler";

declare const __EVALUATOR_VERSION__: string;

/** What Go sends (internal/evaluate's invoke payload). Anything else is refused with a 400. */
export type Event = { op?: unknown; input?: unknown } | null | undefined;

export function makeHandler(version: string, ops: Partial<Ops> = {}): (event: Event) => Promise<Reply> {
  return async (event) => {
    const start = performance.now();
    const op = event && typeof event === "object" ? event.op : undefined;
    const input = event && typeof event === "object" ? event.input : undefined;
    let reply: Reply;
    if (op === "health") reply = health(version);
    else if (!isOpName(op)) reply = { status: 400, body: { error: "op must be evaluate, publish-problems, shown or health." } };
    else reply = run(op, input, version, ops);
    const what = !isOpName(op) ? (op === "health" ? "health" : "unknown op") : op === "publish-problems" ? op : `${op} form ${formIdOf(input)}`;
    console.log(`evaluate: ${what}: ${reply.status} in ${Math.round(performance.now() - start)} ms`);
    return reply;
  };
}

/** The Lambda's handler (index.handler), with the build's version (build.mjs: the git commit). */
export const handler = makeHandler(typeof __EVALUATOR_VERSION__ === "string" ? __EVALUATOR_VERSION__ : "dev");
