// The trust's procedure list (plan C3, S19; PRC-01) as the Go API returns it
// (apps/api/internal/httpapi/procedures.go), and the rules that don't need React, unit-tested in
// procedures.test.ts. Go decides which procedures are offered where and which HQ each one picks.
import type { ProcedureInfoItem } from "@/features/information/info";

/** GET /api/o/{trust}/procedures: the list for managing it, retired ones included. */
export type Procedure = {
  id: string;
  name: string;
  code: string;
  specialty: string;
  hospitalId: string | null; // null: every hospital of the trust
  hospitalName: string | null;
  defaultQuestionnaireId: string | null;
  defaultQuestionnaireName: string | null;
  retired: boolean;
  /** The patient's Information (S36, INF-04): content library item ids, in order, and those items. */
  infoContent: string[];
  infoItems: ProcedureInfoItem[];
};

/**
 * GET /api/o/{trust}/procedures?hospital={id}: what a new episode at that hospital can be given.
 * defaultVersionId is the HQ chosen when it's picked: its own default HQ's published version,
 * else the hospital's default HQ (Settings), else null (the clinician's choice stays).
 */
export type OfferedProcedure = {
  id: string;
  name: string;
  code: string;
  specialty: string;
  hospitalId: string | null;
  defaultVersionId: string | null;
};

/** A published HQ the hospital can give (GET …/published-hqs); isDefault: the hospital's default HQ. */
export type PublishedHQ = { versionId: string; versionNo: number; name: string; questionnaireId: string; isDefault: boolean };

/** "Total hip replacement (HIP-01)", or the name alone without a code. */
export const procedureLabel = (p: { name: string; code: string }) => (p.code ? `${p.name} (${p.code})` : p.name);

/** The HQ chosen first when the New episode page opens: the hospital's default HQ, or none. */
export function initialHQ(hqs: PublishedHQ[]): string {
  return hqs.find((h) => h.isDefault)?.versionId ?? "";
}

/**
 * The HQ after a procedure is picked (S19 answers 6): the procedure's default here, if it's one of
 * the HQs offered, else the HQ already chosen. The clinician can change it afterwards.
 */
export function hqForProcedure(current: string, procedure: OfferedProcedure | undefined, hqs: PublishedHQ[]): string {
  const v = procedure?.defaultVersionId;
  return v && hqs.some((h) => h.versionId === v) ? v : current;
}

/**
 * The choices for an episode's procedure: those offered at its hospital, plus the one it has if
 * that's no longer offered (retired, or moved to another hospital), so it stays shown and kept.
 */
export function procedureChoices(offered: OfferedProcedure[], current: { id: string | null; name: string }): { value: string; label: string }[] {
  const choices = offered.map((p) => ({ value: p.id, label: procedureLabel(p) }));
  if (current.id && !offered.some((p) => p.id === current.id)) {
    choices.unshift({ value: current.id, label: `${current.name} (no longer offered)` });
  }
  return choices;
}

/** The fields of `next` that differ from `was`: an edit sends only those (PATCH keeps the rest). Lists compare by their items. */
export function changedFields<T extends Record<string, unknown>>(was: T, next: T): Partial<T> {
  const same = (a: unknown, b: unknown) => (Array.isArray(a) && Array.isArray(b) ? JSON.stringify(a) === JSON.stringify(b) : a === b);
  return Object.fromEntries(Object.entries(next).filter(([k, v]) => !same(was[k], v))) as Partial<T>;
}
