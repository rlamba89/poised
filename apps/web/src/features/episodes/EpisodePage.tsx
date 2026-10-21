// One episode: the patient, the procedure, the patient's invite (S14: resend it, or fill in the HQ
// on their behalf), the Question Sets to validate (Step 4 of plan-workflow.md), the status and the
// General notes.
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router";
import {
  Alert, Anchor, Badge, Button, Card, Center, Group, Loader, SimpleGrid, Stack, Table, Text, Title,
} from "@mantine/core";
import { IconArrowLeft } from "@tabler/icons-react";
import { ageFrom, patientVariables, shownSets } from "@poised/clinical";
import { AUDIENCE_LABELS } from "@/features/chapters/types";
import { api } from "@/lib/api";
import { nationalIdLabel } from "@/lib/config";
import { formatDate, formatDateTime, formatOptionalDate } from "@/lib/format";
import { EditProcedure } from "./EditProcedure";
import { StatusBadge } from "./EpisodeList";
import { GeneralNotes } from "./GeneralNotes";
import { currentAnswers, useEpisodeHQ } from "./hq";
import type { StatusEvent, StatusOption, UndoOption } from "./status";
import { StatusControl, StatusHistory } from "./StatusControl";
import { patientName, type Episode, type EpisodeEvent, type EpisodeInvite } from "./types";
import { undeliveredText, type Undelivered } from "./undelivered";

/**
 * GET …/episodes/:id. S20 added the status: `moves` (what this user may do from here), `undo`
 * (the last move made by hand, if it can be stepped back) and `statusHistory`.
 */
export type EpisodeDetail = {
  episode: Episode;
  events: EpisodeEvent[];
  invite: EpisodeInvite;
  undelivered: Undelivered[];
  moves: StatusOption[];
  undo: UndoOption;
  statusHistory: StatusEvent[];
};

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
  const [editing, setEditing] = useState(false);
  const path = `${base}/episodes/${episodeId}`;
  if (!data) return error ? <Alert color="red">{error}</Alert> : <Center h="40vh"><Loader /></Center>;
  const e = data.episode;
  const cancelled = e.status === "cancelled";
  // The reason as recorded, with any details (S20): the latest cancel in the history.
  const cancelledWhy = data.statusHistory.findLast((h) => h.move === "cancel")?.reason;

  return (
    <Stack maw={960} mx="auto">
      <Anchor component={Link} to={`${base}/episodes`} size="sm">
        <Group gap={4}><IconArrowLeft size={14} /> Episodes</Group>
      </Anchor>
      <Group justify="space-between">
        <Group gap="sm">
          <Title order={3}>
            <Anchor component={Link} to={`${base}/patients/${e.patientId}`} inherit>{patientName(e)}</Anchor>
          </Title>
          <StatusBadge status={e.status} />
          {e.notRespondingAt && !e.patientSubmittedAt && (
            // EPI-02 (S16): the HQ wasn't sent by the hospital's "not responding" day. The worklist is C3's.
            <Badge color="red" variant="light" data-testid="not-responding">Not responding</Badge>
          )}
          {data.undelivered.length > 0 && !e.patientSubmittedAt && (
            // OPS-05 (S17): a text or email of the latest invite, or a later reminder, didn't reach the patient.
            <Badge color="red" variant="light" data-testid="undelivered-badge">Invite not delivered</Badge>
          )}
        </Group>
        <Button component={Link} to={`${path}/poa`} variant="default">POA Summary</Button>
      </Group>
      {cancelled && (
        <Alert color="gray" title="This episode is cancelled" data-testid="cancelled">
          {cancelledWhy ? `Reason: ${cancelledWhy}. ` : ""}Cancelling is final: the patient&apos;s link no longer works, and their home shows the
          questionnaire as cancelled by the hospital.
        </Alert>
      )}

      <Card withBorder>
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="xs">
          <Detail label="Hospital number" value={e.hospitalNumber} />
          <Detail label="Procedure" value={e.procedure} />
          <Detail label={nationalIdLabel()} value={e.nationalId ?? ""} />
          <Detail label="Anaesthetic" value={e.anaesthetic} />
          <Detail label="Date of birth" value={`${formatDate(e.dateOfBirth)} (age ${ageFrom(e.dateOfBirth)})`} />
          <Detail label="Consultant" value={e.consultant} />
          <Detail label="Mobile" value={e.mobile} />
          <Detail label="Health questionnaire" value={`${e.questionnaireName} v${e.versionNo}`} />
          <Detail label="Email" value={e.email} />
          <Detail label="Created" value={formatDateTime(e.createdAt)} />
          <Detail label="Planned procedure date" value={e.plannedDate ? formatOptionalDate(e.plannedDate) : ""} />
          <Detail label="POA appointment date" value={e.poaDate ? formatOptionalDate(e.poaDate) : ""} />
          {e.admittedAt && <Detail label="Admission date" value={formatOptionalDate(e.admittedAt)} />}
          {e.dischargedAt && <Detail label="Discharge date" value={formatOptionalDate(e.dischargedAt)} />}
        </SimpleGrid>
        {/* The procedure and its dates can be changed until the HQ review is complete (S19), unless it's cancelled (S20). */}
        {!e.reviewCompletedAt && !cancelled && !editing && (
          <Group justify="flex-end" mt="xs">
            <Button size="xs" variant="default" onClick={() => setEditing(true)}>Edit procedure and dates</Button>
          </Group>
        )}
      </Card>
      {editing && (
        <EditProcedure path={path} episode={e} onCancel={() => setEditing(false)}
          onSaved={() => { setEditing(false); reload(); }} />
      )}

      {e.status === "hq_not_complete" && <Invite path={path} episode={e} invite={data.invite} undelivered={data.undelivered} onSent={reload} />}

      <QuestionSets base={base} episode={e} onReviewed={reload} />

      <Card withBorder>
        <StatusControl path={path} detail={data} onChanged={reload} />
      </Card>

      <Card withBorder>
        <Title order={4} mb="xs">Status history</Title>
        <StatusHistory events={data.statusHistory} />
      </Card>

      <Card withBorder>
        <Title order={4} mb="xs">General notes</Title>
        <GeneralNotes path={path} events={data.events} onAdded={reload} />
      </Card>
    </Stack>
  );
}

/**
 * The patient's invite while they haven't sent their answers (S14): when it was sent (the
 * General notes say how), whether five wrong dates of birth locked it, Resend, and filling in the
 * HQ on the patient's behalf (PAT-09).
 */
function Invite({ path, episode, invite, undelivered, onSent }: {
  path: string; episode: Episode; invite: EpisodeInvite; undelivered: Undelivered[]; onSent: () => void;
}) {
  const [sending, setSending] = useState(false);
  const [result, setResult] = useState<{ ok: boolean; text: string } | null>(null);
  const resend = async () => {
    setSending(true);
    try {
      const { message } = await api<{ message: string }>(`${path}/resend-invite`, { method: "POST" });
      setResult({ ok: true, text: message });
      onSent();
    } catch (err) {
      setResult({ ok: false, text: (err as Error).message });
      onSent(); // a failed send is in the General notes too
    } finally {
      setSending(false);
    }
  };
  return (
    <Card withBorder data-testid="invite">
      <Text fw={600} mb={4}>Patient&apos;s invite</Text>
      <Text size="sm" c="dimmed" mb="xs">
        {invite
          ? `Last sent ${formatDateTime(invite.sentAt)}. The patient opens the link in their text message or email, then confirms their date of birth. The General notes say how it was sent.`
          : "No invite has been sent. The General notes say why."}
      </Text>
      {episode.invitedAt && (
        <Text size="sm" c="dimmed" mb="xs" data-testid="reminders">
          Reminders go on the hospital&apos;s reminder days after the first invite, until the patient starts answering. The General
          notes list each one.
        </Text>
      )}
      {invite?.lockedAt && (
        <Alert color="red" mb="xs" title="The link is locked">
          The date of birth was entered wrongly 5 times ({formatDateTime(invite.lockedAt)}), so every link sent to the patient for this
          episode is locked, and no reminders go. Resend the invite to send them a new link.
        </Alert>
      )}
      {undelivered.length > 0 && (
        // S17 (OPS-05): AWS reported these undelivered, or they couldn't be sent at all.
        <Alert color="red" mb="xs" title="Not delivered" data-testid="undelivered">
          {undelivered.map((u) => (
            <Text size="sm" key={`${u.sentAt}-${u.kind}-${u.channel}`}>
              {undeliveredText(u)} (Sent {formatDateTime(u.sentAt)}.)
            </Text>
          ))}
          <Text size="sm" mt={4}>Check the patient&apos;s mobile number and email address, then resend the invite.</Text>
        </Alert>
      )}
      {result && <Alert color={result.ok ? "green" : "red"} mb="xs">{result.text}</Alert>}
      <Group gap="xs">
        <Button variant="default" onClick={resend} loading={sending} disabled={!episode.mobile && !episode.email}>
          {invite ? "Resend invite" : "Send invite"}
        </Button>
        <Button variant="default" component={Link} to={`${path}/patient-form`}>Fill in for the patient</Button>
      </Group>
    </Card>
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
      {episode.patientSubmittedByName && (
        <Text size="sm" mb="xs">Sent by {episode.patientSubmittedByName} on the patient&apos;s behalf on {formatDateTime(episode.patientSubmittedAt ?? "")}.</Text>
      )}
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
                  {hq.enteredBy[s.id] && <Text size="xs" c="orange.8">Patient&apos;s answers entered by {hq.enteredBy[s.id]}</Text>}
                </Table.Td>
                <Table.Td ta="right">
                  {!waiting && (
                    <Button size="xs" variant={reviewing && !row?.validatedAt ? "filled" : "default"} component={Link} to={`${base}/episodes/${episode.id}/sets/${s.id}`}>
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
