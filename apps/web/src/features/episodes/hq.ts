"use client";
// The episode's Question Sets with both actors' answers (GET …/episodes/{eid}/hq).
import { useCallback, useEffect, useState } from "react";
import type { Answers, ChapterJson } from "@sj/clinical";
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
};
export type EpisodeHQ = {
  chapters: HQSet[];
  /** The patient's answers per Question Set, frozen once sent. */
  patient: Record<string, Answers>;
  /** The clinician's row per Question Set: final answers and the "Validated by" stamp. */
  clinician: Record<string, AnswerRow>;
};

/** The answers that count for a Question Set: the clinician's if they've started, else the patient's. */
export const currentAnswers = (hq: EpisodeHQ, id: string): Answers => hq.clinician[id]?.data ?? hq.patient[id] ?? {};

export function useEpisodeHQ(hospitalId: string, episodeId: string) {
  const [hq, setHq] = useState<EpisodeHQ | null>(null);
  const [error, setError] = useState("");
  const reload = useCallback(
    () =>
      api<{ chapters: HQSet[]; answers: AnswerRow[] }>(`/h/${hospitalId}/episodes/${episodeId}/hq`)
        .then(({ chapters, answers }) => {
          const patient: Record<string, Answers> = {};
          const clinician: Record<string, AnswerRow> = {};
          for (const a of answers) {
            if (a.actor === "patient") patient[a.chapterId] = a.data;
            else clinician[a.chapterId] = a;
          }
          setHq({ chapters, patient, clinician });
        })
        .catch((e: Error) => setError(e.message)),
    [hospitalId, episodeId],
  );
  useEffect(() => {
    reload();
  }, [reload]);
  return { hq, error, reload };
}
