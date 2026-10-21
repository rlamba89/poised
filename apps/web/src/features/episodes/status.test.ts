import { describe, expect, it } from "vitest";
import { historyText, historyWho, pickOptions, statusRequest, type StatusEvent, type StatusOption } from "./status";
import { STATUS_LABELS } from "./types";

const onHold: StatusOption = { to: "on_hold", label: "On hold", move: "set", needsReason: false, needsDate: false };
const admit: StatusOption = { to: "admitted", label: "Admitted", move: "admit", needsReason: false, needsDate: true };
const cancel: StatusOption = { to: "cancelled", label: "Cancelled", move: "cancel", needsReason: true, needsDate: false };
const form = { reason: "", details: "", date: "" };

describe("pickOptions", () => {
  it("lists the picks, Admitted and Discharged, not cancel or re-open (they have buttons)", () => {
    const reopen: StatusOption = { to: "ready_for_review", label: "Ready for review", move: "reopen", needsReason: true, needsDate: false };
    expect(pickOptions([reopen, onHold, admit, cancel]).map((o) => o.to)).toEqual(["on_hold", "admitted"]);
  });
});

describe("statusRequest", () => {
  it("a pick, with an optional reason trimmed, and the revision", () => {
    expect(statusRequest({ ...form, kind: "pick", to: onHold }, 4)).toEqual({ path: "/status", body: { to: "on_hold", reason: "", revision: 4 } });
    expect(statusRequest({ ...form, kind: "pick", to: onHold, reason: " Bloods " }, 4)).toMatchObject({ body: { reason: "Bloods" } });
  });
  it("Admitted needs its date", () => {
    expect(statusRequest({ ...form, kind: "pick", to: admit }, 1)).toEqual({ error: "Enter the admission date as a date." });
    expect(statusRequest({ ...form, kind: "pick", to: admit, date: "2026-10-21" }, 1)).toEqual({
      path: "/status", body: { to: "admitted", reason: "", date: "2026-10-21", revision: 1 },
    });
  });
  it("cancel needs a reason from the list, and Other a sentence", () => {
    expect(statusRequest({ ...form, kind: "cancel" }, 2)).toEqual({ error: "Choose why the episode is cancelled." });
    expect(statusRequest({ ...form, kind: "cancel", reason: "other", details: "  " }, 2)).toEqual({ error: "Say why the episode is cancelled." });
    expect(statusRequest({ ...form, kind: "cancel", reason: "other", details: " Emigrated " }, 2)).toEqual({
      path: "/cancel", body: { reason: "other", details: "Emigrated", revision: 2 },
    });
    expect(statusRequest({ ...form, kind: "cancel", reason: "patient_withdrew" }, 2)).toMatchObject({ path: "/cancel" });
  });
  it("re-open and undo need a reason", () => {
    expect(statusRequest({ ...form, kind: "reopen", reason: " " }, 3)).toEqual({ error: "Say why the HQ review is being re-opened." });
    expect(statusRequest({ ...form, kind: "undo" }, 3)).toEqual({ error: "Say why the status change is being undone." });
    expect(statusRequest({ ...form, kind: "undo", reason: "Mistake" }, 3)).toEqual({ path: "/undo", body: { reason: "Mistake", revision: 3 } });
  });
});

describe("the history", () => {
  const ev = (e: Partial<StatusEvent>): StatusEvent => ({
    id: "1", move: "set", fromStatus: "ready_for_poa", toStatus: "on_hold", reason: "", rule: null, undoneAt: null,
    createdAt: "2026-10-21T10:00:00Z", userName: "Cara Clinician", ...e,
  });
  it.each([
    [ev({}), "Ready for POA → On hold"],
    [ev({ move: "create", fromStatus: null, toStatus: "hq_not_complete" }), "Created: HQ not complete"],
    [ev({ move: "submit", fromStatus: "hq_not_complete", toStatus: "ready_for_review" }), "HQ sent: HQ not complete → Ready for review"],
    [ev({ move: "cancel", toStatus: "cancelled" }), "Cancelled: Ready for POA → Cancelled"],
    [ev({ move: "undo", fromStatus: "admitted", toStatus: "ready_for_admission" }), "Undone: Admitted → Ready for admission"],
  ])("%#: %s", (e, want) => {
    expect(historyText(e, STATUS_LABELS)).toBe(want);
  });
  it("says who: the clinician, the patient, or the rule", () => {
    expect(historyWho(ev({}))).toBe("Cara Clinician");
    expect(historyWho(ev({ userName: null }))).toBe("Patient");
    expect(historyWho(ev({ userName: null, rule: "auto-triage" }))).toBe("Automatic (auto-triage)");
  });
});
