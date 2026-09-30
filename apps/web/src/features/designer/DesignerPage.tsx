"use client";
// The chapter designer: SurveyJS Creator with autosave (LCY-03), undo/redo (LCY-05)
// and the stale-revision guard (LCY-04).
import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { Alert, Anchor, Badge, Box, Button, Center, Group, Loader, Text } from "@mantine/core";
import { SurveyCreatorComponent, type SurveyCreator } from "survey-creator-react";
import "survey-core/survey-core.css";
import "survey-creator-core/survey-creator-core.css";
import { api, ApiError } from "@/lib/api";
import { hasRole, useMe } from "@/lib/auth";
import { createChapterCreator } from "./creator";

export type ChapterDetail = {
  id: string;
  name: string;
  content: object;
  revision: number;
  versionStatus: string;
  questionnaireId: string;
  questionnaireName: string;
};

type SaveStatus = "idle" | "modified" | "saving" | "saved" | "error" | "conflict";

const STATUS_LABEL: Record<SaveStatus, { text: string; color: string }> = {
  idle: { text: "All changes saved", color: "gray" },
  modified: { text: "Unsaved changes", color: "yellow" },
  saving: { text: "Saving…", color: "blue" },
  saved: { text: "All changes saved", color: "green" },
  error: { text: "Not saved", color: "red" },
  conflict: { text: "Not saved", color: "red" },
};

export default function DesignerPage({ hospitalId, chapterId }: { hospitalId: string; chapterId: string }) {
  const [chapter, setChapter] = useState<ChapterDetail | null>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    api<ChapterDetail>(`/h/${hospitalId}/chapters/${chapterId}`)
      .then(setChapter)
      .catch((e: Error) => setError(e.message));
  }, [hospitalId, chapterId]);

  if (error) return <Alert color="red" m="md">{error}</Alert>;
  if (!chapter) {
    return (
      <Center h="50vh">
        <Loader />
      </Center>
    );
  }
  return <Designer hospitalId={hospitalId} chapter={chapter} />;
}

function Designer({ hospitalId, chapter }: { hospitalId: string; chapter: ChapterDetail }) {
  const me = useMe();
  const readOnly = !hasRole(me, hospitalId, "author") || chapter.versionStatus !== "draft";
  const [status, setStatus] = useState<SaveStatus>("idle");
  const [saveError, setSaveError] = useState("");
  const revision = useRef(chapter.revision);
  const queue = useRef<Promise<void>>(Promise.resolve());
  const [creator] = useState<SurveyCreator>(() => {
    const c = createChapterCreator({ readOnly });
    c.JSON = chapter.content;
    return c;
  });

  useEffect(() => {
    const onModified = () => setStatus((s) => (s === "conflict" ? s : "modified"));
    creator.onModified.add(onModified);

    // Saves run one at a time and always send the latest JSON with the latest revision.
    creator.saveSurveyFunc = (saveNo: number, callback: (no: number, ok: boolean) => void) => {
      queue.current = queue.current.then(async () => {
        setStatus("saving");
        try {
          const res = await api<{ revision: number }>(`/h/${hospitalId}/chapters/${chapter.id}/content`, {
            method: "PUT",
            body: JSON.stringify({ content: creator.JSON, revision: revision.current }),
          });
          revision.current = res.revision;
          callback(saveNo, true);
          setStatus("saved");
        } catch (e) {
          callback(saveNo, false);
          setSaveError((e as Error).message);
          if (e instanceof ApiError && e.status === 409) {
            // Someone else saved first: stop editing so nothing more is lost.
            creator.readOnly = true;
            setStatus("conflict");
          } else {
            setStatus("error");
          }
        }
      });
    };
    return () => creator.onModified.remove(onModified);
  }, [creator, hospitalId, chapter.id]);

  // Warn before leaving with unsaved changes.
  useEffect(() => {
    const unsaved = status === "modified" || status === "saving" || status === "error";
    if (!unsaved) return;
    const warn = (e: BeforeUnloadEvent) => e.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [status]);

  const label = STATUS_LABEL[status];
  const base = `/h/${hospitalId}`;
  return (
    <Box style={{ display: "flex", flexDirection: "column", height: "calc(100vh - 56px)" }}>
      <Group justify="space-between" px="md" py={8} style={{ borderBottom: "1px solid var(--mantine-color-gray-3)" }}>
        <Group gap="xs">
          <Anchor component={Link} href={`${base}/questionnaires/${chapter.questionnaireId}`} size="sm">
            ← {chapter.questionnaireName}
          </Anchor>
          <Text fw={600}>/ {chapter.name}</Text>
          {readOnly && <Badge color="gray">Read only</Badge>}
        </Group>
        <Group gap="xs">
          {!readOnly && <Badge color={label.color} variant="light" data-testid="save-status">{label.text}</Badge>}
          <Button size="xs" variant="light" component={Link} href={`${base}/chapters/${chapter.id}/preview`}>
            Preview
          </Button>
        </Group>
      </Group>
      {status === "conflict" && (
        <Alert color="red" radius={0} title="Someone else changed this chapter">
          {saveError}{" "}
          <Button size="xs" variant="white" color="red" onClick={() => window.location.reload()}>Reload</Button>
        </Alert>
      )}
      {status === "error" && (
        <Alert color="red" radius={0} title="Your latest changes are not saved">
          {saveError}{" "}
          <Button size="xs" variant="white" color="red" onClick={() => creator.saveSurvey()}>Try again</Button>
        </Alert>
      )}
      <Box style={{ flex: 1, minHeight: 0 }}>
        <SurveyCreatorComponent creator={creator} />
      </Box>
    </Box>
  );
}
