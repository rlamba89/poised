// An episode's tasks (S21, TSK-01), as GET …/episodes/{eid}/tasks returns them (Go: ListTasks in
// apps/api/db/queries/tasks.sql). Each is for a team; anyone at the hospital may complete any
// task, or re-open it. Which teams there are is Go's too (internal/episode, Teams); the labels
// here only name them.
import { formatDateTime } from "@/lib/format";

export type Team = "nurse" | "anaesthetist" | "admin";

export type Task = {
  id: string;
  title: string;
  team: Team;
  createdAt: string;
  /** Who added it, or null for a task a rule made (C6), which names its rule instead. */
  createdByName: string | null;
  createdByRule: string | null;
  /** Set while it's done. */
  doneAt: string | null;
  doneByName: string | null;
  /** The latest re-open, if it was ever re-opened. */
  reopenedAt: string | null;
  reopenedByName: string | null;
};

/** The teams, in the order the Add task form lists them (S21 answers 1). */
export const TEAMS: { value: Team; label: string }[] = [
  { value: "nurse", label: "Nurse" },
  { value: "anaesthetist", label: "Anaesthetist" },
  { value: "admin", label: "Admin" },
];

export const teamLabel = (t: Team) => TEAMS.find((x) => x.value === t)?.label ?? t;

/** The longest title the API takes (episode.MaxTaskTitle). */
export const MAX_TASK_TITLE = 200;

/**
 * Who did what, for one task: "Added by Cara Clinician on 21 Oct 2026, 09:00", then either
 * "Completed by …" or, for one that was re-opened, "Re-opened by …".
 */
export function taskHistory(t: Task): string[] {
  const out = [`Added by ${t.createdByName ?? `rule ${t.createdByRule ?? ""}`.trim()} on ${formatDateTime(t.createdAt)}`];
  if (t.doneAt) out.push(`Completed by ${t.doneByName ?? "someone"} on ${formatDateTime(t.doneAt)}`);
  else if (t.reopenedAt) out.push(`Re-opened by ${t.reopenedByName ?? "someone"} on ${formatDateTime(t.reopenedAt)}`);
  return out;
}

/** Open tasks first, then done ones; each group oldest first, as the API lists them. */
export function openFirst(tasks: Task[]): Task[] {
  return [...tasks.filter((t) => !t.doneAt), ...tasks.filter((t) => t.doneAt)];
}

/** How many open tasks to show (S21), or null for none: an archived episode shows no count (S29). */
export function openTaskCount(status: string, open: number): number | null {
  return status === "archived" ? null : open;
}
