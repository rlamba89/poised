import { describe, expect, it } from "vitest";
import { formatDateTime } from "@/lib/format";
import { openFirst, taskHistory, teamLabel, type Task } from "./tasks";

const task = (over: Partial<Task>): Task => ({
  id: "t1", title: "Book anaesthetic review", team: "anaesthetist", createdAt: "2026-10-21T09:00:00Z",
  createdByName: "Cara Clinician", createdByRule: null, doneAt: null, doneByName: null, reopenedAt: null, reopenedByName: null,
  ...over,
});

// The Tasks panel (S21): who did what to each task.
describe("taskHistory", () => {
  it("says who added an open task", () => {
    expect(taskHistory(task({}))).toEqual([`Added by Cara Clinician on ${formatDateTime("2026-10-21T09:00:00Z")}`]);
  });
  it("adds who completed a done one", () => {
    const t = task({ doneAt: "2026-10-21T10:00:00Z", doneByName: "Val Clinician" });
    expect(taskHistory(t)[1]).toBe(`Completed by Val Clinician on ${formatDateTime("2026-10-21T10:00:00Z")}`);
  });
  it("adds who re-opened one that's open again, but not once it's done again", () => {
    const reopened = task({ reopenedAt: "2026-10-21T11:00:00Z", reopenedByName: "Sue Super" });
    expect(taskHistory(reopened)[1]).toBe(`Re-opened by Sue Super on ${formatDateTime("2026-10-21T11:00:00Z")}`);
    const doneAgain = { ...reopened, doneAt: "2026-10-21T12:00:00Z", doneByName: "Cara Clinician" };
    expect(taskHistory(doneAgain)).toHaveLength(2);
    expect(taskHistory(doneAgain)[1]).toMatch(/^Completed by Cara Clinician/);
  });
  it("names a rule's task by its rule (C6)", () => {
    expect(taskHistory(task({ createdByName: null, createdByRule: "Anaesthetic review" }))[0]).toMatch(/^Added by rule Anaesthetic review on /);
  });
});

describe("openFirst and teamLabel", () => {
  it("puts open tasks before done ones, keeping each group's order", () => {
    const a = task({ id: "a", doneAt: "2026-10-21T10:00:00Z" }), b = task({ id: "b" }), c = task({ id: "c" });
    expect(openFirst([a, b, c]).map((t) => t.id)).toEqual(["b", "c", "a"]);
  });
  it("names the teams", () => {
    expect([teamLabel("nurse"), teamLabel("anaesthetist"), teamLabel("admin")]).toEqual(["Nurse", "Anaesthetist", "Admin"]);
  });
});
