import { describe, expect, it } from "vitest";
import { undeliveredText, type Undelivered } from "./undelivered";

// An undelivered text or email (S17, OPS-05), as the episode page's invite card says it.
describe("undeliveredText", () => {
  const u = (kind: Undelivered["kind"], channel: Undelivered["channel"], failure: string): Undelivered => ({
    kind, channel, failure, sentAt: "2026-10-14T09:02:00Z", failedAt: "2026-10-14T09:03:00Z",
  });
  it("names the message and says why", () => {
    expect(undeliveredText(u("invite", "sms", "the mobile number isn't valid"))).toBe(
      "The invite's text message wasn't delivered: the mobile number isn't valid.");
    expect(undeliveredText(u("reminder", "email", "it bounced"))).toBe("A reminder's email wasn't delivered: it bounced.");
  });
});
