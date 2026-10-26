// What the Rules page and the whole-HQ test page need (S40): the questionnaire with its ASA rules,
// every Question Set's JSON, and the trust's procedures here (a rule may name them). Saving the
// rules sends the whole set with its revision; Go checks its shape and refuses a stale save (409).
import { useCallback, useEffect, useRef, useState } from "react";
import { ruleNames, ruleSetOf, type ChapterJson, type HqSet, type RuleNames, type RuleSet } from "@poised/clinical";
import { api } from "@/lib/api";
import { canChange, useHospital } from "@/lib/auth";
import type { Chapter } from "@/features/chapters/types";
import type { OfferedProcedure } from "@/features/procedures/procedures";
import { oneAtATime } from "./oneAtATime";

export type HqDetail = {
  questionnaire: {
    id: string; hospitalId: string | null; name: string; versionId: string; versionNo: number; status: string;
    rules: unknown; rulesRevision: number;
  };
  chapters: Chapter[];
};

export type Hq = {
  detail: HqDetail;
  /** Every Question Set in order, with its JSON. */
  sets: HqSet[];
  /** The version's rules; null for a version published before rules existed. */
  rules: RuleSet | null;
  revision: number;
  procedures: OfferedProcedure[];
  names: RuleNames;
  /** Why the rules can only be read, or null when they can be changed. */
  readOnly: string | null;
};

export function useHq(base: string, questionnaireId: string) {
  const here = useHospital();
  const [hq, setHq] = useState<Hq | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const detail = await api<HqDetail>(`${base}/questionnaires/${questionnaireId}`);
      const [contents, procedures] = await Promise.all([
        Promise.all(detail.chapters.map((c) => api<{ content: ChapterJson }>(`${base}/chapters/${c.id}`))),
        api<OfferedProcedure[]>(`/o/${here.trust.id}/procedures?hospital=${here.hospital.id}`).catch(() => [] as OfferedProcedure[]),
      ]);
      const sets = detail.chapters.map((c, i) => ({ id: c.id, name: c.name, content: contents[i].content, audience: c.audience }));
      const q = detail.questionnaire;
      const readOnly = !canChange(here, q.hospitalId, "super_clinician")
        ? "Read only: only super clinicians and admins of the whole trust can change the trust's rules."
        : q.status !== "draft"
          ? "Read only: this version is published. To change its rules, create a new version."
          : null;
      setHq({
        detail, sets, rules: ruleSetOf(q.rules), revision: q.rulesRevision, procedures, readOnly,
        names: ruleNames(sets.map((s) => s.content), procedures),
      });
      setError("");
    } catch (e) {
      setError((e as Error).message);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [base, questionnaireId]);

  useEffect(() => {
    load();
  }, [load]);

  // The latest state for a save in flight, and one save at a time (test report 3, F-1): each is sent
  // with the revision the last one gave back, never two with the same one.
  const latest = useRef<Hq | null>(null);
  latest.current = hq;
  const [saving, setSaving] = useState(false);
  const saver = useRef(
    oneAtATime(async (next: RuleSet): Promise<string> => {
      const now = latest.current;
      if (!now) return "";
      setSaving(true);
      try {
        const res = await api<{ revision: number; rules: RuleSet }>(`${base}/questionnaires/${questionnaireId}/rules`, {
          method: "PUT",
          // The version loaded: Go refuses a save once another version is the latest (verification 1, F1).
          body: JSON.stringify({ rules: next, revision: now.revision, versionId: now.detail.questionnaire.versionId }),
        });
        // Go sends back what it stored (names and titles trimmed).
        const saved = { ...now, rules: ruleSetOf(res.rules), revision: res.revision };
        latest.current = saved;
        setHq(saved);
        return "";
      } catch (e) {
        return (e as Error).message;
      } finally {
        setSaving(false);
      }
    }),
  );

  /**
   * Saves the whole set: on success the page shows what Go stored; on a refusal it keeps what was
   * there and returns Go's message. While another save is in flight it sends nothing and says so.
   */
  const save = (next: RuleSet): Promise<string> => saver.current.run(next);

  return { hq, error, save, saving, reload: load };
}
