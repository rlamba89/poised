// The episode's Question Sets with both actors' answers (GET …/episodes/{eid}/hq).
import { useCallback, useEffect, useState } from "react";
import { countingHqAnswers, patientVariables, type Answers, type ChapterJson, type PrefillChoice, type PrefillEntry, type SamplePatient, type SetPrefill } from "@poised/clinical";
import type { Audience } from "@/features/chapters/types";
import { api } from "@/lib/api";
import { deploymentTimeZone } from "@/lib/config";

export type HQSet = { id: string; name: string; description: string; icon: string; audience: Audience; content: ChapterJson };
export type AnswerRow = {
  chapterId: string;
  actor: "patient" | "clinician";
  data: Answers;
  updatedAt: string;
  validatedAt: string | null;
  updatedByName: string | null;
  /** Counts the row's saves; the next save sends it back (S02a). */
  revision: number;
  /** The patient's row, pre-filled (S25): where each answer came from, and the opening card's button (who pressed it, if staff did). */
  prefill?: Record<string, PrefillEntry> | null;
  prefillChoice?: PrefillChoice | null;
  prefillChoiceAt?: string | null;
  prefillChoiceByName?: string | null;
};
export type EpisodeHQ = {
  chapters: HQSet[];
  /** The patient's answers per Question Set, frozen once sent. */
  patient: Record<string, Answers>;
  /** Who entered the patient's answers for them, per Question Set (PAT-09); absent where the patient did. */
  enteredBy: Record<string, string>;
  /** The clinician's row per Question Set: final answers and the "Validated by" stamp. */
  clinician: Record<string, AnswerRow>;
  /** The patient's pre-filled Question Sets (S26, PRE-04): where each answer came from, and the button pressed. */
  prefill: Record<string, SetPrefill>;
};

/**
 * The answers that count for a Question Set: the clinician's if they've started, else the patient's.
 * One rule with the assessments' pre-fill (@poised/clinical countingHqAnswers), so they can't drift.
 */
export const currentAnswers = (hq: EpisodeHQ, id: string): Answers => countingHqAnswers([{ id }], hq.patient, hq.clinician)[id] ?? {};

/** Every Question Set's answers that count, by set. */
export const allCurrentAnswers = (hq: EpisodeHQ): Record<string, Answers> => Object.fromEntries(hq.chapters.map((c) => [c.id, currentAnswers(hq, c.id)]));

/**
 * The patient as the HQ's rules see them: their age on the day they sent their answers, so a score
 * never changes later (S38: "over 50" stays as it was), or today's while they haven't sent them.
 */
export const hqPatient = (e: {
  firstName: string; lastName: string; dateOfBirth: string; sex: string; patientSubmittedAt: string | null; hospitalTimeZone?: string | null;
}): SamplePatient =>
  // S44a: the day of sending at the hospital (its time zone, else the deployment's: A-20), as Go works it out.
  patientVariables(e, e.patientSubmittedAt ? new Date(e.patientSubmittedAt) : undefined, e.hospitalTimeZone ?? deploymentTimeZone());

export function useEpisodeHQ(base: string, episodeId: string) {
  const [hq, setHq] = useState<EpisodeHQ | null>(null);
  const [error, setError] = useState("");
  const reload = useCallback(
    () =>
      api<{ chapters: HQSet[]; answers: AnswerRow[] }>(`${base}/episodes/${episodeId}/hq`)
        .then(({ chapters, answers }) => {
          const patient: Record<string, Answers> = {};
          const enteredBy: Record<string, string> = {};
          const clinician: Record<string, AnswerRow> = {};
          const prefill: Record<string, SetPrefill> = {};
          for (const a of answers) {
            if (a.actor === "patient") {
              patient[a.chapterId] = a.data;
              if (a.updatedByName) enteredBy[a.chapterId] = a.updatedByName; // staff, on the patient's behalf
              if (a.prefill && Object.keys(a.prefill).length > 0) {
                prefill[a.chapterId] = {
                  answers: a.prefill, choice: a.prefillChoice ?? null, choiceAt: a.prefillChoiceAt ?? null, choiceByName: a.prefillChoiceByName ?? null,
                };
              }
            } else clinician[a.chapterId] = a;
          }
          setHq({ chapters, patient, enteredBy, clinician, prefill });
        })
        .catch((e: Error) => setError(e.message)),
    [base, episodeId],
  );
  useEffect(() => {
    reload();
  }, [reload]);
  return { hq, error, reload };
}
