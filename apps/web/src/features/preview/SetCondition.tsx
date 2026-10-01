"use client";
// The Question Set's own condition (LOG-02), in plain words above the preview. A one-set preview
// can't see earlier sets' answers, so it decides only conditions that test just the patient.
import { useEffect, useState } from "react";
import { Alert, Text } from "@mantine/core";
import { IconGitBranch } from "@tabler/icons-react";
import {
  chapterConditionOf, combineChapters, describeLogic, isChapterShown, testsOnlyThePatient, type ChapterJson, type Viewer,
} from "@sj/clinical";
import { api } from "@/lib/api";
import { DisplaysWhen } from "@/features/editor/ElementCard";

type Detail = { chapters: { id: string; name: string }[] };

export function SetCondition(props: {
  hospitalId: string;
  questionnaireId: string;
  chapterId: string;
  content: ChapterJson;
  variables: { patientAge?: number; patientSex?: string; viewer: Viewer };
}) {
  const { hospitalId, questionnaireId, chapterId, content, variables } = props;
  const expr = chapterConditionOf(content);
  const [earlier, setEarlier] = useState<ChapterJson | null>(null);
  useEffect(() => {
    if (!expr) return;
    let current = true;
    (async () => {
      const d = await api<Detail>(`/h/${hospitalId}/questionnaires/${questionnaireId}`);
      const before = d.chapters.slice(0, d.chapters.findIndex((c) => c.id === chapterId));
      const docs = await Promise.all(before.map((c) => api<{ content: ChapterJson }>(`/h/${hospitalId}/chapters/${c.id}`)));
      if (current) setEarlier(combineChapters(before.map((c, i) => ({ name: c.name, doc: docs[i].content ?? {} }))));
    })().catch(() => current && setEarlier({}));
    return () => {
      current = false;
    };
  }, [expr, hospitalId, questionnaireId, chapterId]);

  if (!expr) return null;
  const described = describeLogic(earlier ?? {}, expr);
  const decided = testsOnlyThePatient(expr) ? isChapterShown(content, variables) : undefined;
  return (
    <Alert color={decided === false ? "red" : "orange"} icon={<IconGitBranch size={16} />} mb="md" p="sm">
      <Text size="sm" component="div">
        {described && <DisplaysWhen condition={described} label="This Question Set is shown only when" />}
      </Text>
      <Text size="xs" c="dimmed" mt={4}>
        {decided === undefined
          ? "It depends on answers in earlier Question Sets, which this preview doesn't have. The questions below are shown as if it were met."
          : decided
            ? "The sample patient meets it, so they would see this Question Set."
            : "The sample patient doesn't meet it, so they wouldn't see this Question Set. The questions below are shown anyway."}
      </Text>
    </Alert>
  );
}
