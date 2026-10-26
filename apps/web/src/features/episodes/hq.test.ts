import { describe, expect, it } from "vitest";
import { hqPatient } from "./hq";

describe("hqPatient (S38)", () => {
  const sam = { firstName: "Sam", lastName: "Sample", dateOfBirth: "1975-11-01", sex: "male" };
  it("is the patient's age on the day they sent their answers, so a score never changes later", () => {
    // Sent two days before their 50th birthday: 49, however late a clinician looks.
    expect(hqPatient({ ...sam, patientSubmittedAt: "2025-10-30T10:00:00Z" })).toEqual({ name: "Sam Sample", age: 49, sex: "male" });
    expect(hqPatient({ ...sam, patientSubmittedAt: "2026-10-30T10:00:00Z" }).age).toBe(50);
  });
  it("is today's age while they haven't sent them", () => {
    const today = new Date();
    const expected = today.getFullYear() - 1975 - (today.getMonth() < 10 || (today.getMonth() === 10 && today.getDate() < 1) ? 1 : 0);
    expect(hqPatient({ ...sam, patientSubmittedAt: null }).age).toBe(expected);
  });
});
