// The episode's status (S20). Which moves are allowed, and to whom, is Go's transition table
// (apps/api/internal/episode); the API sends the moves this user may make from here (`moves`),
// the undo if there is one (`undo`), and the history (`statusHistory`). This file only turns a
// filled-in form into the request to send, and words the history.
import { STATUS_LABELS, type EpisodeStatus, type RuleSuggestion } from "./types";

/** internal/episode's Move: the kind of a status change. */
export type StatusMove = "create" | "submit" | "complete_review" | "set" | "admit" | "discharge" | "cancel" | "reopen" | "undo"
  | "archive" | "unarchive" // S22
  | "auto_triage"; // S42: the HQ's rules, never a person

/**
 * A move the status control offers (episode.Option). `blocked` (S43): why it's refused for now,
 * though the table allows it (a rule requires the anaesthetist's ASA grade before Ready for admission).
 */
export type StatusOption = { to: EpisodeStatus; label: string; move: StatusMove; needsReason: boolean; needsDate: boolean; blocked?: string };

/**
 * The undo, if there's one: back to `to` (S20 answers 2). Also the un-archive (S22): back to the status it was archived from.
 * `blocked` (S43): why the undo is refused for now (back to Ready for admission while the anaesthetist's ASA is required).
 */
export type UndoOption = { to: EpisodeStatus; label: string; blocked?: string } | null;

/** One change in the status history (EPI-03). userName is null for the patient's submit and automatic changes. */
export type StatusEvent = {
  id: string;
  move: StatusMove;
  fromStatus: EpisodeStatus | null;
  toStatus: EpisodeStatus;
  reason: string;
  rule: string | null;
  undoneAt: string | null;
  createdAt: string;
  userName: string | null;
};

/**
 * Whether the episode shows the red Not responding badge (EPI-02, S16): flagged on the hospital's
 * "not responding" day and the HQ still not sent. Not once archived (S27) or cancelled (S28): the
 * flag is history then.
 */
export function notResponding(e: { status: EpisodeStatus; notRespondingAt: string | null; patientSubmittedAt: string | null }): boolean {
  return !!e.notRespondingAt && !e.patientSubmittedAt && e.status !== "archived" && e.status !== "cancelled";
}

/** The fixed list of cancel reasons (S20 answers 3); "Other" needs a sentence. */
export const CANCEL_REASONS = [
  { value: "procedure_cancelled", label: "Procedure cancelled" },
  { value: "patient_withdrew", label: "Patient withdrew" },
  { value: "moved_hospital", label: "Moved to another hospital" },
  { value: "other", label: "Other" },
];

/** The moves the "Episode status" dropdown lists: the picks, Admitted and Discharged. Cancel, re-open and archive have buttons. */
export const pickOptions = (moves: StatusOption[]) => moves.filter((m) => m.move === "set" || m.move === "admit" || m.move === "discharge");

/** What the person filled in: a pick (with its target), cancel, re-open, undo, archive or un-archive (S22). */
export type StatusForm = {
  kind: "pick" | "cancel" | "reopen" | "undo" | "archive" | "unarchive";
  to?: StatusOption;
  reason: string; // cancel: the list's value
  details: string; // cancel only
  date: string; // YYYY-MM-DD, for Admitted and Discharged
};

/**
 * The request for a filled-in form, sent with the revision the page loaded; or the problem to
 * show first, worded as the API words it. The API checks everything again.
 */
export function statusRequest(f: StatusForm, revision: number): { path: string; body: Record<string, unknown> } | { error: string } {
  const reason = f.reason.trim();
  switch (f.kind) {
    case "cancel":
      if (!f.reason) return { error: "Choose why the episode is cancelled." };
      if (f.reason === "other" && !f.details.trim()) return { error: "Say why the episode is cancelled." };
      return { path: "/cancel", body: { reason: f.reason, details: f.details.trim(), revision } };
    case "reopen":
      if (!reason) return { error: "Say why the HQ review is being re-opened." };
      return { path: "/reopen", body: { reason, revision } };
    case "undo":
      if (!reason) return { error: "Say why the status change is being undone." };
      return { path: "/undo", body: { reason, revision } };
    case "archive": // S22: the reason is optional
      return { path: "/archive", body: { reason, revision } };
    case "unarchive":
      if (!reason) return { error: "Say why the episode is being un-archived." };
      return { path: "/unarchive", body: { reason, revision } };
  }
  if (!f.to) return { error: "Choose a status." };
  if (f.to.needsDate && !f.date) return { error: `Enter the ${f.to.move === "discharge" ? "discharge" : "admission"} date as a date.` };
  return { path: "/status", body: { to: f.to.to, reason, ...(f.to.needsDate ? { date: f.date } : {}), revision } };
}

/** What a history row says happened, e.g. "Ready for POA → On hold", using the clinicians' labels. */
export function historyText(ev: StatusEvent, labels: Record<EpisodeStatus, string>): string {
  const to = labels[ev.toStatus] ?? ev.toStatus;
  const from = ev.fromStatus ? (labels[ev.fromStatus] ?? ev.fromStatus) : "";
  const kind: Partial<Record<StatusMove, string>> = {
    create: "Created",
    submit: "HQ sent",
    complete_review: "HQ review completed",
    cancel: "Cancelled",
    reopen: "HQ review re-opened",
    undo: "Undone",
    archive: "Archived",
    unarchive: "Un-archived",
    auto_triage: "Auto-triaged",
  };
  if (ev.move === "create") return `Created: ${to}`;
  const what = kind[ev.move];
  return `${what ? `${what}: ` : ""}${from} → ${to}`;
}

/** Who made a change: the clinician, the patient, or an automatic rule (C6). */
export function historyWho(ev: StatusEvent): string {
  if (ev.userName) return ev.userName;
  if (ev.rule) return `Automatic (${ev.rule})`;
  return "Patient";
}

/** An episode's auto-triage as GET …/episodes/:id gives it (S42); all null when it wasn't, or was re-opened since. */
type Triage = { status: EpisodeStatus; triageRule: string | null; triageSummary: string | null; triagedAt: string | null; triageCheckedAt: string | null };

/** The stamp on each Question Set of an auto-triaged episode: "Auto-triaged (ASA II, no red or amber flags)". */
export const triageStamp = (e: Triage): string | null => (e.triagedAt && e.triageSummary ? `Auto-triaged (${e.triageSummary})` : null);

/** Whether the episode waits for a clinician's Checked (S42 answers 1): on the Auto-triaged worklist, as Go's in_worklist. */
export const awaitingCheck = (e: Triage): boolean => !!e.triagedAt && !e.triageCheckedAt && e.status !== "cancelled" && e.status !== "archived";

/** The status dropdown's label for a move: a blocked one says why, e.g. "Ready for admission (add the anaesthetist's ASA grade first)". */
export const pickLabel = (m: { label: string; blocked?: string }): string =>
  m.blocked ? `${m.label} (${m.blocked.charAt(0).toLowerCase()}${m.blocked.slice(1).replace(/\.$/, "")})` : m.label;

/**
 * A rule's requirement of the anaesthetist's ASA grade before Ready for admission (S43), in words,
 * or null when no rule asked for it. It's met once an anaesthetist's grade is saved.
 */
export function asaRequirement(e: { anaesthetistAsaRequiredBy: string | null; anaesthetistAsa: number | null }): { met: boolean; text: string } | null {
  const rule = e.anaesthetistAsaRequiredBy;
  if (!rule) return null;
  return e.anaesthetistAsa
    ? { met: true, text: `The anaesthetist's ASA grade is added, as the rule “${rule}” asks.` }
    : { met: false, text: `Ready for admission needs the anaesthetist's ASA grade first (the rule “${rule}”).` };
}

/** The main path an episode follows (S44a): a status later on it has passed the ones before. */
const MAIN_PATH: EpisodeStatus[] = ["hq_not_complete", "ready_for_review", "ready_for_poa", "poa_complete", "ready_for_admission", "admitted", "discharged"];
/** Where the rules' suggestions no longer matter (S44a): over, or out of the clinic's hands. */
const SUGGESTIONS_DONE: EpisodeStatus[] = ["cancelled", "archived", "admitted", "discharged"];

/**
 * Whether an episode has reached or passed a status (S44a, Rahul 27 Oct). On the main path, by its
 * place there; for On hold and Not ready (off it), by the history: has it ever been at the status
 * (a change undone doesn't count), or on the main path past it?
 */
export function reachedOrPassed(status: EpisodeStatus, history: Pick<StatusEvent, "toStatus" | "undoneAt">[], target: EpisodeStatus): boolean {
  if (status === target) return true;
  const rank = (x: EpisodeStatus) => MAIN_PATH.indexOf(x);
  if (rank(status) >= 0 && rank(target) >= 0) return rank(status) > rank(target);
  return history.some((h) => !h.undoneAt && (h.toStatus === target || (rank(target) >= 0 && rank(h.toStatus) > rank(target))));
}

/**
 * The statuses the HQ's rules suggest (S43), one line each: they're only shown; nothing moves by
 * itself. A suggestion shows until the episode reaches or passes it, and none once it's cancelled,
 * archived or admitted (S44a, Rahul 27 Oct); the history keeps what the rules suggested.
 */
export const suggestionLines = (
  suggestions: RuleSuggestion[], episode?: { status: EpisodeStatus }, history: Pick<StatusEvent, "toStatus" | "undoneAt">[] = [],
): { text: string; rule: string }[] =>
  suggestions
    .filter((s) => !episode || (!SUGGESTIONS_DONE.includes(episode.status) && !reachedOrPassed(episode.status, history, s.status)))
    .map((s) => ({ text: `The HQ's rules suggest: ${STATUS_LABELS[s.status] ?? s.status}`, rule: s.rule }));

/** The standing flags' names (S43), each once in the order added: two rules may add the same flag (verification 1, finding 4). */
export const standingFlagNames = (flags: { name: string; clearedAt: string | null }[] | undefined): string[] =>
  [...new Set((flags ?? []).filter((f) => !f.clearedAt).map((f) => f.name))];
