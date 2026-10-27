import { describe, expect, it } from "vitest";
import {
  asaRequirement, awaitingCheck, historyText, pickLabel, standingFlagNames, suggestionLines, historyWho, notResponding, pickOptions, statusRequest, triageStamp, type StatusEvent, type StatusOption,
} from "./status";
import { STATUS_LABELS, type EpisodeStatus } from "./types";

const onHold: StatusOption = { to: "on_hold", label: "On hold", move: "set", needsReason: false, needsDate: false };
const admit: StatusOption = { to: "admitted", label: "Admitted", move: "admit", needsReason: false, needsDate: true };
const cancel: StatusOption = { to: "cancelled", label: "Cancelled", move: "cancel", needsReason: true, needsDate: false };
const form = { reason: "", details: "", date: "" };

describe("pickOptions", () => {
  it("lists the picks, Admitted and Discharged, not cancel, re-open or archive (they have buttons)", () => {
    const reopen: StatusOption = { to: "ready_for_review", label: "Ready for review", move: "reopen", needsReason: true, needsDate: false };
    const archive: StatusOption = { to: "archived", label: "Archived", move: "archive", needsReason: false, needsDate: false };
    expect(pickOptions([reopen, onHold, admit, cancel, archive]).map((o) => o.to)).toEqual(["on_hold", "admitted"]);
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
  it("archive takes an optional reason; un-archive needs one (S22)", () => {
    expect(statusRequest({ ...form, kind: "archive" }, 5)).toEqual({ path: "/archive", body: { reason: "", revision: 5 } });
    expect(statusRequest({ ...form, kind: "archive", reason: " Duplicate " }, 5)).toEqual({ path: "/archive", body: { reason: "Duplicate", revision: 5 } });
    expect(statusRequest({ ...form, kind: "unarchive", reason: " " }, 6)).toEqual({ error: "Say why the episode is being un-archived." });
    expect(statusRequest({ ...form, kind: "unarchive", reason: "Mistake" }, 6)).toEqual({ path: "/unarchive", body: { reason: "Mistake", revision: 6 } });
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
    [ev({ move: "archive", fromStatus: "discharged", toStatus: "archived" }), "Archived: Discharged → Archived"],
    [ev({ move: "unarchive", fromStatus: "archived", toStatus: "discharged" }), "Un-archived: Archived → Discharged"],
    [ev({ move: "auto_triage", fromStatus: "ready_for_review", toStatus: "ready_for_admission" }), "Auto-triaged: Ready for review → Ready for admission"],
  ])("%#: %s", (e, want) => {
    expect(historyText(e, STATUS_LABELS)).toBe(want);
  });
  it("says who: the clinician, the patient, or the rule", () => {
    expect(historyWho(ev({}))).toBe("Cara Clinician");
    expect(historyWho(ev({ userName: null }))).toBe("Patient");
    expect(historyWho(ev({ userName: null, rule: "auto-triage" }))).toBe("Automatic (auto-triage)");
    expect(historyWho(ev({ userName: null, rule: "Auto-archive: 30 days after discharge" }))).toBe("Automatic (Auto-archive: 30 days after discharge)");
  });
});

// S27 (pending.md, S22): the flag is history once the episode is archived; S28: or cancelled.
describe("notResponding (the red badge)", () => {
  const at = "2026-10-01T10:00:00Z";
  it("while flagged and the HQ isn't sent, unless the episode is archived or cancelled", () => {
    expect(notResponding({ status: "hq_not_complete", notRespondingAt: at, patientSubmittedAt: null })).toBe(true);
    expect(notResponding({ status: "hq_not_complete", notRespondingAt: null, patientSubmittedAt: null })).toBe(false);
    expect(notResponding({ status: "ready_for_review", notRespondingAt: at, patientSubmittedAt: at })).toBe(false);
    expect(notResponding({ status: "archived", notRespondingAt: at, patientSubmittedAt: null })).toBe(false);
    expect(notResponding({ status: "cancelled", notRespondingAt: at, patientSubmittedAt: null })).toBe(false);
  });
});

// S42: an auto-triaged episode's stamp, and whether it waits for a clinician's Checked.
describe("auto-triage", () => {
  const triaged = { status: "ready_for_admission" as const, triageRule: "Low risk", triageSummary: "ASA II, no red or amber flags",
    triagedAt: "2026-10-27T10:00:00Z", triageCheckedAt: null };
  it("stamps each set with the rule's words while the auto-triage stands", () => {
    expect(triageStamp(triaged)).toBe("Auto-triaged (ASA II, no red or amber flags)");
    expect(triageStamp({ ...triaged, triageCheckedAt: "2026-10-27T11:00:00Z" })).toBe("Auto-triaged (ASA II, no red or amber flags)");
    expect(triageStamp({ ...triaged, triageRule: null, triageSummary: null, triagedAt: null })).toBeNull(); // never, or re-opened
  });
  it("waits for Checked until a clinician clicks it; never once cancelled or archived", () => {
    expect(awaitingCheck(triaged)).toBe(true);
    expect(awaitingCheck({ ...triaged, triageCheckedAt: "2026-10-27T11:00:00Z" })).toBe(false);
    expect(awaitingCheck({ ...triaged, triagedAt: null, triageRule: null, triageSummary: null })).toBe(false);
    expect(awaitingCheck({ ...triaged, status: "cancelled" })).toBe(false);
    expect(awaitingCheck({ ...triaged, status: "archived" })).toBe(false);
  });
});

// S43: the rules' requirement of the anaesthetist's ASA grade, their suggestions, and the status control's blocked move.
describe("asaRequirement", () => {
  it("is null when no rule requires the anaesthetist's grade", () => {
    expect(asaRequirement({ anaesthetistAsaRequiredBy: null, anaesthetistAsa: null })).toBeNull();
  });
  it("says what's needed while it's unmet", () => {
    expect(asaRequirement({ anaesthetistAsaRequiredBy: "High risk", anaesthetistAsa: null })).toEqual({
      met: false,
      text: "Ready for admission needs the anaesthetist's ASA grade first (the rule “High risk”).",
    });
  });
  it("says it's met once the grade is saved", () => {
    expect(asaRequirement({ anaesthetistAsaRequiredBy: "High risk", anaesthetistAsa: 3 })).toEqual({
      met: true,
      text: "The anaesthetist's ASA grade is added, as the rule “High risk” asks.",
    });
  });
});

describe("suggestionLines", () => {
  it("words each status the rules suggest, with its rule", () => {
    expect(suggestionLines([{ rule: "Fit", status: "ready_for_poa" }, { rule: "Wait", status: "on_hold" }])).toEqual([
      { text: "The HQ's rules suggest: Ready for POA", rule: "Fit" },
      { text: "The HQ's rules suggest: On hold", rule: "Wait" },
    ]);
    expect(suggestionLines([])).toEqual([]);
  });

  // S44a (Rahul, 27 Oct): until the episode reaches or passes the status; never once it's over.
  const poa = [{ rule: "Fit", status: "ready_for_poa" as const }];
  const ev = (toStatus: EpisodeStatus, undoneAt: string | null = null) => ({ toStatus, undoneAt });
  it("shows a suggestion until the episode reaches or passes it on the main path", () => {
    expect(suggestionLines(poa, { status: "ready_for_review" })).toHaveLength(1);
    expect(suggestionLines(poa, { status: "ready_for_poa" })).toEqual([]);
    expect(suggestionLines(poa, { status: "poa_complete" })).toEqual([]);
    expect(suggestionLines(poa, { status: "ready_for_admission" })).toEqual([]);
  });
  it("shows none once cancelled, archived, admitted or discharged", () => {
    for (const status of ["cancelled", "archived", "admitted", "discharged"] as const) {
      expect(suggestionLines(poa, { status }, [ev("ready_for_review")])).toEqual([]);
    }
  });
  it("uses the history for On hold and Not ready", () => {
    // On hold straight after the review: it never reached Ready for POA, so the suggestion stands.
    expect(suggestionLines(poa, { status: "on_hold" }, [ev("ready_for_review"), ev("on_hold")])).toHaveLength(1);
    // It was at Ready for POA (or past it) before going on hold.
    expect(suggestionLines(poa, { status: "on_hold" }, [ev("ready_for_review"), ev("ready_for_poa"), ev("on_hold")])).toEqual([]);
    expect(suggestionLines(poa, { status: "not_ready" }, [ev("ready_for_review"), ev("ready_for_admission"), ev("not_ready")])).toEqual([]);
    // A change undone doesn't count.
    expect(suggestionLines(poa, { status: "on_hold" }, [ev("ready_for_review"), ev("ready_for_poa", "2026-10-27T10:00:00Z"), ev("on_hold")])).toHaveLength(1);
    // A suggested On hold: shown until the episode has been on hold.
    const hold = [{ rule: "Wait", status: "on_hold" as const }];
    expect(suggestionLines(hold, { status: "ready_for_poa" }, [ev("ready_for_review"), ev("ready_for_poa")])).toHaveLength(1);
    expect(suggestionLines(hold, { status: "on_hold" }, [ev("on_hold")])).toEqual([]);
    expect(suggestionLines(hold, { status: "ready_for_poa" }, [ev("on_hold"), ev("ready_for_poa")])).toEqual([]);
  });
});

describe("pickLabel", () => {
  it("adds why a blocked move is refused for now", () => {
    const rfa: StatusOption = { to: "ready_for_admission", label: "Ready for admission", move: "set", needsReason: false, needsDate: false };
    expect(pickLabel(rfa)).toBe("Ready for admission");
    expect(pickLabel({ ...rfa, blocked: "Add the anaesthetist's ASA grade first." })).toBe("Ready for admission (add the anaesthetist's ASA grade first)");
  });
});

// Verification 1, finding 4: two rules may add the same flag; the header shows each standing name once.
describe("standingFlagNames", () => {
  const flag = (name: string, clearedAt: string | null = null) => ({ name, clearedAt });
  it("lists the standing flags' names, each once, in order", () => {
    expect(standingFlagNames([flag("Anaesthetic review"), flag("Airway", "2026-10-27T09:00:00Z"), flag("Anaesthetic review"), flag("Diabetes")]))
      .toEqual(["Anaesthetic review", "Diabetes"]);
    expect(standingFlagNames(undefined)).toEqual([]);
  });
});
