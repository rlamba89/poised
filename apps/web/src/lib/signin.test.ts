import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { authorizeUrl, blankPage, challengeOf, finishSignIn, logoutUrl, refuseSignIn, startSignIn, clearRefusal, refusal, type Cognito } from "./signin";

const cognito: Cognito = {
  authorizeUrl: "https://poised.auth.example.test/oauth2/authorize",
  logoutUrl: "https://poised.auth.example.test/logout",
  clientId: "client1",
  redirectUri: "https://app.example.test/auth/callback",
  signedOutUri: "https://app.example.test/login",
};

// sessionStorage, as a browser tab has it (Vitest runs in Node).
function fakeSession() {
  const m = new Map<string, string>();
  vi.stubGlobal("sessionStorage", {
    getItem: (k: string) => m.get(k) ?? null,
    setItem: (k: string, v: string) => void m.set(k, v),
    removeItem: (k: string) => void m.delete(k),
  });
  return m;
}

beforeEach(() => {
  fakeSession();
});
afterEach(() => vi.unstubAllGlobals());

describe("PKCE (RFC 7636)", () => {
  it("the challenge is the verifier's SHA-256, base64url without padding", async () => {
    // RFC 7636 appendix B.
    expect(await challengeOf("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")).toBe("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM");
  });
});

describe("startSignIn", () => {
  it("keeps a new verifier and state for this tab, and sends the browser to the managed login", async () => {
    const go = vi.fn();
    await startSignIn(cognito, go);
    const url = new URL(go.mock.calls[0][0]);
    expect(url.origin + url.pathname).toBe(cognito.authorizeUrl);
    const q = url.searchParams;
    expect(q.get("response_type")).toBe("code");
    expect(q.get("client_id")).toBe("client1");
    expect(q.get("redirect_uri")).toBe(cognito.redirectUri);
    expect(q.get("scope")).toBe("openid email");
    expect(q.get("code_challenge_method")).toBe("S256");
    const kept = JSON.parse(sessionStorage.getItem("poised.signin")!);
    expect(kept.verifier).toMatch(/^[A-Za-z0-9_-]{43}$/);
    expect(q.get("state")).toBe(kept.state);
    expect(q.get("code_challenge")).toBe(await challengeOf(kept.verifier));
  });

  it("makes a new verifier and state every time", async () => {
    const go = vi.fn();
    await startSignIn(cognito, go);
    const first = sessionStorage.getItem("poised.signin");
    await startSignIn(cognito, go);
    expect(sessionStorage.getItem("poised.signin")).not.toBe(first);
  });
});

describe("finishSignIn", () => {
  async function started() {
    const go = vi.fn();
    await startSignIn(cognito, go);
    return new URL(go.mock.calls[0][0]).searchParams.get("state")!;
  }

  it("gives the code and this tab's verifier back, once", async () => {
    const state = await started();
    const { verifier } = JSON.parse(sessionStorage.getItem("poised.signin")!);
    expect(finishSignIn(`?code=abc&state=${state}`)).toEqual({ code: "abc", verifier });
    expect(finishSignIn(`?code=abc&state=${state}`)).toEqual({ error: expect.stringContaining("sign in again") });
  });

  it("refuses a state this tab didn't start (another site's link, or another tab)", async () => {
    await started();
    expect(finishSignIn("?code=abc&state=forged")).toEqual({ error: expect.stringContaining("sign in again") });
  });

  it("refuses a callback with no sign-in started here", () => {
    expect(finishSignIn("?code=abc&state=s")).toEqual({ error: expect.stringContaining("sign in again") });
  });

  it("passes on the managed login's own error", async () => {
    const state = await started();
    expect(finishSignIn(`?error=access_denied&error_description=Nope&state=${state}`)).toEqual({ error: expect.stringContaining("Nope") });
  });
});

describe("logoutUrl", () => {
  it("signs out of the managed login too, then comes back to the sign-in page", () => {
    const url = new URL(logoutUrl(cognito));
    expect(url.origin + url.pathname).toBe(cognito.logoutUrl);
    expect(url.searchParams.get("client_id")).toBe("client1");
    expect(url.searchParams.get("logout_uri")).toBe(cognito.signedOutUri);
  });
});

describe("authorizeUrl", () => {
  it("asks for nothing beyond the sign-in", () => {
    const url = new URL(authorizeUrl(cognito, "st", "ch"));
    expect([...url.searchParams.keys()].sort()).toEqual(
      ["client_id", "code_challenge", "code_challenge_method", "redirect_uri", "response_type", "scope", "state"].sort(),
    );
  });
});

describe("a sign-in Poised refuses (S12 test report 1, finding 2)", () => {
  it("ends the managed login's session too, and keeps the message for /login", () => {
    const go = vi.fn();
    refuseSignIn("Your trust is suspended.", cognito, go);
    const url = new URL(go.mock.calls[0][0]);
    expect(url.origin + url.pathname).toBe(cognito.logoutUrl);
    expect(url.searchParams.get("logout_uri")).toBe(cognito.signedOutUri);
    expect(refusal()).toBe("Your trust is suspended.");
    clearRefusal();
    expect(refusal()).toBe(""); // shown once
  });

  it("without Cognito (the dev login), nothing to end: the message is still kept", () => {
    const go = vi.fn();
    refuseSignIn("No.", null, go);
    expect(go).not.toHaveBeenCalled();
    expect(refusal()).toBe("No.");
  });
});

describe("leaving the signed-in page (shared workstations, S12 test report 1, finding 1)", () => {
  it("empties the page before going to the managed login's sign-out, so Back can't show it", () => {
    const root = { replaceChildren: vi.fn() };
    vi.stubGlobal("document", { getElementById: (id: string) => (id === "root" ? root : null) });
    blankPage();
    expect(root.replaceChildren).toHaveBeenCalledWith();
  });
});
