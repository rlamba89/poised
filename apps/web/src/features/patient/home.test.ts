import { describe, expect, it } from "vitest";
import { stateOf } from "./home";

describe("stateOf", () => {
  it.each([
    ["to_do", "To do", "Start"],
    ["in_progress", "In progress", "Continue"],
    ["sent", "Sent", "View"],
    ["cancelled", "Cancelled by the hospital", ""], // S20: no form, so no button
  ] as const)("%s → %s, %s", (state, label, action) => {
    expect(stateOf(state)).toMatchObject({ label, action });
  });
});
