// The texts and emails that didn't reach the patient (S17, OPS-05): GET …/episodes/{eid} lists
// them as "undelivered", since the latest invite or Resend. The reason (failure) comes from the
// API in words for staff. S21 adds the status messages (EPI-08), kind "status", with the status
// they were sent for.
import { STATUS_LABELS, type EpisodeStatus } from "./types";

export type Undelivered = {
  kind: "invite" | "reminder" | "status";
  /** For a status message, the status it was about; "" otherwise. */
  status: EpisodeStatus | "";
  channel: "sms" | "email";
  sentAt: string;
  failedAt: string;
  failure: string;
};

/** One line for staff, e.g. "The invite's text message wasn't delivered: the mobile number isn't valid." */
export function undeliveredText(u: Undelivered): string {
  const what = u.channel === "sms" ? "text message" : "email";
  if (u.kind === "status") return `The ${u.status ? STATUS_LABELS[u.status] : "status"} ${what} wasn't delivered: ${u.failure}.`;
  const which = u.kind === "reminder" ? "A reminder's" : "The invite's";
  return `${which} ${what} wasn't delivered: ${u.failure}.`;
}

/** The invite's and reminders' (the invite card's). */
export const inviteUndelivered = (all: Undelivered[]) => all.filter((u) => u.kind !== "status");

/** The status messages' (S21). */
export const statusUndelivered = (all: Undelivered[]) => all.filter((u) => u.kind === "status");
