import { describe, expect, it } from "vitest";
import { allHospitals, atLeast, canChange, canManageStaff, findHospital, homePath, startPath, type Here, type Hospital, type Me, type Trust } from "./auth";

const hospital = (id: string, role: Hospital["role"]): Hospital => ({ id, name: `Hospital ${id}`, timeZone: "UTC", role });
const trustA: Trust = { id: "ta", name: "Trust A", code: "TA", role: "super_clinician", hospitals: [hospital("a1", "super_clinician"), hospital("a2", "admin")] };
const trustB: Trust = { id: "tb", name: "Trust B", code: "TB", role: null, hospitals: [hospital("b1", "clinician")] };
const me = (...trusts: Trust[]): Me => ({ user: { id: "u", name: "U", email: "u@x" }, platformAdmin: false, trusts });

describe("the role ladder (A-5)", () => {
  it("lets each role do what the roles below it can", () => {
    expect(atLeast("clinician", "clinician")).toBe(true);
    expect(atLeast("clinician", "super_clinician")).toBe(false); // a clinician can't publish
    expect(atLeast("super_clinician", "super_clinician")).toBe(true);
    expect(atLeast("super_clinician", "admin")).toBe(false);
    expect(atLeast("admin", "super_clinician")).toBe(true);
    expect(atLeast(null, "clinician")).toBe(false);
    expect(atLeast(undefined, "clinician")).toBe(false);
  });
});

describe("where a user goes", () => {
  it("starts clinicians on Episodes and the authoring roles on Questionnaires", () => {
    expect(homePath(trustB, trustB.hospitals[0])).toBe("/o/tb/h/b1/episodes");
    expect(homePath(trustA, trustA.hospitals[0])).toBe("/o/ta/h/a1/questionnaires");
  });
  it("goes straight in with one hospital, else to the picker", () => {
    expect(startPath(me(trustB))).toBe("/o/tb/h/b1/episodes");
    expect(startPath(me(trustA))).toBe("/");
    expect(startPath(me(trustA, trustB))).toBe("/");
    expect(startPath(me())).toBe("/");
  });
  it("lists every hospital with its trust", () => {
    expect(allHospitals(me(trustA, trustB)).map((x) => `${x.trust.id}/${x.hospital.id}`)).toEqual(["ta/a1", "ta/a2", "tb/b1"]);
  });
  it("finds a hospital only in its own trust", () => {
    expect(findHospital(me(trustA, trustB), "ta", "a2")?.hospital.role).toBe("admin");
    expect(findHospital(me(trustA, trustB), "tb", "a2")).toBeNull(); // A2 isn't Trust B's
    expect(findHospital(me(trustA), "tb", "b1")).toBeNull(); // not a member
  });
});

describe("changing content (S08 answers, 5)", () => {
  const at = (trust: Trust, h: Hospital): Here => ({ trust, hospital: h, role: h.role, base: "" });
  // Hospital-only super clinician and admin at H1, in a trust where they have no whole-trust role.
  const local: Trust = { id: "tc", name: "Trust C", code: "TC", role: null, hospitals: [hospital("c1", "super_clinician"), hospital("c2", "admin")] };
  it("needs a whole-trust role for the trust's shared content", () => {
    expect(canChange(at(trustA, trustA.hospitals[0]), null, "super_clinician")).toBe(true);
    expect(canChange(at(local, local.hospitals[0]), null, "super_clinician")).toBe(false);
    expect(canChange(at(local, local.hospitals[1]), null, "admin")).toBe(false);
    expect(canChange(at(trustA, trustA.hospitals[0]), null, "admin")).toBe(false); // super clinician of the trust
  });
  it("needs only the role at this hospital for the hospital's own content", () => {
    expect(canChange(at(local, local.hospitals[0]), "c1", "super_clinician")).toBe(true);
    expect(canChange(at(trustB, trustB.hospitals[0]), "b1", "super_clinician")).toBe(false);
  });
});

describe("who manages staff (S09)", () => {
  it("is only the trust's whole-trust admins", () => {
    expect(canManageStaff({ ...trustA, role: "admin" })).toBe(true);
    expect(canManageStaff(trustA)).toBe(false); // a whole-trust super clinician
    // An admin of one hospital only (trust role null) can't, even with "admin" at that hospital.
    expect(canManageStaff({ ...trustB, role: null, hospitals: [hospital("b1", "admin")] })).toBe(false);
  });
});
