// The episode page's Observations card (plan C7, session S46; OBS-01–04): the hospital BMI, Add a
// reading (now by default, or an earlier time), and the history newest first. A reading outside
// the usual range asks "Save anyway?" before it's sent; outside the hard limits Go refuses it with
// its reason. Each reading still counting has Correct (a new value, with a reason; the old one stays,
// struck through) and Entered in error (a reason). Nothing edits or deletes a reading: the API has
// no way to. A cancelled or archived episode's readings are shown, not changed. With observations
// off at the hospital (403) the card isn't shown.
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { Alert, Badge, Button, Card, Group, Select, Stack, Text, Textarea, TextInput, Title } from "@mantine/core";
import { counting, type Reading } from "@poised/clinical";
import { api, ApiError } from "@/lib/api";
import { toLocalInput } from "@/lib/localTime";
import type { Episode } from "./types";
import {
  correctionRequest, errorLine, historyLine, hospitalBmiText, kindOf, measuredAt, newestFirst, parseNumber, readingRequest, unitOf, valueText, warningText,
  type ObservationsView,
} from "./observations";

/**
 * The readings of one episode, loaded again after each change; null while loading, "off" with
 * observations off. `on` is the episode's observationsOn (undefined until it's loaded): off, the
 * route isn't called, as a refused call would write an audit row on each page open.
 */
export function useObservations(path: string, on: boolean | undefined) {
  const [view, setView] = useState<ObservationsView | null | "off">(null);
  const [error, setError] = useState("");
  const load = useCallback(
    () =>
      api<ObservationsView>(`${path}/observations`)
        .then((v) => {
          setView(v);
          setError("");
        })
        .catch((e: Error) => {
          if (e instanceof ApiError && e.status === 403) setView("off");
          else setError(e.message);
        }),
    [path],
  );
  useEffect(() => {
    if (on) load();
  }, [load, on]);
  return { view: on === false ? ("off" as const) : view, error, load };
}

/** "Hospital BMI: 28.5 (measured …)", or why there's none yet. `obs` is the page's useObservations. */
export function ObservationsPanel({ path, episode, obs }: { path: string; episode: Episode; obs: ReturnType<typeof useObservations> }) {
  const { view, error, load } = obs;
  const readOnly = episode.status === "cancelled" || episode.status === "archived";
  if (view === "off") return null;
  return (
    <Card withBorder data-testid="observations">
      <Title order={4}>Observations</Title>
      {error && <Alert color="red" mt="xs">{error}</Alert>}
      {view && (
        <Stack gap="sm" mt="xs">
          <Text size="sm">
            <Text span fw={600}>Hospital BMI: </Text>
            <span data-testid="hospital-bmi">{hospitalBmiText(view.readings)}</span>
          </Text>
          {!readOnly && <AddReading path={path} view={view} onSaved={load} />}
          {view.readings.length === 0 ? (
            <Text size="sm" c="dimmed">No readings yet.</Text>
          ) : (
            <Stack gap={6} data-testid="observation-history">
              {newestFirst(view.readings).map((r) => (
                <ReadingEntry key={r.id} r={r} view={view} path={path} onChanged={load} readOnly={readOnly} />
              ))}
            </Stack>
          )}
        </Stack>
      )}
    </Card>
  );
}

/** The value boxes for a kind: one per measure (two for a blood pressure), or a note. */
function ValueBoxes({ view, kind, inputs, setInputs, note, setNote }: {
  view: ObservationsView; kind: string; inputs: string[]; setInputs: (v: string[]) => void; note: string; setNote: (v: string) => void;
}) {
  const k = kindOf(view, kind);
  if (!k) return null;
  if (k.note) {
    return <Textarea label={`${k.label} note`} autosize minRows={2} value={note} onChange={(e) => setNote(e.currentTarget.value)} style={{ flex: 1, minWidth: 220 }} />;
  }
  return (
    <>
      {k.measures.map((m, i) => {
        const def = view.measures.find((x) => x.key === m);
        return (
          <TextInput key={m} label={`${def?.label ?? m} (${unitOf(m, def?.unit ?? "")})`} inputMode="decimal" w={160} value={inputs[i] ?? ""}
            onChange={(e) => {
              const next = [...inputs];
              next[i] = e.currentTarget.value;
              setInputs(next);
            }} />
        );
      })}
    </>
  );
}

/** OBS-03's warning: outside the usual range, the reading can still be saved. */
function SaveAnyway({ warning, saving, onSave, onChange }: { warning: string; saving: boolean; onSave: () => void; onChange: () => void }) {
  return (
    <Alert color="yellow" data-testid="observation-warning">
      <Text size="sm">{warning} Save it anyway?</Text>
      <Group gap="xs" mt="xs">
        <Button size="xs" color="yellow" loading={saving} onClick={onSave}>Save anyway</Button>
        <Button size="xs" variant="default" onClick={onChange}>Change it</Button>
      </Group>
    </Alert>
  );
}

function AddReading({ path, view, onSaved }: { path: string; view: ObservationsView; onSaved: () => void }) {
  const [kind, setKind] = useState<string | null>(null);
  const [inputs, setInputs] = useState<string[]>([]);
  const [note, setNote] = useState("");
  const [when, setWhen] = useState(() => toLocalInput(new Date()));
  const [whenTouched, setWhenTouched] = useState(false); // untouched: the time of Save (test report 1)
  const [warning, setWarning] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const choose = (k: string | null) => {
    setKind(k);
    if (!whenTouched) setWhen(toLocalInput(new Date()));
    setInputs([]);
    setNote("");
    setWarning("");
    setError("");
  };
  const save = async (ev?: FormEvent, anyway = false) => {
    ev?.preventDefault();
    if (saving) return;
    const req = readingRequest(view, kind ?? "", inputs, note, measuredAt(whenTouched, when, new Date()));
    if ("error" in req) {
      setError(req.error);
      return;
    }
    const soft = warningText(view, req.body.kind, inputs.map(parseNumber));
    if (soft && !anyway) {
      setWarning(soft); // OBS-03: a warning only; Save anyway sends it
      setError("");
      return;
    }
    setSaving(true);
    try {
      await api(`${path}/observations`, { method: "POST", body: JSON.stringify(req.body) });
      setWhenTouched(false);
      setWhen(toLocalInput(new Date()));
      // The kind stays chosen, for a second blood pressure.
      setInputs([]);
      setNote("");
      setWarning("");
      setError("");
      onSaved();
    } catch (err) {
      setError((err as Error).message); // what was typed stays
      setWarning("");
    } finally {
      setSaving(false);
    }
  };
  return (
    <Card withBorder component="form" onSubmit={save} noValidate bg="gray.0" data-testid="observation-add">
      <Stack gap="sm">
        <Group align="flex-end">
          <Select label="Add a reading" placeholder="What was measured" data={view.kinds.map((k) => ({ value: k.key, label: k.label }))}
            value={kind} onChange={choose} w={200} allowDeselect={false} />
          {kind && <ValueBoxes view={view} kind={kind} inputs={inputs} setInputs={(v) => { setInputs(v); setWarning(""); }} note={note} setNote={setNote} />}
          {kind && (
            <TextInput label="Measured at" type="datetime-local" value={when} w={220}
              description={whenTouched ? undefined : "The time you save it, unless you change it."}
              onChange={(e) => { setWhen(e.currentTarget.value); setWhenTouched(true); }} />
          )}
          {kind && !warning && <Button type="submit" loading={saving}>Save reading</Button>}
        </Group>
        {warning && (
          <SaveAnyway warning={warning} saving={saving} onSave={() => save(undefined, true)} onChange={() => setWarning("")} />
        )}
        {error && <Text size="sm" c="red" role="alert">{error}</Text>}
      </Stack>
    </Card>
  );
}

function ReadingEntry({ r, view, path, onChanged, readOnly }: { r: Reading; view: ObservationsView; path: string; onChanged: () => void; readOnly: boolean }) {
  const [open, setOpen] = useState<"" | "correct" | "error">("");
  const [inputs, setInputs] = useState<string[]>([]);
  const [note, setNote] = useState("");
  const [reason, setReason] = useState("");
  const [warning, setWarning] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const counts = counting([r]).length === 1;
  const label = kindOf(view, r.kind)?.label ?? r.kind;
  const corrected = r.supersededBy ? view.readings.find((x) => x.id === r.supersededBy) : undefined;
  const start = (what: "correct" | "error") => {
    setOpen(what);
    setInputs([]);
    setNote("");
    setReason("");
    setWarning("");
    setError("");
  };
  // Saved, or Back: the form's refusal goes with it (test report 2, F-4); only a 409's message stays.
  const close = () => {
    setOpen("");
    setWarning("");
    setError("");
  };
  const send = async (ev?: FormEvent, anyway = false) => {
    ev?.preventDefault();
    if (saving) return;
    let url: string;
    let body: unknown;
    if (open === "correct") {
      const req = correctionRequest(view, r.kind, inputs, note, reason);
      if ("error" in req) return setError(req.error);
      const soft = warningText(view, r.kind, inputs.map(parseNumber));
      if (soft && !anyway) {
        setWarning(soft); // OBS-03, as for a new reading (S46 verification)
        setError("");
        return;
      }
      url = `${path}/observations/${r.id}/correct`;
      body = req.body;
    } else {
      if (!reason.trim()) return setError("Say why the reading was entered in error.");
      url = `${path}/observations/${r.id}/entered-in-error`;
      body = { reason: reason.trim() };
    }
    setSaving(true);
    try {
      await api(url, { method: "POST", body: JSON.stringify(body) });
      close();
      onChanged();
    } catch (err) {
      setError((err as Error).message);
      setWarning("");
      // Another tab corrected or marked it meanwhile: show the list as it is now (test report 1).
      if (err instanceof ApiError && err.status === 409) onChanged();
    } finally {
      setSaving(false);
    }
  };
  const formShown = !!open && counts && !readOnly;
  return (
    <div data-testid="observation" data-counts={counts ? "true" : "false"}>
      <Group justify="space-between" wrap="nowrap" align="flex-start">
        <div style={{ minWidth: 0, overflowWrap: "anywhere" }}>
          <Group gap="xs">
            <Text size="sm" fw={600} td={counts ? undefined : "line-through"} c={counts ? undefined : "dimmed"}>
              {label}: {valueText(r, view)}
            </Text>
            {r.supersededBy && <Badge size="xs" color="gray" variant="outline">Corrected</Badge>}
            {r.errorAt && <Badge size="xs" color="gray" variant="outline">Entered in error</Badge>}
          </Group>
          <Text size="xs" c="dimmed" td={counts ? undefined : "line-through"}>{historyLine(r)}</Text>
          {corrected && <Text size="xs" c="dimmed">Corrected to {valueText(corrected, view)}</Text>}
          {r.errorAt && <Text size="xs" c="dimmed" data-testid="observation-error">{errorLine(r)}</Text>}
        </div>
        {counts && !readOnly && !open && (
          <Group gap={4} wrap="nowrap" style={{ flexShrink: 0 }}>
            <Button size="xs" variant="subtle" color="gray" onClick={() => start("correct")} aria-label={`Correct ${label} ${valueText(r, view)}`}>Correct</Button>
            <Button size="xs" variant="subtle" color="gray" onClick={() => start("error")} aria-label={`Mark ${label} ${valueText(r, view)} entered in error`}>
              Entered in error
            </Button>
          </Group>
        )}
      </Group>
      {error && !formShown && <Text size="sm" c="red" role="alert">{error}</Text>}
      {formShown && (
        <Card withBorder component="form" onSubmit={send} noValidate mt="xs" bg="gray.0" data-testid={open === "correct" ? "observation-correct" : "observation-error-form"}>
          <Stack gap="sm">
            {open === "correct" && (
              <Group align="flex-end">
                <ValueBoxes view={view} kind={r.kind} inputs={inputs} setInputs={(v) => { setInputs(v); setWarning(""); }} note={note} setNote={setNote} />
              </Group>
            )}
            <TextInput label={open === "correct" ? "Why is it corrected?" : "Why was it entered in error?"} maxLength={1000} value={reason}
              onChange={(e) => setReason(e.currentTarget.value)} autoFocus={open === "error"} />
            {open === "correct" && <Text size="xs" c="dimmed">The reading keeps its time. The old value stays in the history.</Text>}
            {open === "error" && r.corrects && (
              <Text size="xs" c="dimmed">The value it corrected won&apos;t count again. To put a value back, use Correct.</Text>
            )}
            {warning && (
              <SaveAnyway warning={warning} saving={saving} onSave={() => send(undefined, true)} onChange={() => setWarning("")} />
            )}
            {error && <Text size="sm" c="red" role="alert">{error}</Text>}
            {!warning && (
              <Group gap="xs">
                <Button type="submit" color={open === "error" ? "red" : undefined} variant={open === "error" ? "light" : "filled"} loading={saving}>
                  {open === "correct" ? "Save correction" : "Mark entered in error"}
                </Button>
                <Button variant="default" onClick={close}>Back</Button>
              </Group>
            )}
          </Stack>
        </Card>
      )}
    </div>
  );
}
