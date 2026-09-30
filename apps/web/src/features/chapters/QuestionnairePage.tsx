"use client";
// A questionnaire and its ordered chapters (FRM-07): add, edit, reorder, delete.
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  ActionIcon, Alert, Anchor, Badge, Button, Card, Group, Modal, Stack, Text, TextInput, Title, Tooltip,
} from "@mantine/core";
import { IconArrowDown, IconArrowUp, IconGripVertical } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { hasRole, useMe } from "@/lib/auth";
import { formatDateTime } from "@/lib/format";
import { chapterIcon } from "./icons";
import { ChapterEditModal, AUDIENCE_LABELS, type Chapter } from "./ChapterEditModal";

type Detail = {
  questionnaire: { id: string; name: string; description: string; versionNo: number; status: string; updatedAt: string; updatedByName: string };
  chapters: Chapter[];
};

export function QuestionnairePage({ hospitalId, questionnaireId }: { hospitalId: string; questionnaireId: string }) {
  const me = useMe();
  const canEdit = hasRole(me, hospitalId, "author");
  const [data, setData] = useState<Detail | null>(null);
  const [error, setError] = useState("");
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState<Chapter | null>(null);
  const [deleting, setDeleting] = useState<Chapter | null>(null);
  const [dragId, setDragId] = useState<string | null>(null);
  const base = `/h/${hospitalId}`;

  const load = useCallback(() => {
    api<Detail>(`${base}/questionnaires/${questionnaireId}`)
      .then(setData)
      .catch((e: Error) => setError(e.message));
  }, [base, questionnaireId]);
  useEffect(load, [load]);

  const run = async (fn: () => Promise<unknown>) => {
    setError("");
    try {
      await fn();
    } catch (e) {
      setError((e as Error).message);
    }
    load();
  };

  const saveOrder = (ids: string[]) => {
    if (!data) return;
    // Show the new order straight away; the reload after saving confirms it.
    setData({ ...data, chapters: ids.map((id) => data.chapters.find((c) => c.id === id)!) });
    run(() => api(`${base}/questionnaires/${questionnaireId}/chapter-order`, { method: "PUT", body: JSON.stringify(ids) }));
  };
  const move = (from: number, to: number) => {
    if (!data || to < 0 || to >= data.chapters.length || from === to) return;
    const ids = data.chapters.map((c) => c.id);
    const [id] = ids.splice(from, 1);
    ids.splice(to, 0, id);
    saveOrder(ids);
  };

  if (!data) return error ? <Alert color="red">{error}</Alert> : null;
  const q = data.questionnaire;

  return (
    <Stack maw={900}>
      <Anchor component={Link} href={`${base}/questionnaires`} size="sm">← All questionnaires</Anchor>
      <Group justify="space-between" align="flex-start">
        <div>
          <Title order={3}>{q.name}</Title>
          <Text c="dimmed">{q.description}</Text>
          <Text size="sm" c="dimmed" mt={4} component="div">
            Version {q.versionNo} <Badge variant="light" size="sm">{q.status}</Badge> · last changed {formatDateTime(q.updatedAt)} by {q.updatedByName}
          </Text>
        </div>
        {canEdit && <Button onClick={() => setAdding(true)}>Add chapter</Button>}
      </Group>
      {error && <Alert color="red">{error}</Alert>}
      {data.chapters.length === 0 && <Text c="dimmed">No chapters yet.</Text>}

      {data.chapters.map((c, i) => {
        const Icon = chapterIcon(c.icon);
        return (
          <Card
            key={c.id}
            withBorder
            padding="sm"
            draggable={canEdit}
            onDragStart={() => setDragId(c.id)}
            onDragOver={(e) => e.preventDefault()}
            onDrop={() => {
              if (dragId) move(data.chapters.findIndex((x) => x.id === dragId), i);
              setDragId(null);
            }}
            style={{ opacity: dragId === c.id ? 0.5 : 1 }}
          >
            <Group justify="space-between" wrap="nowrap">
              <Group wrap="nowrap">
                {canEdit && <IconGripVertical size={18} style={{ cursor: "grab", color: "var(--mantine-color-gray-5)" }} />}
                <Icon size={28} stroke={1.5} />
                <div>
                  <Group gap="xs">
                    <Text fw={600}>{c.name}</Text>
                    <Badge variant="outline" size="sm">{AUDIENCE_LABELS[c.audience]}</Badge>
                  </Group>
                  {c.description && <Text size="sm" c="dimmed">{c.description}</Text>}
                </div>
              </Group>
              <Group gap="xs" wrap="nowrap">
                {canEdit && (
                  <>
                    <Tooltip label="Move up">
                      <ActionIcon variant="subtle" aria-label="Move up" disabled={i === 0} onClick={() => move(i, i - 1)}>
                        <IconArrowUp size={16} />
                      </ActionIcon>
                    </Tooltip>
                    <Tooltip label="Move down">
                      <ActionIcon variant="subtle" aria-label="Move down" disabled={i === data.chapters.length - 1} onClick={() => move(i, i + 1)}>
                        <IconArrowDown size={16} />
                      </ActionIcon>
                    </Tooltip>
                    <Button size="xs" variant="default" onClick={() => setEditing(c)}>Edit</Button>
                    <Button size="xs" component={Link} href={`${base}/chapters/${c.id}/design`}>Design</Button>
                  </>
                )}
                <Button size="xs" variant="light" component={Link} href={`${base}/chapters/${c.id}/preview`}>Preview</Button>
                {canEdit && (
                  <Button size="xs" variant="subtle" color="red" onClick={() => setDeleting(c)}>Delete</Button>
                )}
              </Group>
            </Group>
          </Card>
        );
      })}

      <AddChapterModal
        opened={adding}
        onClose={() => setAdding(false)}
        onAdd={(name) =>
          run(async () => {
            await api(`${base}/questionnaires/${questionnaireId}/chapters`, { method: "POST", body: JSON.stringify({ name }) });
            setAdding(false);
          })
        }
      />
      <ChapterEditModal
        chapter={editing}
        onClose={() => setEditing(null)}
        onSave={(patch) =>
          run(async () => {
            await api(`${base}/chapters/${editing!.id}`, { method: "PATCH", body: JSON.stringify(patch) });
            setEditing(null);
          })
        }
      />
      <Modal opened={!!deleting} onClose={() => setDeleting(null)} title="Delete chapter?">
        <Stack>
          <Text>“{deleting?.name}” and all its questions will be deleted. This can&apos;t be undone.</Text>
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setDeleting(null)}>Cancel</Button>
            <Button
              color="red"
              onClick={() =>
                run(async () => {
                  await api(`${base}/chapters/${deleting!.id}`, { method: "DELETE" });
                  setDeleting(null);
                })
              }
            >
              Delete
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}

function AddChapterModal(props: { opened: boolean; onClose: () => void; onAdd: (name: string) => void }) {
  const [name, setName] = useState("");
  useEffect(() => {
    if (props.opened) setName("");
  }, [props.opened]);
  return (
    <Modal opened={props.opened} onClose={props.onClose} title="Add chapter">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          props.onAdd(name);
        }}
      >
        <Stack>
          <TextInput label="Name" required value={name} onChange={(e) => setName(e.currentTarget.value)} data-autofocus />
          <Group justify="flex-end">
            <Button variant="default" onClick={props.onClose}>Cancel</Button>
            <Button type="submit">Add</Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  );
}
