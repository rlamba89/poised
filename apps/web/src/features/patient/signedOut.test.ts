import { describe, expect, it } from "vitest";
import { lostWork } from "./signedOut";

// When the patient's session has ended (30 minutes idle, a resend or a lock), a save or Send
// that was refused must say so: nothing may be lost without a word (S14 verification 1).
describe("lostWork", () => {
  it("says nothing more when only loading was refused", () => {
    expect(lostWork({})).toEqual([]);
  });
  it("names the Question Set whose change wasn't saved, and what to do", () => {
    expect(lostWork({ unsavedSet: "About you" })).toEqual([
      "Your latest change in About you wasn't saved. When you open the link again, please check your answers in that section.",
    ]);
  });
  it("says the answers weren't sent", () => {
    expect(lostWork({ unsent: true })).toEqual([
      "Your answers haven't been sent yet. When you open the link again, check them and tap Send my answers.",
    ]);
  });
  it("says both when both were lost", () => {
    expect(lostWork({ unsavedSet: "Your health", unsent: true })).toHaveLength(2);
  });
});
