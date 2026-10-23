// The episode's Question Sets with both actors' answers (GET …/episodes/{eid}/hq).
import { useCallback, useEffect, useState } from "react";
import type { Answers, ChapterJson, PrefillChoice, PrefillEntry, SetPrefill } from "@poised/clinical";
import type { Audience } from "@/features/chapters/types";
import { api } from "@/lib/api";

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

/** The answers that count for a Question Set: the clinician's if they've started, else the patient's. */
export const currentAnswers = (hq: EpisodeHQ, id: string): Answers => hq.clinician[id]?.data ?? hq.patient[id] ?? {};

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
