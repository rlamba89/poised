// The episode's Information panel (S36, INF-04): the videos and leaflets this patient sees on their
// home, under Information. The list was copied from the procedure when the episode was made (S36
// answers 2); anyone at the hospital may add one of the library's active items for this patient
// only, or remove any. A retired item stays listed but the patient doesn't see it. A cancelled or
// archived episode's list is shown, not changed. Each change is in the General notes (onChanged
// reloads them).
import { useCallback, useEffect, useState } from "react";
import { Alert, Anchor, Button, Card, Group, Select, Stack, Text, Title } from "@mantine/core";
import { IconFileText, IconMovie } from "@tabler/icons-react";
import { formatLength } from "@poised/clinical";
import { api } from "@/lib/api";
import { WholeBadge } from "@/lib/WholeBadge";
import type { Episode } from "@/features/episodes/types";
import { addable, kindLabel, type EpisodeInfo } from "./info";

export function EpisodeInformation({ path, episode, onChanged }: { path: string; episode: Episode; onChanged: () => void }) {
  const [info, setInfo] = useState<EpisodeInfo | null>(null);
  const [loadError, setLoadError] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [search, setSearch] = useState("");
  const readOnly = episode.status === "cancelled" || episode.status === "archived";

  // Loaded again when the episode changes (its revision): a new procedure brings its own list.
  const load = useCallback(
    () =>
      api<EpisodeInfo>(`${path}/information`)
        .then((i) => {
          setInfo(i);
          setLoadError("");
        })
        .catch((e: Error) => setLoadError(e.message)),
    [path],
  );
  useEffect(() => {
    load();
  }, [load, episode.revision]);

  const change = async (key: string, request: () => Promise<unknown>) => {
    if (busy) return;
    setBusy(key);
    try {
      await request();
      setError("");
      await load();
      onChanged();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy("");
    }
  };
  const add = (id: string) => change("add", () => api(`${path}/information`, { method: "POST", body: JSON.stringify({ contentId: id }) }));
  const remove = (id: string) => change(id, () => api(`${path}/information/${id}`, { method: "DELETE" }));

  const choices = info ? addable(info.library, info.items.map((i) => i.id)).map((l) => ({ value: l.id, label: `${l.title} (${kindLabel(l.kind)})` })) : [];
  return (
    <Card withBorder data-testid="episode-information">
      <Title order={4}>Information for the patient</Title>
      <Text size="sm" c="dimmed" mb="xs">
        Videos and leaflets the patient sees on their home, with the hospital&apos;s phone number and email. Copied from the procedure when the
        episode was made; changes here are for this patient only.
      </Text>
      {loadError && <Alert color="red">{loadError}</Alert>}
      {info && (
        <Stack gap="xs">
          {info.items.length === 0 && <Text size="sm" c="dimmed">None.</Text>}
          {info.items.map((i) => {
            const Icon = i.kind === "video" ? IconMovie : IconFileText;
            return (
              <Group key={i.id} justify="space-between" wrap="nowrap" align="flex-start" data-testid={`episode-info-${i.title}`}>
                <Group gap="xs" wrap="nowrap" align="flex-start" style={{ minWidth: 0 }}>
                  <Icon size={18} style={{ flexShrink: 0, marginTop: 2 }} aria-label={kindLabel(i.kind)} />
                  <div style={{ minWidth: 0 }}>
                    <Group gap={6}>
                      <Anchor href={i.url} target="_blank" rel="noopener noreferrer" size="sm" style={{ overflowWrap: "anywhere" }}>{i.title}</Anchor>
                      {i.status === "retired" && (
                        <WholeBadge color="gray" variant="light" title="Retired in the content library: the patient doesn't see it">Retired</WholeBadge>
                      )}
                    </Group>
                    <Text size="xs" c="dimmed">
                      {kindLabel(i.kind)}
                      {i.lengthSeconds ? ` · ${formatLength(i.lengthSeconds)}` : ""}
                      {" · "}
                      {i.fromProcedure ? "From the procedure" : `Added for this patient${i.addedByName ? ` by ${i.addedByName}` : ""}`}
                    </Text>
                  </div>
                </Group>
                {!readOnly && (
                  <Button size="compact-xs" variant="subtle" color="red" loading={busy === i.id} onClick={() => remove(i.id)} aria-label={`Remove ${i.title}`}>
                    Remove
                  </Button>
                )}
              </Group>
            );
          })}
          {!readOnly && (
            <Select
              aria-label="Add a video or leaflet for this patient"
              placeholder={choices.length ? "Add a video or leaflet for this patient…" : "Nothing else in the content library"}
              data={choices}
              value={null}
              onChange={(v) => {
                if (v) add(v);
                setSearch("");
              }}
              searchable
              searchValue={search}
              onSearchChange={setSearch}
              disabled={choices.length === 0 || busy !== ""}
              maw={460}
              nothingFoundMessage="Nothing matches"
            />
          )}
          {error && <Alert color="red" p="xs">{error}</Alert>}
        </Stack>
      )}
    </Card>
  );
}
