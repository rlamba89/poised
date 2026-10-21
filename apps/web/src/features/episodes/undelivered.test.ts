import { describe, expect, it } from "vitest";
import { inviteUndelivered, statusUndelivered, undeliveredText, type Undelivered } from "./undelivered";

// An undelivered text or email (S17, OPS-05), as the episode page's invite card says it.
describe("undeliveredText", () => {
  const u = (kind: Undelivered["kind"], channel: Undelivered["channel"], failure: string, status: Undelivered["status"] = ""): Undelivered => ({
    kind, status, channel, failure, sentAt: "2026-10-14T09:02:00Z", failedAt: "2026-10-14T09:03:00Z",
  });
  it("names the message and says why", () => {
    expect(undeliveredText(u("invite", "sms", "the mobile number isn't valid"))).toBe(
      "The invite's text message wasn't delivered: the mobile number isn't valid.");
    expect(undeliveredText(u("reminder", "email", "it bounced"))).toBe("A reminder's email wasn't delivered: it bounced.");
  });
  it("names a status message by its status (S21)", () => {
    expect(undeliveredText(u("status", "sms", "the phone couldn't be reached", "ready_for_admission"))).toBe(
      "The Ready for admission text message wasn't delivered: the phone couldn't be reached.");
    expect(undeliveredText(u("status", "email", "it bounced", "cancelled"))).toBe("The Cancelled email wasn't delivered: it bounced.");
  });
  it("keeps the invite card's apart from the status messages'", () => {
    const all = [u("invite", "sms", "x"), u("status", "email", "y", "cancelled"), u("reminder", "email", "z")];
    expect(inviteUndelivered(all).map((x) => x.kind)).toEqual(["invite", "reminder"]);
    expect(statusUndelivered(all).map((x) => x.failure)).toEqual(["y"]);
  });
});
