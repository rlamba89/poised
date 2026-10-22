// "General notes" (incumbent POA Summary): the episode's events and clinicians' comments. An archived
// episode's are shown without "Add comment" (S22: read-only).
import { useState } from "react";
import { Alert, Button, Group, Stack, Text, Textarea } from "@mantine/core";
import { api } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import type { EpisodeEvent } from "./types";

export function GeneralNotes({ path, events, onAdded, readOnly = false }: {
  path: string; events: EpisodeEvent[]; onAdded: () => void; readOnly?: boolean;
}) {
  const [adding, setAdding] = useState(false);
  const [text, setText] = useState("");
  const [error, setError] = useState("");

  const save = async () => {
    try {
      await api(`${path}/notes`, { method: "POST", body: JSON.stringify({ text }) });
      setText("");
      setAdding(false);
      setError("");
      onAdded();
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <Stack gap={6}>
      {events.map((ev) => (
        <Group key={ev.id} gap={8} wrap="nowrap" align="baseline">
          <Text size="sm" c={ev.kind === "event" ? "blue.8" : undefined}>{ev.text}</Text>
          <Text size="xs" c="dimmed">
            | {ev.automatic ? "Automatic" : (ev.userName ?? "Patient")} on {formatDateTime(ev.createdAt)}
          </Text>
        </Group>
      ))}
      {adding ? (
        <Stack gap={6} mt={6}>
          {error && <Alert color="red">{error}</Alert>}
          <Textarea aria-label="Comment" autosize minRows={2} value={text} onChange={(e) => setText(e.currentTarget.value)} data-autofocus />
          <Group gap="xs">
            <Button size="xs" onClick={save} disabled={!text.trim()}>Save comment</Button>
            <Button size="xs" variant="default" onClick={() => setAdding(false)}>Cancel</Button>
          </Group>
        </Stack>
      ) : (
        !readOnly && <Button size="xs" mt={6} w="fit-content" onClick={() => setAdding(true)}>Add comment</Button>
      )}
    </Stack>
  );
}
