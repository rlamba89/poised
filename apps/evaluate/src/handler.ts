// The evaluate service's requests (plan C6, S41), apart from how they arrive: server.ts serves
// them over HTTP locally (npm run evaluate, through handle()); S45's Lambda (lambda.ts) calls run()
// with the same operations, so both give the same answers.
//
//   POST /evaluate          a sent HQ (EvaluateInput) → what it gives (EvaluateResult)
//   POST /publish-problems  a version's sets and rules (PublishInput) → { problems }
//   POST /shown             a reviewed HQ's sets and the answers that count now (ShownInput) → { shown } (S44)
//   GET  /health            { ok, evaluator }
//
// Nothing here ever logs an answer: an error log names only the episode form's id, the error's
// kind and where in the code it happened (not its message, which could quote a value).
import { checkPublish, evaluate, EvaluateInputError, shownSetsFor } from "@poised/clinical";

/** The largest body accepted, in bytes: a whole HQ's JSON with its answers fits many times over. */
export const MAX_BODY = 10 << 20;

/** Where Go calls the service by default (internal/config's EVALUATE_URL), so its port. */
export const DEFAULT_PORT = 8090;

export type Reply = { status: number; body: unknown };

/** The operations; a test swaps one for a failing one. */
export type Ops = { evaluate: typeof evaluate; checkPublish: typeof checkPublish; shownSetsFor: typeof shownSetsFor };
const real: Ops = { evaluate, checkPublish, shownSetsFor };

/** The port in EVALUATE_URL (the address Go calls), so one setting serves both sides. */
export function portOf(url: string | undefined): number {
  if (!url) return DEFAULT_PORT;
  const port = Number(new URL(url).port);
  return port > 0 ? port : DEFAULT_PORT;
}

/** The error's kind and its stack's code locations, without its message. */
function where(e: unknown): string {
  if (!(e instanceof Error)) return typeof e;
  const frames = (e.stack ?? "").split("\n").filter((l) => l.trim().startsWith("at ")).slice(0, 5).map((l) => l.trim());
  return [e.name, ...frames].join(" | ");
}

/** The operations, by name: the HTTP path without its slash, and the Lambda's op (S45). */
export const OP_NAMES = ["evaluate", "publish-problems", "shown"] as const;
export type OpName = (typeof OP_NAMES)[number];
export const isOpName = (v: unknown): v is OpName => typeof v === "string" && (OP_NAMES as readonly string[]).includes(v);

/** The id the logs may name for a request: the episode form's, when it looks like one; else "unknown". */
export const formIdOf = (v: unknown): string => {
  const id = v && typeof v === "object" ? (v as { formId?: unknown }).formId : undefined;
  return typeof id === "string" && /^[0-9a-f-]{36}$/i.test(id) ? id : "unknown";
};

/** Runs one operation on its parsed input. Its only log is the error line (no answer, no message). */
export function run(name: OpName, input: unknown, version: string, ops: Partial<Ops> = {}): Reply {
  const op = { ...real, ...ops };
  try {
    const out =
      name === "evaluate" ? op.evaluate(input as Parameters<Ops["evaluate"]>[0], version)
      : name === "shown" ? op.shownSetsFor(input as Parameters<Ops["shownSetsFor"]>[0])
      : op.checkPublish(input as Parameters<Ops["checkPublish"]>[0]);
    return { status: 200, body: out };
  } catch (e) {
    if (e instanceof EvaluateInputError) return { status: 400, body: { error: e.message } };
    const what = name === "publish-problems" ? "a version's publish checks" : `form ${formIdOf(input)}`;
    console.error(`evaluate: failed on ${what}: ${where(e)}`);
    return { status: 500, body: { error: "The checks couldn't be worked out." } };
  }
}

/** The health answer: the service is up, and which build it is. */
export const health = (version: string): Reply => ({ status: 200, body: { ok: true, evaluator: version } });

export async function handle(method: string, path: string, body: string, version: string, ops: Partial<Ops> = {}): Promise<Reply> {
  if (path === "/health") return method === "GET" ? health(version) : { status: 405, body: { error: "Use GET." } };
  const name = path.slice(1);
  if (!isOpName(name)) return { status: 404, body: { error: "Not found." } };
  if (method !== "POST") return { status: 405, body: { error: "Use POST." } };
  let input: unknown;
  try {
    input = JSON.parse(body);
  } catch {
    return { status: 400, body: { error: "The body isn't JSON." } };
  }
  return run(name, input, version, ops);
}
