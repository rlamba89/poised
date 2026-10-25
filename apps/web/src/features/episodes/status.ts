// The episode's status (S20). Which moves are allowed, and to whom, is Go's transition table
// (apps/api/internal/episode); the API sends the moves this user may make from here (`moves`),
// the undo if there is one (`undo`), and the history (`statusHistory`). This file only turns a
// filled-in form into the request to send, and words the history.
import type { EpisodeStatus } from "./types";

/** internal/episode's Move: the kind of a status change. */
export type StatusMove = "create" | "submit" | "complete_review" | "set" | "admit" | "discharge" | "cancel" | "reopen" | "undo"
  | "archive" | "unarchive"; // S22

/** A move the status control offers (episode.Option). */
export type StatusOption = { to: EpisodeStatus; label: string; move: StatusMove; needsReason: boolean; needsDate: boolean };

/** The undo, if there's one: back to `to` (S20 answers 2). Also the un-archive (S22): back to the status it was archived from. */
export type UndoOption = { to: EpisodeStatus; label: string } | null;

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
