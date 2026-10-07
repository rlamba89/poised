"use client";
// One episode: the patient, the procedure, the patient's link, the Question Sets to validate
// (Step 4 of plan-workflow.md), the status and the General notes.
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  Alert, Anchor, Badge, Button, Card, Center, CopyButton, Group, Loader, SimpleGrid, Stack, Table, Text, TextInput, Title,
} from "@mantine/core";
import { IconArrowLeft, IconCheck, IconCopy } from "@tabler/icons-react";
import { ageFrom, patientVariables, shownSets } from "@poised/clinical";
import { AUDIENCE_LABELS } from "@/features/chapters/types";
import { api } from "@/lib/api";
import { formatDate, formatDateTime } from "@/lib/format";
import { StatusBadge } from "./EpisodeList";
import { GeneralNotes } from "./GeneralNotes";
import { currentAnswers, useEpisodeHQ } from "./hq";
import { StatusSelect } from "./StatusSelect";
import { patientName, type Episode, type EpisodeEvent } from "./types";

/** patientToken is the patient's link token; null when the link can't be shown (made before
 * links were sealed): the clinician then makes a new one. */
export type EpisodeDetail = { episode: Episode; events: EpisodeEvent[]; patientToken: string | null };

/** Loads an episode and its events; `reload` refetches both. */
export function useEpisode(base: string, episodeId: string) {
  const [data, setData] = useState<EpisodeDetail | null>(null);
  const [error, setError] = useState("");
  const reload = useCallback(
    () =>
      api<EpisodeDetail>(`${base}/episodes/${episodeId}`)
        .then(setData)
        .catch((e: Error) => setError(e.message)),
    [base, episodeId],
  );
  useEffect(() => {
    reload();
  }, [reload]);
  return { data, error, reload };
}

export function EpisodePage({ base, episodeId }: { base: string; episodeId: string }) {
  const { data, error, reload } = useEpisode(base, episodeId);
  const path = `${base}/episodes/${episodeId}`;
  if (!data) return error ? <Alert color="red">{error}</Alert> : <Center h="40vh"><Loader /></Center>;
  const e = data.episode;

  return (
    <Stack maw={960} mx="auto">
      <Anchor component={Link} href={`${base}/episodes`} size="sm">
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

      {e.status === "hq_not_complete" && <PatientLink path={path} token={data.patientToken} onChanged={reload} />}

      <QuestionSets base={base} episode={e} onReviewed={reload} />

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
/** The patient's link, to copy and send, and a way to replace it (the old one stops working). */
function PatientLink({ path, token, onChanged }: { path: string; token: string | null; onChanged: () => void }) {
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const link = token && typeof window !== "undefined" ? `${window.location.origin}/p/${token}` : "";

  const replace = async () => {
    setBusy(true);
    setError("");
    try {
      await api(`${path}/patient-link`, { method: "POST" });
      setConfirming(false);
      onChanged();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card withBorder>
      <Text fw={600} mb={4}>Patient link</Text>
      <Text size="sm" c="dimmed" mb="xs">
        Send this link to the patient. They also need their date of birth to open it.
      </Text>
      {error && <Alert color="red" mb="xs">{error}</Alert>}
      {link ? (
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
      ) : (
        <Text size="sm">This link was made before links were stored securely, so it can&apos;t be shown. Make a new one to send.</Text>
      )}
      <Group gap="xs" mt="xs">
        {confirming ? (
          <>
            <Text size="sm">The old link will stop working.</Text>
            <Button size="xs" color="red" onClick={replace} loading={busy}>Make a new link</Button>
            <Button size="xs" variant="default" onClick={() => setConfirming(false)}>Cancel</Button>
          </>
        ) : (
          <Button size="xs" variant="subtle" onClick={() => setConfirming(true)}>Make a new link</Button>
        )}
      </Group>
    </Card>
  );
}

function QuestionSets({ base, episode, onReviewed }: { base: string; episode: Episode; onReviewed: () => void }) {
  const { hq, error, reload } = useEpisodeHQ(base, episode.id);
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
      await api(`${base}/episodes/${episode.id}/complete-review`, { method: "POST" });
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
                    <Button size="xs" variant={reviewing && !row?.validatedAt ? "filled" : "default"} component={Link} href={`${base}/episodes/${episode.id}/sets/${s.id}`}>
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
