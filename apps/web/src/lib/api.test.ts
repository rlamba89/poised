import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, NETWORK_ERROR, api } from "./api";

afterEach(() => {
  vi.unstubAllGlobals();
});

const answer = (res: () => Response | Promise<Response>) => vi.stubGlobal("fetch", vi.fn(async () => res()));
const failure = (path = "/x") => api(path).then(() => null, (e: unknown) => e);

// S29 (Rahul, 25 Oct): one plain message when the server can't be reached, never the browser's
// own "Failed to fetch"; the API's own messages are kept.
describe("api", () => {
  it("turns a fetch that fails (offline, the API down) into the plain message", async () => {
    answer(() => Promise.reject(new TypeError("Failed to fetch")));
    const e = await failure();
    expect(e).toBeInstanceOf(ApiError);
    expect((e as ApiError).message).toBe(NETWORK_ERROR);
    expect((e as ApiError).status).toBe(0);
  });
  it("turns a gateway error with no JSON body into the plain message", async () => {
    for (const status of [500, 502, 503, 504]) {
      answer(() => new Response(status === 502 ? "" : "<html>Bad Gateway</html>", { status, headers: { "Content-Type": "text/html" } }));
      const e = (await failure()) as ApiError;
      expect(e.message).toBe(NETWORK_ERROR);
      expect(e.status).toBe(status);
    }
  });
  // S30: on AWS, API Gateway answers its own 5xx with JSON ({"message": …}), which isn't the API's.
  it("turns a gateway error whose JSON has no API error into the plain message", async () => {
    for (const status of [502, 503]) {
      answer(() => Response.json({ message: "Internal Server Error" }, { status }));
      const e = (await failure()) as ApiError;
      expect(e.message).toBe(NETWORK_ERROR);
      expect(e.status).toBe(status);
      expect(e.body).toEqual({ message: "Internal Server Error" });
    }
  });
  it("keeps the API's own message, a 500's included", async () => {
    answer(() => Response.json({ error: "Something went wrong on our side." }, { status: 500 }));
    expect(((await failure()) as ApiError).message).toBe("Something went wrong on our side.");
    answer(() => Response.json({ error: "Episode not found.", code: "x" }, { status: 404 }));
    const e = (await failure()) as ApiError;
    expect(e.message).toBe("Episode not found.");
    expect(e.body).toEqual({ error: "Episode not found.", code: "x" });
  });
  it("a 4xx with no JSON body, or JSON without an error, is still the general message", async () => {
    answer(() => new Response("nope", { status: 400 }));
    expect(((await failure()) as ApiError).message).toBe("Something went wrong. Please try again.");
    answer(() => Response.json({ message: "Forbidden" }, { status: 403 }));
    expect(((await failure()) as ApiError).message).toBe("Something went wrong. Please try again.");
  });
  it("keeps a 409's and a 422's own message", async () => {
    answer(() => Response.json({ error: "These answers were changed on another device." }, { status: 409 }));
    expect(((await failure()) as ApiError).message).toBe("These answers were changed on another device.");
    answer(() => Response.json({ error: "Enter a valid date of birth." }, { status: 422 }));
    expect(((await failure()) as ApiError).message).toBe("Enter a valid date of birth.");
  });
  it("returns the body, or null for a 204", async () => {
    answer(() => Response.json({ ok: 1 }));
    expect(await api("/x")).toEqual({ ok: 1 });
    answer(() => new Response(null, { status: 204 }));
    expect(await api("/x")).toBeNull();
  });
  it("lets an aborted request stay an abort", async () => {
    answer(() => Promise.reject(new DOMException("The operation was aborted.", "AbortError")));
    const e = await failure();
    expect((e as Error).name).toBe("AbortError");
  });
});
