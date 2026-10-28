// The anaesthetist's ASA grade on the POA Summary (plan C6, session S44b): the grade that counts,
// its history underneath (newest first; a grade entered in error struck through, with who, when and
// why), Add grade (the reason box appears, and is required, once the episode has any grade), and "Entered in
// error" on each grade still standing. Nothing here edits or deletes a grade: the API has no way to.
// Each write sends the revision the page loaded; a stale one keeps what was typed, with Reload.
import { useState, type FormEvent } from "react";
import { Badge, Button, Card, Group, Select, Stack, Text, TextInput } from "@mantine/core";
import { api } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { addGradeRequest, asaLabel, countingGrade, needsReason, errorLine, givenLine, newestFirst, type AnaesthetistGrade } from "./asaGrades";
import { isStale, SaveError } from "./StatusControl";

const CHOICES = [1, 2, 3, 4, 5, 6].map((n) => ({ value: String(n), label: asaLabel(n) }));

type Props = { path: string; revision: number; grades: AnaesthetistGrade[]; onChanged: () => void; readOnly: boolean };

export function AnaesthetistGrades({ path, revision, grades, onChanged, readOnly }: Props) {
  const counting = countingGrade(grades);
  const reasonNeeded = needsReason(grades); // once any grade exists, even one entered in error
  const [grade, setGrade] = useState<string | null>(null);
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");
  const [stale, setStale] = useState(false);
  const [saving, setSaving] = useState(false);
  const fail = (err: unknown) => {
    setError((err as Error).message);
    setStale(isStale(err));
  };
  const add = async (ev?: FormEvent) => {
    ev?.preventDefault();
    if (saving) return;
    const req = addGradeRequest(grade, reason, reasonNeeded, revision);
    if ("error" in req) {
      setError(req.error);
      setStale(false);
      return;
    }
    setSaving(true);
    try {
      await api(`${path}/anaesthetist-asa`, { method: "POST", body: JSON.stringify(req.body) });
      setGrade(null);
      setReason("");
      setError("");
      onChanged();
    } catch (err) {
      fail(err); // what was typed stays
    } finally {
      setSaving(false);
    }
  };

  return (
    <Stack gap="xs" data-testid="anaesthetist-grades">
      <Text size="sm" fw={600}>
        Anaesthetist ASA grade: <span data-testid="anaesthetist-asa">{counting ? asaLabel(counting.grade) : "No grade yet"}</span>
      </Text>
      {grades.length > 0 && (
        <Stack gap={6} data-testid="asa-history">
          {newestFirst(grades).map((g) => (
            <GradeEntry key={g.id} g={g} counts={g.id === counting?.id} path={path} revision={revision} onChanged={onChanged} readOnly={readOnly} />
          ))}
        </Stack>
      )}
      {!readOnly && (
        <Card withBorder component="form" onSubmit={add} noValidate bg="gray.0" className="sj-no-print" data-testid="asa-add">
          <Stack gap="sm">
            <Group align="flex-end">
              <Select label="Anaesthetist ASA grade" placeholder="ASA grade" data={CHOICES} value={grade} onChange={setGrade} w={200} />
              {reasonNeeded && (
                <TextInput label="Why is the grade changed?" placeholder="e.g. New echo result" maxLength={1000} value={reason}
                  onChange={(ev) => setReason(ev.currentTarget.value)} style={{ flex: 1, minWidth: 200 }} required />
              )}
              <Button type="submit" variant="default" loading={saving}>Add grade</Button>
            </Group>
            {reasonNeeded && <Text size="xs" c="dimmed">The earlier grades stay in the history. A grade is never changed or deleted.</Text>}
            <SaveError error={error} stale={stale} onReload={() => { setError(""); setStale(false); onChanged(); }} />
          </Stack>
        </Card>
      )}
    </Stack>
  );
}

function GradeEntry({ g, counts, path, revision, onChanged, readOnly }: {
  g: AnaesthetistGrade; counts: boolean; path: string; revision: number; onChanged: () => void; readOnly: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");
  const [stale, setStale] = useState(false);
  const [saving, setSaving] = useState(false);
  const label = asaLabel(g.grade);
  const showForm = open && !g.errorAt && !readOnly; // after a Reload, a grade someone else marked closes its form
  const mark = async (ev?: FormEvent) => {
    ev?.preventDefault();
    if (saving) return;
    if (!reason.trim()) {
      setError("Say why the grade was entered in error.");
      setStale(false);
      return;
    }
    setSaving(true);
    try {
      await api(`${path}/anaesthetist-asa/${g.id}/entered-in-error`, { method: "POST", body: JSON.stringify({ reason: reason.trim(), revision }) });
      setOpen(false);
      setError("");
      onChanged();
    } catch (err) {
      setError((err as Error).message);
      setStale(isStale(err));
    } finally {
      setSaving(false);
    }
  };
  return (
    <div data-testid="asa-grade" data-error={g.errorAt ? "true" : "false"}>
      <Group justify="space-between" wrap="nowrap" align="flex-start">
        <div style={{ minWidth: 0 }}>
          <Group gap="xs">
            <Text size="sm" fw={600} td={g.errorAt ? "line-through" : undefined} c={g.errorAt ? "dimmed" : undefined}>{label}</Text>
            {counts && <Badge size="xs" variant="light">Counts</Badge>}
            {g.errorAt && <Badge size="xs" color="gray" variant="outline">Entered in error</Badge>}
          </Group>
          <Text size="xs" c="dimmed" td={g.errorAt ? "line-through" : undefined}>{givenLine(g)}</Text>
          {g.errorAt && <Text size="xs" c="dimmed" data-testid="asa-grade-error">{errorLine(g)}</Text>}
        </div>
        {!g.errorAt && !readOnly && !showForm && (
          // Never squeezed by a wrapped line beside it (test report 1, F1).
          <Button size="xs" variant="subtle" color="gray" className="sj-no-print" style={{ flexShrink: 0 }}
            onClick={() => { setOpen(true); setReason(""); setError(""); setStale(false); }}
            aria-label={`Mark ${label}, given on ${formatDateTime(g.givenAt)}, entered in error`}>
            Entered in error
          </Button>
        )}
      </Group>
      {showForm && (
        <Card withBorder component="form" onSubmit={mark} noValidate mt="xs" bg="gray.0" className="sj-no-print" data-testid="asa-error-form">
          <Stack gap="sm">
            <TextInput label="Why was it entered in error?" maxLength={1000} value={reason} onChange={(ev) => setReason(ev.currentTarget.value)} autoFocus />
            <SaveError error={error} stale={stale} onReload={() => { setError(""); setStale(false); onChanged(); }} />
            <Group gap="xs">
              <Button type="submit" color="red" variant="light" loading={saving}>Mark entered in error</Button>
              <Button variant="default" onClick={() => setOpen(false)}>Back</Button>
            </Group>
          </Stack>
        </Card>
      )}
    </div>
  );
}
