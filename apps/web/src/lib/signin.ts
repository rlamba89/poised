"use client";
// Staff sign-in with Cognito's managed login (OAuth code flow with PKCE), and sign-out.
import { api } from "./api";

/** What the API says about sign-in here (GET /api/auth/config). */
export type CognitoLogin = { authorizeUrl: string; logoutUrl: string; clientId: string; redirectUri: string; logoutRedirectUri: string };
export type AuthConfig = { cognito: CognitoLogin | null; devLogin: boolean };

const VERIFIER = "poised.pkce.verifier";
const STATE = "poised.pkce.state";

function base64url(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function randomString(bytes: number): string {
  return base64url(crypto.getRandomValues(new Uint8Array(bytes)));
}

/** Sends the browser to Cognito's managed login. The PKCE verifier and the state wait in
 * sessionStorage (this tab only) for the callback page. */
export async function startCognitoSignIn(login: CognitoLogin): Promise<void> {
  const verifier = randomString(32);
  const state = randomString(16);
  const challenge = base64url(new Uint8Array(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier))));
  sessionStorage.setItem(VERIFIER, verifier);
  sessionStorage.setItem(STATE, state);
  const q = new URLSearchParams({
    response_type: "code", client_id: login.clientId, redirect_uri: login.redirectUri,
    scope: "openid email profile", state, code_challenge: challenge, code_challenge_method: "S256",
  });
  window.location.assign(`${login.authorizeUrl}?${q}`);
}

/** Finishes a sign-in on /auth/callback: checks the state, then gives the code and verifier to the API. */
export async function finishCognitoSignIn(params: URLSearchParams): Promise<void> {
  const verifier = sessionStorage.getItem(VERIFIER);
  const state = sessionStorage.getItem(STATE);
  sessionStorage.removeItem(VERIFIER);
  sessionStorage.removeItem(STATE);
  const problem = params.get("error_description") ?? params.get("error");
  if (problem) throw new Error(problem);
  const code = params.get("code");
  if (!code || !verifier || params.get("state") !== state) throw new Error("This sign-in has expired. Please sign in again.");
  await api("/auth/callback", { method: "POST", body: JSON.stringify({ code, verifier }) });
}

/** Ends the session and, with Cognito, its managed-login session too, so the next sign-in asks
 * for the password again. */
export async function signOut(): Promise<void> {
  await api("/logout", { method: "POST" });
  const cfg = await api<AuthConfig>("/auth/config").catch(() => null);
  if (cfg?.cognito) {
    const q = new URLSearchParams({ client_id: cfg.cognito.clientId, logout_uri: cfg.cognito.logoutRedirectUri });
    window.location.assign(`${cfg.cognito.logoutUrl}?${q}`);
    return;
  }
  window.location.assign("/login");
}
