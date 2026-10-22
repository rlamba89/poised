// Where the trust uses each question key (plan C4, session S23), from Go's
// GET …/questionnaires/:id/question-keys:
//   - questions: for each patient question, the trust's other questionnaires it pre-fills from,
//     matched by link (the same key) or by the same wording (PRE-05);
//   - problems: the key problems Publish would refuse (a key used twice; a published question whose
//     answer changed), which the Publish window shows before the author clicks.
// Go works it all out; this file only fetches it and draws the label on a question's card.
import { useCallback, useEffect, useState } from "react";
import { Badge, List, Popover, ScrollArea, Text } from "@mantine/core";
import { useWindowEvent } from "@mantine/hooks";
import { IconHistory } from "@tabler/icons-react";
import type { PublishProblem } from "@poised/clinical";
import { api } from "@/lib/api";

export type KeyMatch = { questionnaireId: string; name: string; by: "key" | "wording" };
export type QuestionKeys = { questions: Record<string, KeyMatch[]>; problems: PublishProblem[] };

/**
 * Fetches the keys when the editor opens, again whenever `refresh` changes (each time every change
 * is saved), and on `reload()` (the Publish window, as it opens and after a refusal). `current` is
 * false while a fetch is under way, so the Publish window waits for Go's problems. `path` null: the
 * user can't see them (clinicians), and there's no label.
 */
export function useQuestionKeys(path: string | null, refresh: string): { keys: QuestionKeys | null; current: boolean; reload: () => void } {
  const [keys, setKeys] = useState<QuestionKeys | null>(null);
  const [asked, setAsked] = useState(0);
  const [answered, setAnswered] = useState(-1);
  const reload = useCallback(() => setAsked((n) => n + 1), []);
  useEffect(() => {
    if (!path) return;
    let live = true;
    const request = asked;
    api<QuestionKeys>(path)
      .then((k) => live && setKeys(k))
      .catch(() => undefined) // no label rather than an error: publishing checks the keys again
      .finally(() => live && setAnswered(request));
    return () => {
      live = false;
    };
  }, [path, refresh, asked]);
  return { keys: path ? keys : null, current: !path || answered === asked, reload };
}

/** "Pre-fills from N other HQs", which lists them, or "New question: won't pre-fill". */
export function PrefillLabel({ matches }: { matches: KeyMatch[] }) {
  const [opened, setOpened] = useState(false);
  // Escape closes the list wherever the focus is (test report 3).
  useWindowEvent("keydown", (e) => {
    if (opened && e.key === "Escape") setOpened(false);
  });
  if (matches.length === 0) {
    return (
      <Badge size="sm" variant="outline" color="gray" radius="sm" data-testid="prefill-label">
        New question: won&apos;t pre-fill
      </Badge>
    );
  }
  return (
    <Popover position="bottom-start" withArrow width={320} opened={opened} onDismiss={() => setOpened(false)} returnFocus>
      <Popover.Target>
        <Badge
          component="button"
          type="button"
          size="sm"
          variant="light"
          color="teal"
          radius="sm"
          leftSection={<IconHistory size={10} />}
          style={{ cursor: "pointer" }}
          data-testid="prefill-label"
          aria-expanded={opened}
          onClick={(e: React.MouseEvent) => {
            e.stopPropagation(); // doesn't select the card
            setOpened((o) => !o);
          }}
        >
          Pre-fills from {matches.length} other HQ{matches.length === 1 ? "" : "s"}
        </Badge>
      </Popover.Target>
      <Popover.Dropdown onClick={(e) => e.stopPropagation()}>
        <Text size="sm" fw={600} mb={4}>A patient&apos;s earlier answer is copied from</Text>
        <ScrollArea.Autosize mah={280}>
          <List size="sm" spacing={2} data-testid="prefill-sources">
            {matches.map((m) => (
              <List.Item key={m.questionnaireId}>
                {m.name}{" "}
                <Text span size="xs" c="dimmed">· {m.by === "key" ? "by link" : "by same wording"}</Text>
              </List.Item>
            ))}
          </List>
        </ScrollArea.Autosize>
      </Popover.Dropdown>
    </Popover>
  );
}
