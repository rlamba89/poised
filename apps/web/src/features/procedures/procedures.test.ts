import { describe, expect, it } from "vitest";
import { changedFields, hqForProcedure, initialHQ, procedureChoices, procedureLabel, type OfferedProcedure, type PublishedHQ } from "./procedures";

const hq = (versionId: string, isDefault = false): PublishedHQ => ({ versionId, versionNo: 1, name: versionId, questionnaireId: `q-${versionId}`, isDefault });
const proc = (id: string, defaultVersionId: string | null, code = ""): OfferedProcedure => ({
  id, name: `Procedure ${id}`, code, specialty: "", hospitalId: null, defaultVersionId,
});

describe("procedureLabel", () => {
  it("shows the code when there is one", () => {
    expect(procedureLabel({ name: "Hip", code: "HIP-01" })).toBe("Hip (HIP-01)");
    expect(procedureLabel({ name: "Hip", code: "" })).toBe("Hip");
  });
});

// S19 answers 6: the procedure's default HQ first, then the hospital's (Go resolves both into
// defaultVersionId), and the clinician can always change it.
describe("the HQ chosen on the New episode page", () => {
  const hqs = [hq("v1"), hq("v2", true)];
  it("starts with the hospital's default HQ, or none", () => {
    expect(initialHQ(hqs)).toBe("v2");
    expect(initialHQ([hq("v1")])).toBe("");
  });
  it("picking a procedure chooses its default HQ", () => {
    expect(hqForProcedure("v2", proc("p", "v1"), hqs)).toBe("v1");
  });
  it("keeps the HQ chosen when the procedure has no default, its default isn't offered, or it's cleared", () => {
    expect(hqForProcedure("v1", proc("p", null), hqs)).toBe("v1");
    expect(hqForProcedure("v1", proc("p", "gone"), hqs)).toBe("v1");
    expect(hqForProcedure("v1", undefined, hqs)).toBe("v1");
  });
});

describe("procedureChoices", () => {
  it("is what's offered, with the episode's own procedure kept first when it no longer is", () => {
    const offered = [proc("a", null, "A-1"), proc("b", null)];
    expect(procedureChoices(offered, { id: "a", name: "Procedure a" })).toEqual([
      { value: "a", label: "Procedure a (A-1)" }, { value: "b", label: "Procedure b" },
    ]);
    expect(procedureChoices(offered, { id: "old", name: "Appendix" })[0]).toEqual({ value: "old", label: "Appendix (no longer offered)" });
    expect(procedureChoices(offered, { id: null, name: "" })).toHaveLength(2);
  });
});

describe("changedFields", () => {
  it("keeps only what changed, null included", () => {
    expect(changedFields({ name: "Hip", hospitalId: "a1", code: "" }, { name: "Hip", hospitalId: null, code: "H" })).toEqual({ hospitalId: null, code: "H" });
    expect(changedFields({ name: "Hip" }, { name: "Hip" })).toEqual({});
  });
});
