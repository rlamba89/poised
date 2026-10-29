import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, describe, expect, it, vi } from "vitest";
import sent from "../../api/internal/evaluate/testdata/evaluate.json";
import publishing from "../../api/internal/evaluate/testdata/publish_problems.json";
import reviewing from "../../api/internal/evaluate/testdata/shown.json";
import { makeHandler } from "./lambda";

// S45: the same operations as the HTTP service, as one Lambda invoke: {op, input} → {status, body}.
const handler = makeHandler("test");

afterEach(() => vi.restoreAllMocks());

describe("the evaluate Lambda (S45)", () => {
  it("op evaluate gives the shared fixture's result", async () => {
    expect(await handler({ op: "evaluate", input: sent.input })).toEqual({ status: 200, body: sent.output });
  });

  it.each(publishing.cases)("op publish-problems: $name", async (c) => {
    expect(await handler({ op: "publish-problems", input: c.input })).toEqual({ status: 200, body: c.output });
  });

  it("op shown gives the shared fixture's sets", async () => {
    expect(await handler({ op: "shown", input: reviewing.input })).toEqual({ status: 200, body: reviewing.output });
  });

  it("op health answers ok, with the evaluator's version", async () => {
    expect(await handler({ op: "health" })).toEqual({ status: 200, body: { ok: true, evaluator: "test" } });
  });

  it("refuses an unknown op, a missing one, and input that isn't a sent HQ (400)", async () => {
    vi.spyOn(console, "log").mockImplementation(() => {});
    expect((await handler({ op: "other", input: {} })).status).toBe(400);
    expect((await handler({ input: {} })).status).toBe(400);
    expect((await handler(null)).status).toBe(400);
    expect(await handler({ op: "evaluate", input: { sets: "no" } })).toEqual({ status: 400, body: { error: "sets must be a list" } });
  });
});

describe("its log (S45: never an answer, a name or a date of birth)", () => {
  it("writes one line a call: the op, the form's id, the status and the time taken", async () => {
    const log = vi.spyOn(console, "log").mockImplementation(() => {});
    await handler({ op: "evaluate", input: sent.input });
    expect(log).toHaveBeenCalledTimes(1);
    const line = log.mock.calls[0].join(" ");
    expect(line).toMatch(new RegExp(`^evaluate: evaluate form ${sent.input.formId}: 200 in \\d+ ms$`));
  });

  it("names no form for publish checks, and only a form's id-shaped id", async () => {
    const log = vi.spyOn(console, "log").mockImplementation(() => {});
    await handler({ op: "publish-problems", input: publishing.cases[0].input });
    await handler({ op: "shown", input: { ...reviewing.input, formId: "Jane Smith 1970-01-01" } });
    const lines = log.mock.calls.map((c) => c.join(" "));
    expect(lines[0]).toMatch(/^evaluate: publish-problems: 200 in \d+ ms$/);
    expect(lines[1]).toMatch(/^evaluate: shown form unknown: \d{3} in \d+ ms$/);
    expect(lines.join("\n")).not.toContain("Jane");
  });

  it("an error names only the form, never its answers", async () => {
    vi.spyOn(console, "log").mockImplementation(() => {});
    const err = vi.spyOn(console, "error").mockImplementation(() => {});
    const boom = () => {
      throw new Error("q_secret answer Yes was bad");
    };
    const r = await makeHandler("test", { evaluate: boom })({ op: "evaluate", input: sent.input });
    expect(r).toEqual({ status: 500, body: { error: "The checks couldn't be worked out." } });
    const lines = err.mock.calls.map((c) => c.join(" ")).join("\n");
    expect(lines).toContain(sent.input.formId);
    expect(lines).not.toContain("q_secret");
    expect(lines).not.toContain("Yes");
  });
});

describe("its bundle (build.mjs --lambda, which make build-lambdas runs)", () => {
  it("is one CommonJS file whose handler answers like the source, with the build's version", async () => {
    vi.spyOn(console, "log").mockImplementation(() => {});
    const out = mkdtempSync(join(tmpdir(), "evaluate-lambda-"));
    try {
      const built = execFileSync(process.execPath, ["build.mjs", "--lambda", out], { cwd: join(__dirname, ".."), encoding: "utf8" });
      expect(built).toContain("for node24"); // the Lambda's runtime (infra: nodejs24.x; Rahul, 28 Oct)
      const bundle = createRequire(import.meta.url)(join(out, "index.js")) as { handler: ReturnType<typeof makeHandler> };
      const health = await bundle.handler({ op: "health" });
      expect(health.status).toBe(200);
      expect((health.body as { evaluator: string }).evaluator).toMatch(/^clinical@/);
      expect(await bundle.handler({ op: "shown", input: reviewing.input })).toEqual({ status: 200, body: reviewing.output });
    } finally {
      rmSync(out, { recursive: true, force: true });
    }
  }, 30_000);
});
