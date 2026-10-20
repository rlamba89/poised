// Staff sign-in through Cognito's managed login (S12): the OAuth code flow with PKCE.
//
//   1. startSignIn makes a random verifier and state, keeps them in this tab's sessionStorage, and
//      sends the browser to the managed login with the verifier's hash (the "challenge").
//   2. The managed login (password, then the authenticator code) sends the browser back to
//      /auth/callback?code=…&state=….
//   3. finishSignIn checks the state is this tab's (so another site can't push a code in) and
//      returns the code with the verifier; the callback page posts both to POST /api/auth/callback,
//      where Go exchanges them for the ID token and sets our own session cookie.
// Only the Go API ever sees a token; the browser holds just the session cookie, as with the dev login.
import { staffSignIn } from "./config";

/** The managed login's addresses and our app client (from GET /api/config). */
export type Cognito = {
  authorizeUrl: string;
  logoutUrl: string;
  clientId: string;
  redirectUri: string;
  signedOutUri: string;
};

const KEY = "poised.signin";
const AGAIN = "That sign-in didn't finish. Please sign in again.";

/** 32 random bytes, base64url: a 43-character verifier or state. */
function random(): string {
  return base64url(crypto.getRandomValues(new Uint8Array(32)));
}

function base64url(bytes: Uint8Array): string {
  let s = "";
  for (const b of bytes) s += String.fromCharCode(b);
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/** The PKCE challenge for a verifier: its SHA-256, base64url (method S256). */
export async function challengeOf(verifier: string): Promise<string> {
  const hash = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier));
  return base64url(new Uint8Array(hash));
}

export function authorizeUrl(c: Cognito, state: string, challenge: string): string {
  const q = new URLSearchParams({
    response_type: "code",
    client_id: c.clientId,
    redirect_uri: c.redirectUri,
    scope: "openid email",
    state,
    code_challenge: challenge,
    code_challenge_method: "S256",
  });
  return `${c.authorizeUrl}?${q}`;
}

/** Sends the browser to the managed login. `go` is window.location.assign (tests pass a spy). */
export async function startSignIn(c: Cognito, go: (url: string) => void = (u) => window.location.assign(u)) {
  const verifier = random();
  const state = random();
  sessionStorage.setItem(KEY, JSON.stringify({ verifier, state }));
  go(authorizeUrl(c, state, await challengeOf(verifier)));
}

/** Reads the managed login's redirect (the page's query string): the code and this tab's verifier, or why not. */
export function finishSignIn(search: string): { code: string; verifier: string } | { error: string } {
  const q = new URLSearchParams(search);
  const kept = sessionStorage.getItem(KEY);
  sessionStorage.removeItem(KEY); // a sign-in is finished once
  let started: { verifier?: string; state?: string } = {};
  try {
    started = kept ? JSON.parse(kept) : {};
  } catch {
    started = {};
  }
  if (!started.state || q.get("state") !== started.state || !started.verifier) return { error: AGAIN };
  const error = q.get("error");
  if (error) return { error: `${q.get("error_description") || error}. Please sign in again.` };
  const code = q.get("code");
  return code ? { code, verifier: started.verifier } : { error: AGAIN };
}

/** Signing out of the managed login too (otherwise its own session would sign the next person straight in). */
export function logoutUrl(c: Cognito): string {
  return `${c.logoutUrl}?${new URLSearchParams({ client_id: c.clientId, logout_uri: c.signedOutUri })}`;
}

/**
 * Empties the page before the browser leaves it for the managed login's sign-out, so the copy the
 * browser keeps for Back (its back/forward cache) shows nothing of the signed-out person's work
 * (S12 test report 1, finding 1: shared hospital workstations). React's tree goes with it, which
 * is fine: the browser is leaving.
 */
export function blankPage() {
  document.getElementById("root")?.replaceChildren();
}

/**
 * Signs out: ends our session (POST /api/logout), then also the managed login's where staff sign
 * in with Cognito, which comes back to /login; with the dev login, `toLogin` goes there directly.
 */
export async function signOut(toLogin: () => void, post: () => Promise<unknown>) {
  await post().catch(() => undefined); // the cookie is cleared either way when the session ends
  const cognito = staffSignIn().cognito;
  if (cognito) {
    blankPage();
    window.location.assign(logoutUrl(cognito));
  } else toLogin();
}

const REFUSED = "poised.signin.refused";

/**
 * A sign-in Preopsis refused after the managed login let the person in (e.g. their trust is
 * suspended, or Preopsis has no account for them): end the managed login's session too, or the next
 * Sign in on this computer would go straight back in as them, with no way to sign in as someone
 * else (S12 test report 1, finding 2). The message waits in this tab for /login to show it
 * (takeRefusal). Without Cognito there's nothing to end, and the caller shows /login itself.
 */
export function refuseSignIn(message: string, c: Cognito | null, go: (url: string) => void = (u) => window.location.assign(u)) {
  sessionStorage.setItem(REFUSED, message);
  if (c) go(logoutUrl(c));
}

/** The message of a refused sign-in waiting for /login ("" if none). */
export const refusal = (): string => sessionStorage.getItem(REFUSED) ?? "";

/** /login has shown it: it isn't shown again. */
export const clearRefusal = () => sessionStorage.removeItem(REFUSED);
