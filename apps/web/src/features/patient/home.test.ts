import { describe, expect, it } from "vitest";
import { closedText, stateOf, unsentWarning } from "./home";

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

// S28 (Rahul, 24 Oct): a cancelled questionnaire the patient had sent doesn't say "nothing to fill in".
describe("closedText", () => {
  it("a cancelled one the patient had sent asks them to contact the hospital", () => {
    expect(closedText("cancelled", true)).toBe("The hospital has cancelled this. Please contact them if you have questions.");
  });
  it("an unsent cancelled one, and a closed one, keep their wording", () => {
    expect(closedText("cancelled", false)).toBe("The hospital has cancelled this, so there's nothing to fill in. Please contact them if you have any questions.");
    expect(closedText("closed", false)).toBe("The hospital has closed this, so there's nothing to fill in. Please contact them if you have any questions.");
  });
  it("nothing for the others", () => {
    expect(closedText("sent", true)).toBe("");
    expect(closedText("to_do", false)).toBe("");
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
