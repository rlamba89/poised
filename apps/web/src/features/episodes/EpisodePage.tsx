"use client";
// One episode: the patient, the procedure, the patient's link, the Question Sets to validate
// (Step 4 of plan-workflow.md), the status and the General notes.
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  Alert, Anchor, Badge, Button, Card, Center, CopyButton, Group, Loader, SimpleGrid, Stack, Table, Text, TextInput, Title,
} from "@mantine/core";
import { IconArrowLeft, IconCheck, IconCopy } from "@tabler/icons-react";
import { ageFrom, patientVariables, shownSets } from "@sj/clinical";
import { AUDIENCE_LABELS } from "@/features/chapters/types";
import { api } from "@/lib/api";
import { formatDate, formatDateTime } from "@/lib/format";
import { StatusBadge } from "./EpisodeList";
import { GeneralNotes } from "./GeneralNotes";
import { currentAnswers, useEpisodeHQ } from "./hq";
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

      <QuestionSets hospitalId={hospitalId} episode={e} onReviewed={reload} />

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

/** The Question Sets shown for this patient, each with its validation state, and "Complete HQ review". */
function QuestionSets({ hospitalId, episode, onReviewed }: { hospitalId: string; episode: Episode; onReviewed: () => void }) {
  const { hq, error, reload } = useEpisodeHQ(hospitalId, episode.id);
  const [completing, setCompleting] = useState(false);
  const [completeError, setCompleteError] = useState("");
  if (error) return <Alert color="red">{error}</Alert>;
  if (!hq) return null;

  const answers = Object.fromEntries(hq.chapters.map((c) => [c.id, currentAnswers(hq, c.id)]));
  const sets = shownSets(hq.chapters, answers, "clinician", patientVariables(episode));
  const waiting = episode.status === "hq_not_complete";
  const reviewing = episode.status === "ready_for_review";
  const allValidated = sets.every((s) => hq.clinician[s.id]?.validatedAt);

  const complete = async () => {
    setCompleting(true);
    try {
      await api(`/h/${hospitalId}/episodes/${episode.id}/complete-review`, { method: "POST" });
      setCompleteError("");
      onReviewed();
      reload();
    } catch (e) {
      setCompleteError((e as Error).message);
    } finally {
      setCompleting(false);
    }
  };

  return (
    <Card withBorder>
      <Group justify="space-between" mb="xs">
        <Title order={4}>Health questionnaire</Title>
        {reviewing && (
          <Button onClick={complete} loading={completing} disabled={!allValidated}>Complete HQ review</Button>
        )}
      </Group>
      {waiting && <Text size="sm" c="dimmed" mb="xs">Waiting for the patient to send their answers. Validation starts once they have.</Text>}
      {reviewing && !allValidated && <Text size="sm" c="dimmed" mb="xs">Validate every Question Set to complete the HQ review.</Text>}
      {episode.reviewCompletedAt && (
        <Text size="sm" mb="xs">HQ review completed by {episode.reviewCompletedByName} on {formatDateTime(episode.reviewCompletedAt)}.</Text>
      )}
      {completeError && <Alert color="red" mb="xs">{completeError}</Alert>}
      <Table verticalSpacing="xs">
        <Table.Tbody>
          {sets.map((s) => {
            const row = hq.clinician[s.id];
            return (
              <Table.Tr key={s.id} data-testid={`hq-set-${s.name}`}>
                <Table.Td>
                  <Text fw={600}>{s.name}</Text>
                </Table.Td>
                <Table.Td>
                  <Badge variant="outline" color={s.audience === "patient" ? "gray" : "violet"} size="sm">{AUDIENCE_LABELS[s.audience]}</Badge>
                </Table.Td>
                <Table.Td>
                  {row?.validatedAt ? (
                    <Text size="sm" c="green.8">Validated by {row.updatedByName} on {formatDateTime(row.validatedAt)}</Text>
                  ) : (
                    <Text size="sm" c="dimmed">{waiting ? "Waiting for the patient" : row ? "In progress" : "Not started"}</Text>
                  )}
                </Table.Td>
                <Table.Td ta="right">
                  {!waiting && (
                    <Button size="xs" variant={reviewing && !row?.validatedAt ? "filled" : "default"} component={Link} href={`/h/${hospitalId}/episodes/${episode.id}/sets/${s.id}`}>
                      {reviewing ? (row?.validatedAt ? "Review" : "Validate") : "View"}
                    </Button>
                  )}
                </Table.Td>
              </Table.Tr>
            );
          })}
        </Table.Tbody>
      </Table>
    </Card>
  );
}
