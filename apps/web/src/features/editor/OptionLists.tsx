"use client";
// Reusable option lists (OPT-07): save a question's options under a name for the hospital, and
// copy a saved list into another question. Copies get new IDs, so a question never changes
// when the list does.
import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { ActionIcon, Alert, Badge, Button, Group, Modal, Paper, SegmentedControl, Stack, Text, TextInput, Tooltip } from "@mantine/core";
import { IconTrash } from "@tabler/icons-react";
import { applySavedOptions, toSavedOptions, type ChapterJson, type ElementJson, type SavedOption } from "@poised/clinical";
import { api } from "@/lib/api";

type OptionList = { id: string; name: string; options: SavedOption[]; updatedAt: string; updatedByName: string };

function useOptionLists() {
  const { hospitalId } = useParams<{ hospitalId: string }>();
  const base = `/h/${hospitalId}/option-lists`;
  const [lists, setLists] = useState<OptionList[] | null>(null);
  const [error, setError] = useState("");
  const reload = useCallback(
    () =>
      api<OptionList[]>(base)
        .then(setLists)
        .catch((e: Error) => setError(e.message)),
    [base],
  );
  return { base, lists, error, setError, reload };
}

/** "Use a saved list": pick a list, then add its options or replace the current ones. */
export function UseOptionList({ el, opened, onClose, change }: { el: ElementJson; opened: boolean; onClose: () => void; change: (e: (d: ChapterJson) => ChapterJson) => void }) {
  const { base, lists, error, setError, reload } = useOptionLists();
  const [mode, setMode] = useState<"add" | "replace">("add");
  useEffect(() => {
    if (opened) reload();
  }, [opened, reload]);

  const remove = async (l: OptionList) => {
    setError("");
    try {
      await api(`${base}/${l.id}`, { method: "DELETE" });
      reload();
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <Modal opened={opened} onClose={onClose} title="Use a saved option list" size="lg">
      <Stack>
        <SegmentedControl
          value={mode}
          onChange={(v) => setMode(v as "add" | "replace")}
          data={[
            { value: "add", label: "Add to the current options" },
            { value: "replace", label: "Replace the current options" },
          ]}
        />
        {mode === "replace" && (
          <Alert color="orange" p="xs">
            <Text size="sm">The current options go, with their disclosures. Conditions that test them will stop working.</Text>
          </Alert>
        )}
        {error && <Alert color="red" p="xs"><Text size="sm">{error}</Text></Alert>}
        {lists?.length === 0 && <Text size="sm" c="dimmed">No saved lists yet. Use “Save as a list” on a question&apos;s options to make one.</Text>}
        {lists?.map((l) => (
          <Paper key={l.id} withBorder p="sm">
            <Group justify="space-between" wrap="nowrap" align="flex-start">
              <div style={{ minWidth: 0 }}>
                <Text fw={600}>{l.name}</Text>
                <Text size="xs" c="dimmed" mb={4}>
                  {l.options.length} option{l.options.length === 1 ? "" : "s"} · {l.updatedByName}, {new Date(l.updatedAt).toLocaleDateString("en-GB")}
                </Text>
                <Group gap={4}>
                  {l.options.slice(0, 8).map((o, i) => (
                    <Badge key={i} variant="light" color="gray" tt="none" fw={400}>{o.text}</Badge>
                  ))}
                  {l.options.length > 8 && <Text size="xs" c="dimmed">+{l.options.length - 8} more</Text>}
                </Group>
              </div>
              <Group gap={4} wrap="nowrap">
                <Button
                  size="xs"
                  onClick={() => {
                    change((d) => applySavedOptions(d, el.name, l.options, mode));
                    onClose();
                  }}
                >
                  {mode === "add" ? "Add" : "Replace"}
                </Button>
                <Tooltip label="Delete this saved list. Questions that used it keep their options.">
                  <ActionIcon variant="subtle" color="red" aria-label={`Delete ${l.name}`} onClick={() => remove(l)}>
                    <IconTrash size={14} />
                  </ActionIcon>
                </Tooltip>
              </Group>
            </Group>
          </Paper>
        ))}
      </Stack>
    </Modal>
  );
}

/** "Save as a list": this question's options under a name. Saving under an existing name updates that list. */
export function SaveOptionList({ el, opened, onClose }: { el: ElementJson; opened: boolean; onClose: () => void }) {
  const { base, lists, error, setError, reload } = useOptionLists();
  const [name, setName] = useState("");
  const [saved, setSaved] = useState("");
  useEffect(() => {
    if (!opened) return;
    reload();
    setName("");
    setSaved("");
    setError("");
  }, [opened, reload, setError]);
  const existing = lists?.find((l) => l.name.toLowerCase() === name.trim().toLowerCase());
  const options = toSavedOptions(el);

  const save = async () => {
    setError("");
    const body = JSON.stringify({ name: name.trim(), options });
    try {
      if (existing) await api(`${base}/${existing.id}`, { method: "PUT", body });
      else await api(base, { method: "POST", body });
      setSaved(name.trim());
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <Modal opened={opened} onClose={onClose} title="Save these options as a list">
      {saved ? (
        <Stack>
          <Text size="sm">Saved “{saved}”. Any Select One or Select Many question in this hospital can now use it.</Text>
          <Group justify="flex-end">
            <Button onClick={onClose}>Done</Button>
          </Group>
        </Stack>
      ) : (
        <Stack>
          <Text size="sm" c="dimmed">
            {options.length} option{options.length === 1 ? "" : "s"}, with their scores and disclosures. Conditions on options aren&apos;t saved.
          </Text>
          <TextInput label="Name" placeholder="e.g. Anaesthetic types" value={name} onChange={(e) => setName(e.currentTarget.value)} data-autofocus />
          {existing && (
            <Alert color="orange" p="xs">
              <Text size="sm">A list with this name exists. Saving replaces it; questions that already used it keep their options.</Text>
            </Alert>
          )}
          {error && <Alert color="red" p="xs"><Text size="sm">{error}</Text></Alert>}
          <Group justify="flex-end">
            <Button variant="default" onClick={onClose}>Cancel</Button>
            <Button disabled={!name.trim() || lists === null} onClick={save}>{existing ? "Replace saved list" : "Save list"}</Button>
          </Group>
        </Stack>
      )}
    </Modal>
  );
}
