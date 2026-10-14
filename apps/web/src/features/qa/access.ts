// Why a tester's tool page can't load (S12). On QA the tools need a signed-in platform admin;
// locally (the dev login on) they're open. They exist only with MESSAGING=capture.
import { ApiError } from "@/lib/api";

export const SIGN_IN_FIRST = "Sign in as a platform admin to use this page.";

export function qaError(e: unknown, notCapture: string): string {
  if (e instanceof ApiError && e.status === 404) return notCapture;
  if (e instanceof ApiError && e.status === 401) return SIGN_IN_FIRST;
  if (e instanceof ApiError && e.status === 403) return "Only a platform admin can use this page.";
  return (e as Error).message;
}
