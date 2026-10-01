"use client";
// One episode: the patient, the procedure, the patient's link, the status and the General notes.
// Step 4 of plan-workflow.md adds the Question Sets to validate.
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  Alert, Anchor, Button, Card, Center, CopyButton, Group, Loader, SimpleGrid, Stack, Text, TextInput, Title,
} from "@mantine/core";
import { IconArrowLeft, IconCheck, IconCopy } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { ageFrom } from "@sj/clinical";
import { formatDate, formatDateTime } from "@/lib/format";
import { StatusBadge } from "./EpisodeList";
import { GeneralNotes } from "./GeneralNotes";
import { StatusSelect } from "./StatusSelect";
import { patientName, type Episode, type EpisodeEvent } from "./types";

export type EpisodeDetail = { episode: Episode; events: EpisodeEvent[] };

/** Loads an episode and its events; `reload` refetches both. */
export function useEpisode(hospitalId: string, episodeId: string) {
  const [data, setData] = useState<EpisodeDetail | null>(null);
  const [error, setError] = useState("");
  const reload = useCallback(
    () =>
      api<EpisodeDetail>(`/h/${hospitalId}/episodes/${episodeId}`)
        .then(setData)
        .catch((e: Error) => setError(e.message)),
    [hospitalId, episodeId],
  );
  useEffect(() => {
    reload();
  }, [reload]);
  return { data, error, reload };
}

export function EpisodePage({ hospitalId, episodeId }: { hospitalId: string; episodeId: string }) {
  const { data, error, reload } = useEpisode(hospitalId, episodeId);
  const path = `/h/${hospitalId}/episodes/${episodeId}`;
  if (!data) return error ? <Alert color="red">{error}</Alert> : <Center h="40vh"><Loader /></Center>;
  const e = data.episode;
  const link = typeof window === "undefined" ? "" : `${window.location.origin}/p/${e.patientToken}`;

  return (
    <Stack maw={960} mx="auto">
      <Anchor component={Link} href={`/h/${hospitalId}/episodes`} size="sm">
        <Group gap={4}><IconArrowLeft size={14} /> Episodes</Group>
      </Anchor>
      <Group justify="space-between">
        <Group gap="sm">
          <Title order={3}>{patientName(e)}</Title>
          <StatusBadge status={e.status} />
        </Group>
        <Button component={Link} href={`${path}/poa`} variant="default">POA Summary</Button>
      </Group>

      <Card withBorder>
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="xs">
          <Detail label="Hospital number" value={e.hospitalNumber} />
          <Detail label="Procedure" value={e.procedure} />
          <Detail label="Date of birth" value={`${formatDate(e.dateOfBirth)} (age ${ageFrom(e.dateOfBirth)})`} />
          <Detail label="Anaesthetic" value={e.anaesthetic} />
          <Detail label="Phone number" value={e.phone} />
          <Detail label="Consultant" value={e.consultant} />
          <Detail label="Health questionnaire" value={`${e.questionnaireName} v${e.versionNo}`} />
          <Detail label="Created" value={formatDateTime(e.createdAt)} />
        </SimpleGrid>
      </Card>

      {e.status === "hq_not_complete" && (
        <Card withBorder>
          <Text fw={600} mb={4}>Patient link</Text>
          <Text size="sm" c="dimmed" mb="xs">
            Send this link to the patient. Anyone who has it can fill in the HQ, so share it only with the patient.
          </Text>
          <Group gap="xs" wrap="nowrap">
            <TextInput readOnly value={link} style={{ flex: 1 }} aria-label="Patient link" />
            <CopyButton value={link}>
              {({ copied, copy }) => (
                <Button variant="default" leftSection={copied ? <IconCheck size={14} /> : <IconCopy size={14} />} onClick={copy}>
                  {copied ? "Copied" : "Copy"}
                </Button>
              )}
            </CopyButton>
          </Group>
        </Card>
      )}

      <Card withBorder>
        <StatusSelect path={path} status={e.status} onChanged={reload} />
      </Card>

      <Card withBorder>
        <Title order={4} mb="xs">General notes</Title>
        <GeneralNotes path={path} events={data.events} onAdded={reload} />
      </Card>
    </Stack>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <Group gap="xs" wrap="nowrap" align="baseline">
      <Text size="sm" fw={600} w={170}>{label}</Text>
      <Text size="sm">{value || "—"}</Text>
    </Group>
  );
}
