// The texts and emails that didn't reach the patient (S17, OPS-05): GET …/episodes/{eid} lists
// them as "undelivered", since the latest invite or Resend. The reason (failure) comes from the
// API in words for staff.

export type Undelivered = {
  kind: "invite" | "reminder";
  channel: "sms" | "email";
  sentAt: string;
  failedAt: string;
  failure: string;
};

/** One line for staff, e.g. "The invite's text message wasn't delivered: the mobile number isn't valid." */
export function undeliveredText(u: Undelivered): string {
  const which = u.kind === "reminder" ? "A reminder's" : "The invite's";
  const what = u.channel === "sms" ? "text message" : "email";
  return `${which} ${what} wasn't delivered: ${u.failure}.`;
}
