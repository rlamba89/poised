import { describe, expect, it } from "vitest";
import { stateOf, unsentWarning } from "./home";

describe("stateOf", () => {
  it.each([
    ["to_do", "To do", "Start"],
    ["in_progress", "In progress", "Continue"],
    ["sent", "Sent", "View"],
    ["cancelled", "Cancelled by the hospital", ""], // S20: no form, so no button
    ["closed", "Closed by the hospital", ""], // S22: archived, so no form either
  ] as const)("%s → %s, %s", (state, label, action) => {
    expect(stateOf(state)).toMatchObject({ label, action });
  });
});

// Closing the account (S22, EPI-09): the confirmation warns about each HQ not yet sent.
describe("unsentWarning", () => {
  const ep = (hospital: string, state: "to_do" | "in_progress" | "sent" | "cancelled" | "closed") => ({ hospital, questionnaireName: "Pre-op HQ", state });
  it("names nothing when every questionnaire is sent, cancelled or closed", () => {
    expect(unsentWarning([ep("A", "sent"), ep("B", "cancelled"), ep("C", "closed")])).toBe("");
  });
  it("names one not yet sent", () => {
    expect(unsentWarning([ep("Hospital A1", "in_progress"), ep("B", "sent")])).toBe(
      "Your Pre-op HQ for Hospital A1 hasn't been sent. If you close your account, it won't be sent.",
    );
  });
  it("names several", () => {
    expect(unsentWarning([ep("Hospital A1", "to_do"), ep("Hospital B1", "in_progress")])).toBe(
      "2 health questionnaires haven't been sent: Pre-op HQ for Hospital A1, and Pre-op HQ for Hospital B1. If you close your account, they won't be sent.",
    );
  });
});
