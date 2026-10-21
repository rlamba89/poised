import { describe, expect, it } from "vitest";
import { allHospitals, atLeast, canChange, canManageProcedures, canManageSettings, canManageStaff, canSeeAudit, findHospital, homePath, startPath, switchPath, type Here, type Hospital, type Me, type Trust } from "./auth";

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
  it("sends a platform admin with no hospital to the admin console (S10)", () => {
    expect(startPath({ ...me(), platformAdmin: true })).toBe("/admin");
    expect(startPath({ ...me(trustB), platformAdmin: true })).toBe("/o/tb/h/b1/episodes");
  });
  it("leaves a trust with no hospitals yet out of the hospitals (S10)", () => {
    const empty: Trust = { id: "te", name: "New Trust", code: "NT", role: "admin", hospitals: [] };
    expect(allHospitals(me(empty, trustB)).map((x) => x.hospital.id)).toEqual(["b1"]);
    expect(startPath(me(empty))).toBe("/");
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

describe("who changes the trust's settings (S10)", () => {
  it("is only the trust's whole-trust admins, as for staff", () => {
    expect(canManageSettings({ ...trustA, role: "admin" })).toBe(true);
    expect(canManageSettings(trustA)).toBe(false);
    expect(canManageSettings({ ...trustB, role: null, hospitals: [hospital("b1", "admin")] })).toBe(false);
  });
});

describe("switching hospital from the header", () => {
  const adminA: Trust = { ...trustA, role: "admin" };
  it("keeps the section: Episodes stays Episodes, the editor goes to Questionnaires", () => {
    expect(switchPath("/o/ta/h/a1/episodes/e1", trustB, trustB.hospitals[0])).toBe("/o/tb/h/b1/episodes");
    // A patient's page is under Episodes: another hospital gets its own list, never this patient (S13).
    expect(switchPath("/o/ta/h/a1/patients/p1", adminA, adminA.hospitals[1])).toBe("/o/ta/h/a2/episodes");
    expect(switchPath("/o/ta/h/a1/questionnaires/q1", adminA, adminA.hospitals[1])).toBe("/o/ta/h/a2/questionnaires");
  });
  it("keeps Staff and Settings where the user manages the hospital's trust", () => {
    expect(switchPath("/o/ta/h/a1/settings", adminA, adminA.hospitals[1])).toBe("/o/ta/h/a2/settings");
    expect(switchPath("/o/ta/h/a1/staff", adminA, adminA.hospitals[1])).toBe("/o/ta/h/a2/staff");
    expect(switchPath("/o/ta/h/a1/audit", adminA, adminA.hospitals[1])).toBe("/o/ta/h/a2/audit"); // S11
    // From a trust where they're refused to one they manage (S10 test report 1: Max landed on Questionnaires).
    expect(switchPath("/o/tb/h/b1/settings", adminA, adminA.hospitals[0])).toBe("/o/ta/h/a1/settings");
  });
  it("goes to the start page where they don't manage the trust", () => {
    expect(switchPath("/o/ta/h/a1/settings", trustB, trustB.hospitals[0])).toBe("/o/tb/h/b1/episodes");
    expect(switchPath("/o/ta/h/a1/staff", trustA, trustA.hospitals[1])).toBe("/o/ta/h/a2/questionnaires");
    expect(switchPath("/o/ta/h/a1/audit", trustB, trustB.hospitals[0])).toBe("/o/tb/h/b1/episodes");
  });
});

describe("who sees the audit log (S11)", () => {
  it("is only the trust's whole-trust admins, as for Staff and Settings", () => {
    expect(canSeeAudit({ ...trustA, role: "admin" })).toBe(true);
    expect(canSeeAudit(trustA)).toBe(false); // a whole-trust super clinician
    expect(canSeeAudit({ ...trustB, hospitals: [hospital("b1", "admin")] })).toBe(false); // an admin of one hospital
  });
});

describe("who manages the trust's procedures (S19)", () => {
  it("is the trust's whole-trust super clinicians and admins (answers 1)", () => {
    expect(canManageProcedures(trustA)).toBe(true); // a whole-trust super clinician
    expect(canManageProcedures({ ...trustA, role: "admin" })).toBe(true);
    expect(canManageProcedures({ ...trustA, role: "clinician" })).toBe(false);
    // A super clinician of one hospital only (trust role null), as Sue Super.
    expect(canManageProcedures({ ...trustB, role: null, hospitals: [hospital("b1", "super_clinician")] })).toBe(false);
  });
  it("switching hospital keeps Procedures where they manage the trust", () => {
    expect(switchPath("/o/ta/h/a1/procedures", trustA, trustA.hospitals[1])).toBe("/o/ta/h/a2/procedures");
    expect(switchPath("/o/ta/h/a1/procedures", { ...trustB, role: "clinician" }, trustB.hospitals[0])).toBe("/o/tb/h/b1/episodes");
  });
});
