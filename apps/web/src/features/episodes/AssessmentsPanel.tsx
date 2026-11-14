// The episode page's Assessments card (plan C7, session S47; §9): Add assessment (STOP-Bang), then
// each assessment's items, pre-filled from the HQ, the observations and the profile, each with its
// source; a missing item is highlighted. The clinician answers what's missing or changes any item
// (the profile's age and sex come from the patient's record), then Saves or Confirms. Confirm
// records who and when; a later Save clears it, so a changed item needs a new Confirm. A cancelled
// or archived episode's assessments are shown, not changed. With assessments off it isn't shown.
import { useMemo, useState } from "react";
import { Alert, Badge, Button, Card, Group, Select, Stack, Table, Text, TextInput, Title } from "@mantine/core";
import { assessmentAnswers, assessmentItems, assessmentResult, draftAfter, shownWithout, withDraft, type Answers, type AssessmentItem, type Reading } from "@poised/clinical";
import { api } from "@/lib/api";
import { isStale, SaveError } from "./StatusControl";
import {
  boxPlaceholder, boxTyped, numberInput, cardHints, cardNotes, changedItems, confirmText, filledText, withoutInvalid, scoreDay, prefillFrom, saveRequest, shownItems, sourceText, valueText, type AssessmentRow, type AssessmentsView, sinceConfirmed,
} from "./assessments";
import type { EpisodeHQ } from "./hq";
import type { Episode } from "./types";

type Props = {
  path: string;
  episode: Episode;
  assessments: { view: AssessmentsView | null | "off"; error: string; load: () => void };
  hq: EpisodeHQ | null;
  /** The episode's readings, for the hospital BMI; undefined with observations off or not loaded. */
  readings: Reading[] | undefined;
  /** Whether the HQ and the observations are in (inputsState): until then, no Save or Confirm. */
  inputs: { ready: boolean; problem?: string };
};

export function AssessmentsPanel({ path, episode, assessments, hq, readings, inputs }: Props) {
  const { view, error, load } = assessments;
  // The add's refusal (e.g. another tab added it meanwhile) is kept here, not in the menu, which
  // goes once nothing is left to add.
  const [addError, setAddError] = useState("");
  const readOnly = episode.status === "cancelled" || episode.status === "archived";
  if (view === "off") return null;
  return (
    <Card withBorder data-testid="assessments">
      <Group justify="space-between">
        <Title order={4}>Assessments</Title>
        {view && !readOnly && view.available.length > 0 && <AddAssessment path={path} view={view} onAdded={load} onError={setAddError} />}
      </Group>
      {error && <Alert color="red" mt="xs">{error}</Alert>}
      {addError && <Alert color="red" mt="xs" withCloseButton onClose={() => setAddError("")}>{addError}</Alert>}
      {view && (
        <Stack gap="md" mt="xs">
          {view.assessments.length === 0 && <Text size="sm" c="dimmed">No assessments yet.</Text>}
          {view.assessments.map((a) => (
            // A new key after each save, so the screen's changes start again from what was saved.
            <Assessment key={`${a.id}:${a.revision}`} path={path} episode={episode} row={a} view={view} hq={hq} readings={readings}
              readOnly={readOnly} inputs={inputs} onSaved={load} />
          ))}
        </Stack>
      )}
    </Card>
  );
}

function AddAssessment({ path, view, onAdded, onError }: {
  path: string; view: AssessmentsView; onAdded: () => void; onError: (message: string) => void;
}) {
  const [saving, setSaving] = useState(false);
  const add = async (id: string | null) => {
    if (!id) return;
    setSaving(true);
    try {
      await api(`${path}/assessments`, { method: "POST", body: JSON.stringify({ assessment: id }) });
      onError("");
      onAdded();
    } catch (e) {
      onError((e as Error).message);
      onAdded(); // e.g. someone else added it meanwhile
    } finally {
      setSaving(false);
    }
  };
  return (
    <Select placeholder="Add assessment" aria-label="Add assessment" data={view.available.map((a) => ({ value: a.id, label: a.name }))}
      value={null} onChange={add} disabled={saving} w={200} allowDeselect={false} />
  );
}

function Assessment({ path, episode, row, view, hq, readings, readOnly, inputs, onSaved }: {
  path: string; episode: Episode; row: AssessmentRow; view: AssessmentsView; hq: EpisodeHQ | null; readings: Reading[] | undefined;
  readOnly: boolean; inputs: Props["inputs"]; onSaved: () => void;
}) {
  const [draft, setDraft] = useState<Answers>({});
  // Number boxes holding something that isn't a value (a mistake, or emptied): Save and Confirm wait.
  const [invalid, setInvalid] = useState<Record<string, boolean>>({});
  const [resets, setResets] = useState(0); // Undo changes starts the boxes again
  const [error, setError] = useState("");
  const [stale, setStale] = useState(false);
  const [saving, setSaving] = useState(false);
  // What's shown before any change: a confirmed assessment as confirmed, with the age that day.
  const shownFrom = useMemo(
    () => prefillFrom(hq, episode, readings, view.obsPrecedence, row.confirmedAt ?? undefined),
    [hq, episode, readings, view.obsPrecedence, row.confirmedAt],
  );
  const base = useMemo(() => shownItems(row, shownFrom, {}), [row, shownFrom]);
  const drafted = useMemo(() => withDraft(row.content, base, draft), [row.content, base, draft]);
  const bad = Object.values(invalid).some(Boolean);
  // A change back to what was shown isn't a change.
  const changed = bad || drafted.some((i, n) => i.value !== base[n].value || i.source !== base[n].source);
  // Once changed, the age is today's: the day a Confirm records, so the score shown is the one confirmed.
  const day = scoreDay(row, changed);
  const from = useMemo(
    () => ((day ?? null) === row.confirmedAt ? shownFrom : prefillFrom(hq, episode, readings, view.obsPrecedence, day)),
    [day, row.confirmedAt, shownFrom, hq, episode, readings, view.obsPrecedence],
  );
  const items = useMemo(
    // Once changed, the age and sex are the record's today: what a Confirm then stores (F-1).
    () => withoutInvalid(changed ? changedItems(row, from, draft) : drafted, invalid),
    [changed, from, drafted, row, draft, invalid],
  );
  const result = useMemo(() => assessmentResult(row.content, assessmentAnswers(items).data, from.patient), [row.content, items, from.patient]);
  // Complete once there's a result and no item is missing (VTE's bleeding items aren't its first result's, verification 2).
  const complete = result.score !== undefined && !items.some((i) => !i.source);
  const missing = items.some((i) => !i.source);
  // S47 answers 6: what it used that has changed since Confirm, against a fresh pre-fill (the record
  // as of the day of Confirm). Shown only while it's shown as confirmed.
  const since = useMemo(
    () => (row.confirmedAt && !changed ? sinceConfirmed(base, assessmentItems(row.content, shownFrom)) : []),
    [row.confirmedAt, row.content, changed, base, shownFrom],
  );
  const undo = () => { setDraft({}); setInvalid({}); setResets((n) => n + 1); };
  const hints = useMemo(() => cardHints(row, view, episode), [row, view, episode]);

  const send = async (confirm: boolean) => {
    setSaving(true);
    try {
      await api(`${path}/assessments/${row.id}${confirm ? "/confirm" : ""}`, {
        method: confirm ? "POST" : "PUT", body: JSON.stringify(saveRequest(row, items)),
      });
      setError("");
      onSaved();
    } catch (e) {
      setError((e as Error).message);
      setStale(isStale(e) || (e as { status?: number }).status === 409);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card withBorder padding="sm" data-testid={`assessment-${row.assessment}`}>
      <Group justify="space-between" mb={4}>
        <Group gap="xs">
          <Text fw={700}>{row.name}</Text>
          <Badge variant="light" color={complete ? "blue" : "gray"} data-testid="assessment-result">{result.text}</Badge>
        </Group>
        <Text size="sm" c={row.confirmedAt && !changed ? "green.8" : "dimmed"} data-testid="assessment-confirmed">
          {row.confirmedAt && changed ? "Changed: confirm it again" : confirmText(row)}
        </Text>
      </Group>
      <Text size="sm" c={missing ? "orange.8" : "dimmed"} mb="xs" data-testid="assessment-filled">
        {filledText(items, readOnly)}
      </Text>
      {hints.length > 0 && (
        <Alert color="blue" variant="light" py={6} mb="xs" data-testid="assessment-hints">
          <Text size="xs" fw={600}>For reference</Text>
          {hints.map((h) => <Text key={h} size="sm">{h}</Text>)}
        </Alert>
      )}
      <Table verticalSpacing={4}>
        <Table.Tbody>
          {items.map((i) => (
            <Table.Tr key={i.key} data-testid={`item-${i.letter}`} data-missing={i.source ? "false" : "true"} bg={i.source ? undefined : "orange.0"}>
              <Table.Td w={30}><Text fw={700}>{i.letter}</Text></Table.Td>
              <Table.Td>
                <Text size="sm" fw={600}>{i.label}</Text>
                {i.title && i.title !== i.label && <Text size="xs" c="dimmed">{i.title}</Text>}
              </Table.Td>
              {/* A menu needs room for its longest choice ("8: Living with Very Severe Frailty", report 1). */}
              <Table.Td w={(i.options ?? []).length > 2 && !readOnly ? 270 : 200}>
                <ItemValue key={`${i.key}:${resets}`} item={i} readOnly={readOnly} had={!!(i.measure ? shownWithout(row.content, base, draft, i.key) : base.find((b) => b.key === i.key))?.source} invalid={!!invalid[i.key]}
                  onChange={(v) => setDraft((d) => draftAfter(row.content, base, d, i.key, v))}
                  onInvalid={(no) => setInvalid((x) => ({ ...x, [i.key]: no }))} />
              </Table.Td>
              {/* Wide enough for the longest source, so the answers don't move as a source changes. */}
              <Table.Td w={170} ta="right" style={{ whiteSpace: "nowrap" }}>
                <Badge variant={i.source ? "outline" : "filled"} color={i.source ? "gray" : "orange"} size="sm" tt="none" data-testid="item-source"
                  styles={{ root: { maxWidth: "none" }, label: { overflow: "visible" } }}>
                  {sourceText(i)}
                </Badge>
              </Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
      {since.length > 0 && (
        <Alert color="yellow" py={6} mt="xs" data-testid="assessment-since">
          {since.map((l) => <Text key={l} size="sm">{l}</Text>)}
          <Text size="xs" c="dimmed" mt={2}>
            The score stays as confirmed. To use a new value, change the item and confirm again.
            {since.some((l) => l.startsWith("The patient's record")) && " Once any item is changed, the age and sex are the record's."}
          </Text>
        </Alert>
      )}

      {!readOnly && (
        <Stack gap="xs" mt="xs">
          <SaveError error={error} stale={stale} onReload={() => { setError(""); setStale(false); onSaved(); }} />
          {inputs.problem && <Alert color="red" py={6} data-testid="assessment-inputs">{inputs.problem}</Alert>}
          <Group gap="xs">
            <Button onClick={() => send(true)} loading={saving}
              disabled={!inputs.ready || !complete || bad || (!!row.confirmedAt && !changed)}>Confirm</Button>
            <Button variant="default" onClick={() => send(false)} loading={saving} disabled={!inputs.ready || !changed || bad}>Save</Button>
            {changed && <Button variant="subtle" onClick={undo}>Undo changes</Button>}
          </Group>
          {!complete && <Text size="xs" c="dimmed">Confirm once every item has an answer.</Text>}
        </Stack>
      )}
      {cardNotes(items, day).map((n) => <Text key={n} size="xs" c="dimmed" mt={4} data-testid="assessment-note">{n}</Text>)}
    </Card>
  );
}

/** Yes and No buttons (a menu for more than two options), a number box, or the profile's value (not changed here). */
/**
 * `had`: the box showed a value before the clinician typed in it (saved or pre-filled; not one the
 * MUST pairing rule emptied, S49 verification 5). Only a BMI or weight depends on the other boxes,
 * so only those rows work it out from the draft (verification 6, finding 2). Emptying its box
 * then needs a value ("Enter a BMI"); emptying a box that started empty leaves the item missing.
 */
function ItemValue({ item: i, readOnly, had, invalid, onChange, onInvalid }: {
  item: AssessmentItem; readOnly: boolean; had: boolean; invalid: boolean; onChange: (v: unknown) => void; onInvalid: (invalid: boolean) => void;
}) {
  // What the clinician typed, while they're typing; otherwise the item's value, which follows a
  // reading recorded meanwhile.
  const [typing, setTyped] = useState<string | null>(null);
  const typed = boxTyped(typing, i);
  const text = typed ?? (i.source ? String(i.value) : "");
  if (i.kind === "profile" || readOnly) return <Text size="sm">{valueText(i)}</Text>;
  if (i.kind === "number") {
    const typedNow = typed === null ? {} : numberInput(typed, i);
    const what = boxPlaceholder(i);
    return (
      <TextInput size="xs" aria-label={`${i.letter}: ${i.label}`} value={text} placeholder={what}
        onChange={(e) => {
          const t = e.currentTarget.value;
          setTyped(t);
          const { value } = numberInput(t, i);
          if (value !== undefined) onChange(value);
          else if (!t.trim() && !had) onChange(undefined); // back to missing, as it started
          // Emptying a box that showed a value isn't an answer either: Undo changes brings it back.
          onInvalid(value === undefined && (!!t.trim() || had));
        }}
        error={typedNow.error ?? (invalid && typed !== null && !typed.trim() ? (what === "BMI" ? "Enter a BMI" : "Enter a number") : undefined)} />
    );
  }
  if ((i.options ?? []).length > 2) {
    return (
      <Select size="xs" aria-label={`${i.letter}: ${i.label}`} placeholder="Choose" value={i.source ? String(i.value) : null}
        data={(i.options ?? []).map((o) => ({ value: o.value, label: o.text }))} allowDeselect={false}
        onChange={(v) => v && onChange(v)} comboboxProps={{ width: 300, position: "bottom-end" }} />
    );
  }
  return (
    <Button.Group>
      {(i.options ?? []).map((o) => (
        <Button key={o.value} size="xs" variant={i.value === o.value ? "filled" : "default"} aria-label={`${i.letter}: ${o.text}`}
          aria-pressed={i.value === o.value} onClick={() => onChange(o.value)}>
          {o.text}
        </Button>
      ))}
    </Button.Group>
  );
}
