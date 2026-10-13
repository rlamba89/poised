// The patient's links (S14, S15): an invite (/i/:token) opens one questionnaire, a sign-in link
// (/s/:token) opens their home. Both take Continue, then the date of birth. When the link itself
// can't be used, the page stops with a card saying why, and, for a sign-in link, how to get a new one.
import { ApiError } from "@/lib/api";

export type LinkKind = "invite" | "signin";

export type Stopped = { title: string; message: string; newLink: boolean };

/** Why a link can't be used (locked, used or expired, unknown, or the hospital suspended), or null for any other error. */
export function linkStopped(e: unknown, kind: LinkKind): Stopped | null {
  if (!(e instanceof ApiError)) return null;
  const code = (e.body as { code?: string } | null)?.code;
  const newLink = kind === "signin";
  if (code === "locked") return { title: "This link is locked", message: e.message, newLink };
  if (code === "expired") return { title: "This link has expired", message: e.message, newLink };
  if (e.status === 404) return { title: "This link doesn't work", message: e.message, newLink };
  if (e.status === 403) return { title: "Not available at the moment", message: e.message, newLink };
  return null;
}

/** Whether a page is one of the patient's (never send them to the staff sign-in). */
export function isPatientPage(pathname: string): boolean {
  return /^\/(home|signin)(\/|$)/.test(pathname) || /^\/[isv]\//.test(pathname);
}
