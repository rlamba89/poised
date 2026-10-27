import type { AddressInfo } from "node:net";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import sent from "../../api/internal/evaluate/testdata/evaluate.json";
import publishing from "../../api/internal/evaluate/testdata/publish_problems.json";
import { handle, portOf } from "./handler";
import { startServer } from "./server";

let base = "";
let close: () => Promise<void>;

beforeAll(async () => {
  const srv = await startServer({ port: 0, version: "test" });
  base = `http://127.0.0.1:${(srv.address() as AddressInfo).port}`;
  close = () => new Promise((done) => srv.close(() => done()));
});
afterAll(() => close());

const post = (path: string, body: unknown) =>
  fetch(base + path, { method: "POST", headers: { "Content-Type": "application/json" }, body: typeof body === "string" ? body : JSON.stringify(body) });

describe("the evaluate service over HTTP (S41)", () => {
  it("POST /evaluate gives the shared fixture's result", async () => {
    const r = await post("/evaluate", sent.input);
    expect(r.status).toBe(200);
    expect(r.headers.get("content-type")).toBe("application/json");
    expect(await r.json()).toEqual(sent.output);
  });

  it.each(publishing.cases)("POST /publish-problems: $name", async (c) => {
    const r = await post("/publish-problems", c.input);
    expect(r.status).toBe(200);
    expect(await r.json()).toEqual(c.output);
  });

  it("GET /health answers ok, with the evaluator's version", async () => {
    const r = await fetch(base + "/health");
    expect(r.status).toBe(200);
    expect(await r.json()).toEqual({ ok: true, evaluator: "test" });
  });

  it("refuses JSON that isn't a sent HQ, and what isn't JSON (400)", async () => {
    expect((await post("/evaluate", "{not json")).status).toBe(400);
    const r = await post("/evaluate", { sets: "no" });
    expect(r.status).toBe(400);
    expect(await r.json()).toEqual({ error: "sets must be a list" });
  });

  it("answers 404 for another path and 405 for another method", async () => {
    expect((await post("/other", {})).status).toBe(404);
    expect((await fetch(base + "/evaluate")).status).toBe(405);
  });

  it("refuses a body over the limit (413)", async () => {
    expect((await post("/evaluate", "x".repeat(11 << 20))).status).toBe(413);
  });
});

describe("its error log (S41: it never logs answers)", () => {
  it("names only the form, when the work throws", async () => {
    const log = vi.spyOn(console, "error").mockImplementation(() => {});
    const boom = () => {
      throw new Error("q_secret answer Yes was bad");
    };
    const r = await handle("POST", "/evaluate", JSON.stringify(sent.input), "test", { evaluate: boom });
    expect(r.status).toBe(500);
    expect(r.body).toEqual({ error: "The checks couldn't be worked out." });
    const lines = log.mock.calls.map((c) => c.join(" ")).join("\n");
    expect(lines).toContain(sent.input.formId);
    expect(lines).not.toContain("q_secret");
    expect(lines).not.toContain("Yes");
    log.mockRestore();
  });
});

describe("the port", () => {
  it("comes from EVALUATE_URL, the address Go calls (one setting for both)", () => {
    expect(portOf("http://127.0.0.1:8090")).toBe(8090);
    expect(portOf("http://127.0.0.1:9123/")).toBe(9123);
    expect(portOf(undefined)).toBe(8090);
    expect(portOf("")).toBe(8090);
  });
});
