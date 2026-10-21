// Step 5 of plan-workflow.md: the POA Summary, laid out like Lifebox's (layout, not styling).
// "Validated summary" is built from the clinician's answers, which are final, and marks what the
// clinician changed; "Patient answers" is the same document from the patient's original answers.
import { useMemo, useState } from "react";
import { Link } from "react-router";
import {
  Accordion, Alert, Anchor, Badge, Breadcrumbs, Button, Card, Center, Group, List, Loader, Select, SimpleGrid, Stack, Table, Tabs, Text,
  Title, Tooltip,
} from "@mantine/core";
import { IconPrinter } from "@tabler/icons-react";
import { ageFrom, patientVariables, poaSummary, reportedBmi, type PoaCapture, type PoaSet } from "@poised/clinical";
import { api } from "@/lib/api";
import { nationalIdLabel } from "@/lib/config";
import { formatDate, formatDateTime, formatOptionalDate } from "@/lib/format";
import { StatusBadge } from "./EpisodeList";
import { useEpisode } from "./EpisodePage";
import { GeneralNotes } from "./GeneralNotes";
import { currentAnswers, useEpisodeHQ, type EpisodeHQ } from "./hq";
import { SaveError, isStale, StatusControl } from "./StatusControl";
import { patientName, type Episode, type EpisodeEvent } from "./types";

type Tab = "validated" | "patient";

export default function PoaSummary({ base, episodeId }: { base: string; episodeId: string }) {
  const { data, error, reload } = useEpisode(base, episodeId);
  const { hq, error: hqError } = useEpisodeHQ(base, episodeId);
  const [tab, setTab] = useState<Tab>("validated");
  if (error || hqError) return <Alert color="red">{error || hqError}</Alert>;
  if (!data || !hq) return <Center h="40vh"><Loader /></Center>;
  const e = data.episode;
  const path = `${base}/episodes/${episodeId}`;

  return (
    <Stack maw={960} mx="auto">
      <Breadcrumbs className="sj-no-print">
        <Anchor component={Link} to={`${base}/episodes`} size="sm">Episodes</Anchor>
        <Anchor component={Link} to={path} size="sm">Episode</Anchor>
        <Text size="sm" fw={600}>POA summary</Text>
      </Breadcrumbs>
      <Group gap="md">
        <Anchor component={Link} to={path} fw={700} underline="always">{patientName(e)}</Anchor>
        <StatusBadge status={e.status} />
        <Text size="sm">Hospital number: <b>{e.hospitalNumber || "—"}</b></Text>
        <Text size="sm">{nationalIdLabel()}: <b>{e.nationalId || "—"}</b></Text>
        <Text size="sm">Mobile: <b>{e.mobile || "—"}</b></Text>
      </Group>
      <Group justify="space-between">
        <Title order={2}>POA Summary</Title>
        <Button className="sj-no-print" leftSection={<IconPrinter size={16} />} onClick={() => window.print()}>Print</Button>
      </Group>

      <Card withBorder padding="lg">
        <Stack>
          <div className="sj-no-print">
            <StatusControl path={path} detail={data} onChanged={reload} />
          </div>
          <Tabs value={tab} onChange={(v) => setTab((v as Tab) ?? "validated")}>
            <Tabs.List>
              <Tabs.Tab value="validated">Validated summary</Tabs.Tab>
              <Tabs.Tab value="patient">Patient answers</Tabs.Tab>
            </Tabs.List>
          </Tabs>
          <Summary key={tab} tab={tab} episode={e} events={data.events} hq={hq} path={path} onChanged={reload} />
        </Stack>
      </Card>
    </Stack>
  );
}

type SummaryProps = { tab: Tab; episode: Episode; events: EpisodeEvent[]; hq: EpisodeHQ; path: string; onChanged: () => void };

function Summary({ tab, episode: e, events, hq, path, onChanged }: SummaryProps) {
  const vars = useMemo(() => patientVariables(e), [e]);
  const validated = tab === "validated";
  const sets = useMemo(() => {
    if (validated) {
      const answers = Object.fromEntries(hq.chapters.map((c) => [c.id, currentAnswers(hq, c.id)]));
      return poaSummary(hq.chapters, answers, "clinician", vars, hq.patient);
    }
    return poaSummary(hq.chapters.filter((c) => c.audience === "patient"), hq.patient, "patient", vars);
  }, [validated, hq, vars]);
  const bmi = reportedBmi(sets);

  return (
    <Stack>
      {validated && !e.reviewCompletedAt && (
        <Alert color="orange" className="sj-no-print">The HQ review isn&apos;t complete yet. This shows the answers as they stand.</Alert>
      )}
      {!validated && !e.patientSubmittedAt && (
        <Alert color="orange" className="sj-no-print">The patient hasn&apos;t sent their answers yet. This shows what they have saved so far.</Alert>
      )}
      <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="xs" verticalSpacing={6}>
        <Stack gap={6}>
          <Detail label="Patient name" value={patientName(e)} />
          <Detail label="Hospital number" value={e.hospitalNumber} />
          <Detail label={nationalIdLabel()} value={e.nationalId ?? ""} />
          <Detail label="Date of birth" value={formatDate(e.dateOfBirth)} />
          <Detail label="Age" value={String(ageFrom(e.dateOfBirth))} />
          <Detail label="Mobile" value={e.mobile} />
          <Detail label="Consultant" value={e.consultant} />
        </Stack>
        <Stack gap={6}>
          <Detail label="Procedure" value={e.procedure} />
          <Detail label="Planned procedure date" value={e.plannedDate ? formatOptionalDate(e.plannedDate) : ""} />
          <Detail label="POA appointment date" value={e.poaDate ? formatOptionalDate(e.poaDate) : ""} />
          <Detail label="Anaesthetic" value={e.anaesthetic} />
          <Detail label="Patient reported BMI" value={bmi === undefined ? "" : String(bmi)} />
          <Detail label="Anaesthetist ASA grade" value={asaLabel(e.anaesthetistAsa)} />
          <Detail label="Nurse ASA grade" value={asaLabel(e.nurseAsa)} />
        </Stack>
      </SimpleGrid>

      <Accordion multiple defaultValue={["notes", ...sets.map((s) => s.id), "asa"]} chevronPosition="right" variant="default">
        <Accordion.Item value="notes">
          <Accordion.Control><Title order={4} c="blue.9">General notes</Title></Accordion.Control>
          <Accordion.Panel>
            <GeneralNotes path={path} events={events} onAdded={onChanged} />
          </Accordion.Panel>
        </Accordion.Item>

        {sets.map((s) => (
          <Accordion.Item key={s.id} value={s.id} data-testid={`poa-set-${s.name}`}>
            <Accordion.Control><Title order={4} c="blue.9">{s.name}</Title></Accordion.Control>
            <Accordion.Panel>
              <SetSection set={s} byLine={byLine(validated, e, hq, s.id)} validated={validated} />
            </Accordion.Panel>
          </Accordion.Item>
        ))}

        {validated && (
          <Accordion.Item value="asa">
            <Accordion.Control><Title order={4} c="blue.9">Pre operative assessment ASA grades</Title></Accordion.Control>
            <Accordion.Panel>
              <AsaGrades episode={e} path={path} onSaved={onChanged} />
            </Accordion.Panel>
          </Accordion.Item>
        )}
      </Accordion>
      {sets.length === 0 && <Text c="dimmed">No answers produce anything for the summary yet.</Text>}
    </Stack>
  );
}

/**
 * "Validated by X on …" for the clinician's answers; "Patient on …" for the patient's, or
 * "Patient (entered by X) on …" where staff entered them on the patient's behalf (PAT-09).
 */
function byLine(validated: boolean, e: Episode, hq: EpisodeHQ, setId: string): string {
  const row = hq.clinician[setId];
  if (validated && row) return `${row.validatedAt ? "Validated by" : "Saved by"} ${row.updatedByName} on ${formatDateTime(row.validatedAt ?? row.updatedAt)}`;
  const who = hq.enteredBy[setId] ? `Patient (entered by ${hq.enteredBy[setId]})` : "Patient";
  return e.patientSubmittedAt ? `${who} on ${formatDateTime(e.patientSubmittedAt)}` : `${who}, not sent yet`;
}

function SetSection({ set, byLine, validated }: { set: PoaSet; byLine: string; validated: boolean }) {
  return (
    <Stack gap="lg">
      {set.pages.map((p) => (
        <div key={p.name} data-testid={`poa-page-${p.title}`}>
          {(p.notes.length > 0 || p.comments) && (
            <Stack gap={4} mb={p.captures.length ? "sm" : 0}>
              <Text fw={700} c="blue.9">{p.title} Summary</Text>
              <Text size="xs" c="dimmed">{byLine}</Text>
              {p.notes.map((n) => (
                <Group key={n.text} gap={6}>
                  <Text size="sm" c="blue.8" td="underline">{n.text}</Text>
                  {validated && n.corrected && (
                    <Tooltip label={changeText(p.changes.find((c) => c.questionId === n.questionId))}>
                      <Badge size="xs" color="orange" variant="light">Changed by clinician</Badge>
                    </Tooltip>
                  )}
                </Group>
              ))}
              {p.comments && (
                <Text size="sm"><b>Clinical comments:</b> {p.comments}</Text>
              )}
            </Stack>
          )}
          {p.captures.map((c) => (
            <Stack key={c.questionId} gap={4} mb="sm">
              <Text fw={700} c="blue.9">{c.title}</Text>
              <Text size="xs" c="dimmed">{byLine}</Text>
              <Capture capture={c} />
            </Stack>
          ))}
          {validated && p.changes.length > 0 && (
            <Alert color="orange" variant="light" p="xs" mt={4} title={`Changed by the clinician on ${p.title}`}>
              <List size="sm">
                {p.changes.map((c) => (
                  <List.Item key={c.questionId}>{c.title} {changeText(c)}</List.Item>
                ))}
              </List>
            </Alert>
          )}
        </div>
      ))}
    </Stack>
  );
}

const changeText = (c?: { patient: string; clinician: string }) => (c ? `Patient answered “${c.patient}”, now “${c.clinician}”` : "");

const COLUMNS: Record<Exclude<PoaCapture["kind"], "bmi">, [string, string][]> = {
  medication: [["name", "Name"], ["dosage", "Dosage"], ["frequency", "Frequency"]],
  admissions: [["hospital", "Hospital Name"], ["anaesthetic", "Anaesthetic Type"], ["reason", "Reason for Admission"], ["year", "Year"]],
};

function Capture({ capture }: { capture: PoaCapture }) {
  if (capture.kind === "bmi") {
    const r = capture.rows[0] ?? {};
    return (
      <Stack gap={2}>
        <Text size="sm">Height: {String(r.height ?? "—")} cm</Text>
        <Text size="sm">Weight: {String(r.weight ?? "—")} kg</Text>
        <Text size="sm">BMI: {String(r.bmi ?? "—")}</Text>
      </Stack>
    );
  }
  const cols = COLUMNS[capture.kind];
  return (
    <Table withTableBorder verticalSpacing={6} fz="sm">
      <Table.Thead>
        <Table.Tr>{cols.map(([k, label]) => <Table.Th key={k}>{label}</Table.Th>)}</Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        {capture.rows.map((r, i) => (
          <Table.Tr key={i}>{cols.map(([k]) => <Table.Td key={k}>{String(r[k] ?? "")}</Table.Td>)}</Table.Tr>
        ))}
      </Table.Tbody>
    </Table>
  );
}

const ASA = ["I", "II", "III", "IV", "V", "VI"];
const asaLabel = (n: number | null) => (n ? `ASA ${ASA[n - 1]}` : "Not recorded");

/**
 * The two ASA grades, each saved on its own with the revision the page loaded (S20). If the other
 * was saved meanwhile (another tab), the save is refused: the grade typed stays, with Reload.
 * After a reload, a grade nobody is changing here shows the saved one.
 */
function AsaGrades({ episode, path, onSaved }: { episode: Episode; path: string; onSaved: () => void }) {
  const saved = { nurseAsa: episode.nurseAsa ? String(episode.nurseAsa) : null, anaesthetistAsa: episode.anaesthetistAsa ? String(episode.anaesthetistAsa) : null };
  // A grade being changed here (`value`), or null: it shows the saved one.
  const [editing, setEditing] = useState<{ nurseAsa: string | null | undefined; anaesthetistAsa: string | null | undefined }>({
    nurseAsa: undefined, anaesthetistAsa: undefined,
  });
  const shown = (f: "nurseAsa" | "anaesthetistAsa") => (editing[f] === undefined ? saved[f] : editing[f]);
  const [error, setError] = useState("");
  const [stale, setStale] = useState(false);
  const save = async (field: "nurseAsa" | "anaesthetistAsa") => {
    const value = shown(field);
    try {
      await api(path, { method: "PATCH", body: JSON.stringify({ [field]: value ? Number(value) : 0, revision: episode.revision }) });
      setError("");
      setEditing((was) => ({ ...was, [field]: undefined }));
      onSaved();
    } catch (err) {
      setError((err as Error).message);
      setStale(isStale(err));
    }
  };
  const data = ASA.map((g, i) => ({ value: String(i + 1), label: `ASA ${g}` }));
  const set = (f: "nurseAsa" | "anaesthetistAsa") => (v: string | null) => setEditing((was) => ({ ...was, [f]: v }));
  return (
    <Stack>
      {!episode.nurseAsa && !episode.anaesthetistAsa && <Text size="sm">No ASA grades have been added.</Text>}
      <SaveError error={error} stale={stale} onReload={() => { setError(""); onSaved(); }} />
      <Group align="flex-end">
        <Select label="Nurse ASA grade" placeholder="ASA grade" data={data} value={shown("nurseAsa")} onChange={set("nurseAsa")} clearable w={180} />
        <Button className="sj-no-print" variant="default" onClick={() => save("nurseAsa")} disabled={shown("nurseAsa") === saved.nurseAsa}>Save</Button>
        <Select label="Anaesthetist ASA grade" placeholder="ASA grade" data={data} value={shown("anaesthetistAsa")} onChange={set("anaesthetistAsa")} clearable w={200} ml="lg" />
        <Button className="sj-no-print" variant="default" onClick={() => save("anaesthetistAsa")} disabled={shown("anaesthetistAsa") === saved.anaesthetistAsa}>Save</Button>
      </Group>
    </Stack>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <Group gap="xs" wrap="nowrap" align="baseline">
      <Text size="sm" fw={600} w={190}>{label}</Text>
      <Text size="sm">{value || "—"}</Text>
    </Group>
  );
}
