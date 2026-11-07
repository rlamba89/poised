// Step 5 of plan-workflow.md: the POA Summary, laid out like the incumbent's (layout, not styling).
// "Validated summary" is built from the clinician's answers, which are final, and marks what the
// clinician changed; "Patient answers" is the same document from the patient's original answers.
// Both list each page's pre-filled answers shown (S26, PRE-04): where each came from, and "Changed by
// patient" where the patient's answer differs from the pre-filled one; and the button the patient
// pressed on the set's opening card.
// Every answer and disclosure on either tab is a link (S27): it opens Validate for its Question Set
// at that question, and Back returns here, to the same tab and the same place (jump.ts).
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigationType, useSearchParams } from "react-router";
import {
  Accordion, Alert, Anchor, Badge, Breadcrumbs, Button, Card, Center, Group, List, Loader, Select, SimpleGrid, Stack, Table, Tabs, Text,
  Title, Tooltip,
} from "@mantine/core";
import { IconPrinter } from "@tabler/icons-react";
import {
  ageFrom, hqSuggestedAsa, poaSummary, prefillChoiceText, prefillSourceText, reportedBmi, startedFreshText, type PoaCapture, type PoaSet,
} from "@poised/clinical";
import { api } from "@/lib/api";
import { nationalIdLabel } from "@/lib/config";
import { formatDate, formatDateTime, formatDay, formatOptionalDate } from "@/lib/format";
import { StatusBadge } from "./EpisodeList";
import { SuggestedAsaBlock } from "./SuggestedAsa";
import { useEpisode } from "./EpisodePage";
import { GeneralNotes } from "./GeneralNotes";
import { useObservations } from "./ObservationsPanel";
import { hospitalBmiRow } from "./observations";
import { poaLines, useAssessments } from "./assessments";
import { allCurrentAnswers, hqPatient, useEpisodeHQ, type EpisodeHQ } from "./hq";
import { restoreSpot, saveSpot, takeSpot, validateAt } from "./jump";
import { sessionStore } from "@/lib/chunkError";
import { SaveError, isStale, StatusControl } from "./StatusControl";
import { asaRequirement, triageStamp } from "./status";
import { AnaesthetistGrades } from "./AnaesthetistGrades";
import { asaLabel, type AnaesthetistGrade } from "./asaGrades";
import { patientName, type Episode, type EpisodeEvent } from "./types";

type Tab = "validated" | "patient";

export default function PoaSummary({ base, episodeId }: { base: string; episodeId: string }) {
  const { data, error, reload } = useEpisode(base, episodeId);
  const { hq, error: hqError } = useEpisodeHQ(base, episodeId);
  const path = `${base}/episodes/${episodeId}`;
  // The hospital BMI (S46, OBS-04): read once here, not per tab, as every read is an audit row.
  const obs = useObservations(path, data?.observationsOn);
  // Each assessment's score and band as confirmed (S47), beside the hospital BMI; read once here too.
  const assessments = useAssessments(path, data?.assessmentsOn);
  // Worked out once per load, not on every render: one survey per confirmed assessment (S47 verification 2).
  const assessmentLines = useMemo(
    () => (data ? poaLines(assessments.view, data.episode, assessments.error) : []),
    [assessments.view, assessments.error, data],
  );
  // The tab is in the address (?tab=patient), so Back from Validate and a reload keep it (S27).
  const [params, setParams] = useSearchParams();
  const tab: Tab = params.get("tab") === "patient" ? "patient" : "validated";
  const setTab = (t: Tab) => setParams(t === "patient" ? { tab: t } : {}, { replace: true });
  // Back from Validate: the answer clicked goes back where it was on the screen, once everything is drawn.
  const navigation = useNavigationType();
  const loaded = !!data && !!hq;
  useEffect(() => {
    if (!loaded) return;
    const spot = takeSpot(sessionStore(), episodeId);
    if (spot && navigation === "POP") return restoreSpot(spot); // stops trying once the page is left
  }, [loaded, navigation, episodeId]);
  if (error || hqError) return <Alert color="red">{error || hqError}</Alert>;
  if (!data || !hq) return <Center h="40vh"><Loader /></Center>;
  const e = data.episode;

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
          <Summary key={tab} tab={tab} episode={e} events={data.events} grades={data.anaesthetistAsaGrades} hq={hq} path={path} onChanged={reload}
            episodeId={episodeId} hospitalBmi={hospitalBmiRow(obs.view, obs.error)} assessments={assessmentLines} />
        </Stack>
      </Card>
    </Stack>
  );
}

type SummaryProps = {
  tab: Tab; episode: Episode; events: EpisodeEvent[]; grades: AnaesthetistGrade[]; hq: EpisodeHQ; path: string; onChanged: () => void; episodeId: string;
  hospitalBmi: string | null;
  /** S47: each assessment's "5 (High risk)", or "Not confirmed". */
  assessments: { name: string; text: string }[];
};

function Summary({ tab, episode: e, events, grades, hq, path, onChanged, episodeId, hospitalBmi, assessments }: SummaryProps) {
  const vars = useMemo(() => hqPatient(e), [e]);
  const validated = tab === "validated";
  const sets = useMemo(() => {
    if (validated) {
      return poaSummary(hq.chapters, allCurrentAnswers(hq), "clinician", vars, hq.patient, hq.prefill);
    }
    return poaSummary(hq.chapters.filter((c) => c.audience === "patient"), hq.patient, "patient", vars, undefined, hq.prefill);
  }, [validated, hq, vars]);
  // The suggested ASA (S39): from the answers that count on the validated tab, from the patient's on theirs.
  const asa = useMemo(
    () => (validated
      ? hqSuggestedAsa(hq.chapters, allCurrentAnswers(hq), "clinician", vars)
      : hqSuggestedAsa(hq.chapters.filter((c) => c.audience === "patient"), hq.patient, "patient", vars)),
    [validated, hq, vars],
  );
  // The patient's own height and weight until their set is validated, then the validated ones (S28), on both tabs;
  // only answers that are shown (S29).
  const bmi = useMemo(
    () => reportedBmi(hq.chapters, (id) => (hq.clinician[id]?.validatedAt ? hq.clinician[id].data : hq.patient[id]), vars),
    [hq, vars],
  );

  return (
    <Stack>
      {validated && !e.reviewCompletedAt && (triageStamp(e) ? (
        // S42: the HQ's rules moved it on, so there's no validated copy: the patient's answers are shown.
        <Alert color="teal" data-testid="poa-auto-triaged">
          {triageStamp(e)} by the rule “{e.triageRule}”. No clinician validated these answers, so this shows the patient&apos;s own.
        </Alert>
      ) : (
        <Alert color="orange" className="sj-no-print">The HQ review isn&apos;t complete yet. This shows the answers as they stand.</Alert>
      ))}
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
          {/* Beside the patient's own; not shown with observations off. */}
          {hospitalBmi !== null && <Detail label="Hospital BMI" value={hospitalBmi} />}
          {assessments.map((a) => <Detail key={a.name} label={a.name} value={a.text} />)}
          <Detail label="Anaesthetist ASA grade" value={asaLabel(e.anaesthetistAsa)} />
          <Detail label="Nurse ASA grade" value={asaLabel(e.nurseAsa)} />
        </Stack>
      </SimpleGrid>

      <Accordion multiple defaultValue={["notes", ...sets.map((s) => s.id), "asa"]} chevronPosition="right" variant="default">
        <Accordion.Item value="notes">
          <Accordion.Control><Title order={4} c="blue.9">General notes</Title></Accordion.Control>
          <Accordion.Panel>
            <GeneralNotes path={path} events={events} onAdded={onChanged} readOnly={e.status === "archived"} />
          </Accordion.Panel>
        </Accordion.Item>

        {sets.map((s) => (
          <Accordion.Item key={s.id} value={s.id} data-testid={`poa-set-${s.name}`}>
            <Accordion.Control><Title order={4} c="blue.9">{s.name}</Title></Accordion.Control>
            <Accordion.Panel>
              <SetSection set={s} byLine={byLine(validated, e, hq, s.id)} validated={validated} answerLink={answerLink(path, episodeId, s.id)} />
            </Accordion.Panel>
          </Accordion.Item>
        ))}

        <Accordion.Item value="asa">
          <Accordion.Control><Title order={4} c="blue.9">Pre operative assessment ASA grades</Title></Accordion.Control>
          <Accordion.Panel>
            <Stack>
              {/* Each reason is a link to its question, as the other answers are (S27). */}
              <SuggestedAsaBlock
                asa={asa}
                link={(r) => (r.setId ? answerLink(path, episodeId, r.setId)(r.questionId, `asa-${r.text}`, r.text) : r.text)}
              />
              {validated && (
                <AsaGrades episode={e} grades={grades} path={path} onSaved={onChanged} readOnly={e.status === "archived" || e.status === "cancelled"} />
              )}
            </Stack>
          </Accordion.Panel>
        </Accordion.Item>
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
  const line = e.patientSubmittedAt ? `${who} on ${formatDateTime(e.patientSubmittedAt)}` : `${who}, not sent yet`;
  const stamp = triageStamp(e); // S42: every set of an auto-triaged episode says so
  return stamp && !row?.validatedAt ? `${stamp} · ${line}` : line;
}

/**
 * An answer or disclosure as a link to Validate at its question (S27). `kind` tells apart the
 * links to one question (its note, its pre-fill line, its change), for the way back.
 */
type AnswerLink = (questionId: string, kind: string, children: React.ReactNode) => React.ReactNode;

const answerLink = (path: string, episodeId: string, setId: string): AnswerLink =>
  function AnswerLinkTo(questionId, kind, children) {
    const anchor = `${setId}:${kind}:${questionId}`;
    return (
      <Anchor
        component={Link}
        to={validateAt(path, setId, questionId)}
        state={{ from: "poa" }}
        inherit
        data-poa-anchor={anchor}
        data-testid="poa-answer-link"
        onClick={(ev: React.MouseEvent<HTMLAnchorElement>) =>
          saveSpot(sessionStore(), episodeId, { anchor, top: ev.currentTarget.getBoundingClientRect().top })
        }
      >
        {children}
      </Anchor>
    );
  };

function SetSection({ set, byLine, validated, answerLink }: { set: PoaSet; byLine: string; validated: boolean; answerLink: AnswerLink }) {
  const choice = set.prefill && set.pages.some((p) => p.prefilled.length > 0) ? prefillChoiceText(set.prefill, formatDay) : "";
  // A set started fresh shows no pre-filled answer, so it says so, as Validate does (S28).
  const fresh = startedFreshText(set.prefill, formatDay);
  return (
    <Stack gap="lg">
      {choice && <Text size="sm" c="dimmed" data-testid="poa-prefill-choice">{choice}</Text>}
      {fresh && <Text size="sm" c="dimmed" data-testid="poa-started-fresh">{fresh}</Text>}
      {set.pages.map((p) => (
        <div key={p.name} data-testid={`poa-page-${p.title}`}>
          {/* One block per Clinical Summary, with its own title, notes and comments (S32). */}
          {p.blocks.map((b) => (
            <Stack key={b.name} gap={4} mb="sm" data-testid={`poa-summary-${b.title}`}>
              <Text fw={700} c="blue.9">{b.title}</Text>
              <Text size="xs" c="dimmed">{byLine}</Text>
              {b.notes.map((n) => (
                <Group key={n.text} gap={6}>
                  <Text size="sm" c="blue.8" td="underline">{answerLink(n.questionId, `note-${n.text}`, n.text)}</Text>
                  {validated && n.corrected && (
                    <Tooltip label={changeText(p.changes.find((c) => c.questionId === n.questionId))}>
                      <Badge size="xs" color="orange" variant="light">Changed by clinician</Badge>
                    </Tooltip>
                  )}
                </Group>
              ))}
              {b.comments && (
                <Text size="sm"><b>Clinical comments:</b> {b.comments}</Text>
              )}
            </Stack>
          ))}
          {p.captures.map((c) => (
            <Stack key={c.questionId} gap={4} mb="sm">
              <Text fw={700} c="blue.9">{answerLink(c.questionId, "capture", c.title)}</Text>
              <Text size="xs" c="dimmed">{byLine}</Text>
              <Capture capture={c} />
            </Stack>
          ))}
          {p.prefilled.length > 0 && (
            <Alert color="brand" variant="light" p="xs" mt={4} title={`Pre-filled on ${p.title}`} data-testid="poa-prefilled">
              <List size="sm">
                {p.prefilled.map((m) => (
                  <List.Item key={m.questionId}>
                    {answerLink(m.questionId, "prefill", m.title)} — {prefillSourceText(m.entry, formatDay)}
                    {m.changed && (
                      <>
                        {" "}<Badge size="xs" color="orange" variant="light">Changed by patient</Badge> pre-filled “{m.was}”, patient answered “{m.now}”
                      </>
                    )}
                  </List.Item>
                ))}
              </List>
            </Alert>
          )}
          {validated && p.changes.length > 0 && (
            <Alert color="orange" variant="light" p="xs" mt={4} title={`Changed by the clinician on ${p.title}`}>
              <List size="sm">
                {p.changes.map((c) => (
                  <List.Item key={c.questionId}>{answerLink(c.questionId, "change", <>{c.title} {changeText(c)}</>)}</List.Item>
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

/**
 * The two ASA grades. The nurse's is saved on its own with the revision the page loaded (S20). If the
 * episode changed meanwhile (another tab), the save is refused: the grade typed stays, with Reload.
 * The anaesthetist's is a history (S44b): added, never changed or cleared (AnaesthetistGrades).
 */
function AsaGrades({ episode, grades, path, onSaved, readOnly }: {
  episode: Episode; grades: AnaesthetistGrade[]; path: string; onSaved: () => void; readOnly: boolean;
}) {
  const requirement = asaRequirement(episode);
  const saved = episode.nurseAsa ? String(episode.nurseAsa) : null;
  // The grade being changed here, or undefined: it shows the saved one.
  const [editing, setEditing] = useState<string | null | undefined>(undefined);
  const shown = editing === undefined ? saved : editing;
  const [error, setError] = useState("");
  const [stale, setStale] = useState(false);
  const save = async () => {
    try {
      await api(path, { method: "PATCH", body: JSON.stringify({ nurseAsa: shown ? Number(shown) : 0, revision: episode.revision }) });
      setError("");
      setEditing(undefined);
      onSaved();
    } catch (err) {
      setError((err as Error).message);
      setStale(isStale(err));
    }
  };
  const data = [1, 2, 3, 4, 5, 6].map((n) => ({ value: String(n), label: asaLabel(n) }));
  return (
    <Stack>
      {/* Not above a history whose grades were all entered in error: its own line says "No grade yet" (test report 1, note 2). */}
      {!episode.nurseAsa && grades.length === 0 && <Text size="sm">No ASA grades have been added.</Text>}
      {/* S43: a rule asked for the anaesthetist's grade before Ready for admission. */}
      {requirement && <Text size="sm" c={requirement.met ? "dimmed" : "red.8"} data-testid="poa-asa-required">{requirement.text}</Text>}
      <SaveError error={error} stale={stale} onReload={() => { setError(""); onSaved(); }} />
      <Group align="flex-end">
        {/* A cancelled (S20) or archived (S22) episode's grades are shown, not changed: the API refuses them. */}
        <Select label="Nurse ASA grade" placeholder="ASA grade" data={data} value={shown} onChange={setEditing} clearable w={180} disabled={readOnly} />
        <Button className="sj-no-print" variant="default" onClick={save} disabled={readOnly || shown === saved}>Save</Button>
      </Group>
      <AnaesthetistGrades path={path} revision={episode.revision} grades={grades} onChanged={onSaved} readOnly={readOnly} />
    </Stack>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <Group gap="xs" wrap="nowrap" align="baseline" data-testid={`poa-detail-${label}`}>
      <Text size="sm" fw={600} w={190}>{label}</Text>
      <Text size="sm">{value || "—"}</Text>
    </Group>
  );
}
