// The episode's status control (S20): the "Episode status" dropdown offers only the moves this
// user may make from here (Go's transition table sends them as `moves`), with Cancel episode,
// Re-open HQ review and Undo as buttons when they're allowed. S22 adds Archive episode and
// Un-archive, for super clinicians (`moves` holds the archive; `unarchive` its way back). Every move opens a short inline
// form first (no modal): the date for Admitted and Discharged, the reason (required for cancel,
// re-open and undo, optional otherwise). Each request sends the revision the page loaded; a
// stale one is refused, and the form keeps what was typed, with Reload. If after the reload the
// move is no longer offered, its form closes, saying so (test report 1).
//
// The reason is one line (a sentence), so Enter sends the form, like every other form here (S20
// test report 1, F-1); at most 1,000 characters, as the API allows.
import { useEffect, useRef, useState, type FormEvent } from "react";
import { Alert, Button, Card, Group, Radio, Select, Stack, Table, Text, TextInput, Title } from "@mantine/core";
import { ApiError, api } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import type { EpisodeDetail } from "./EpisodePage";
import { partlyTypedDate } from "./dates";
import { CANCEL_REASONS, historyText, historyWho, pickOptions, statusRequest, type StatusEvent, type StatusForm } from "./status";
import { STATUS_LABELS } from "./types";

/** The longest reason the API takes (statuses.go maxReasonLen). */
const MAX_REASON = 1000;

/** Today's date in the browser, YYYY-MM-DD: the date fields' first value. */
function todayInput(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

/** Whether an API error is the stale-revision 409, which Reload fixes. */
export const isStale = (e: unknown) => e instanceof ApiError && e.status === 409 && (e.body as { code?: string } | null)?.code === "stale";

/** A refusal, with Reload when the page is out of date (the input stays as typed). */
export function SaveError({ error, stale, onReload }: { error: string; stale: boolean; onReload: () => void }) {
  if (!error) return null;
  return (
    <Alert color="red" py={6}>
      <Group justify="space-between" gap="xs">
        <Text size="sm">{error}</Text>
        {stale && <Button size="xs" variant="white" color="red" onClick={onReload}>Reload</Button>}
      </Group>
    </Alert>
  );
}

export function StatusControl({ path, detail, onChanged }: { path: string; detail: EpisodeDetail; onChanged: () => void }) {
  const e = detail.episode;
  const picks = pickOptions(detail.moves);
  const can = (move: string) => detail.moves.some((m) => m.move === move);
  const [form, setForm] = useState<StatusForm | null>(null);
  const [error, setError] = useState("");
  const [stale, setStale] = useState(false);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState("");
  const dateRef = useRef<HTMLInputElement>(null);

  // After a reload, a form for a move that's no longer offered closes (the API would refuse it).
  const offered = !form || (form.kind === "pick" ? picks.some((m) => m.to === form.to?.to)
    : form.kind === "undo" ? detail.undo !== null : form.kind === "unarchive" ? detail.unarchive !== null : can(form.kind));
  useEffect(() => {
    if (!offered) {
      setForm(null);
      setError("");
      setNotice("The episode changed, so that change is no longer offered.");
    }
  }, [offered]);

  const open = (f: Omit<StatusForm, "reason" | "details" | "date">) => {
    setForm({ ...f, reason: "", details: "", date: f.to?.needsDate ? todayInput() : "" });
    setError("");
    setNotice("");
    setStale(false);
  };
  const close = () => {
    setForm(null);
    setError("");
  };

  const send = async (ev?: FormEvent) => {
    ev?.preventDefault(); // Enter in a field sends the form
    if (!form || saving) return;
    const what = form.to?.move === "discharge" ? "discharge date" : "admission date";
    const partly = partlyTypedDate([{ label: what, input: dateRef.current }]);
    const req = partly ? { error: partly } : statusRequest(form, e.revision);
    if ("error" in req) {
      setError(req.error);
      setStale(false);
      return;
    }
    setSaving(true);
    try {
      await api(`${path}${req.path}`, { method: req.path === "/status" ? "PATCH" : "POST", body: JSON.stringify(req.body) });
      close();
      onChanged();
    } catch (err) {
      setError((err as Error).message);
      setStale(isStale(err));
    } finally {
      setSaving(false);
    }
  };

  const title = !form ? "" : {
    pick: `Change the status to ${form.to?.label}`,
    cancel: "Cancel this episode",
    reopen: "Re-open the HQ review",
    undo: `Undo the last status change: back to ${detail.undo?.label}`,
    archive: "Archive this episode",
    unarchive: `Un-archive this episode: back to ${detail.unarchive?.label}`,
  }[form.kind];
  const confirm = !form ? "" : {
    pick: "Change status", cancel: "Cancel episode", reopen: "Re-open review", undo: "Undo", archive: "Archive episode", unarchive: "Un-archive",
  }[form.kind];

  return (
    <Stack gap="sm" data-testid="status-control">
      <Group gap="sm">
        <Text size="sm" fw={600}>Episode status:</Text>
        <Select
          aria-label="Episode status"
          data={[{ value: e.status, label: STATUS_LABELS[e.status], disabled: true }, ...picks.map((m) => ({ value: m.to, label: m.label }))]}
          value={e.status}
          onChange={(v) => {
            const to = picks.find((m) => m.to === v);
            if (to) open({ kind: "pick", to });
          }}
          allowDeselect={false}
          disabled={picks.length === 0 || saving}
          w={240}
        />
        {can("reopen") && <Button size="xs" variant="default" onClick={() => open({ kind: "reopen" })}>Re-open HQ review</Button>}
        {detail.undo && <Button size="xs" variant="default" onClick={() => open({ kind: "undo" })}>Undo: back to {detail.undo.label}</Button>}
        {can("cancel") && <Button size="xs" variant="light" color="red" onClick={() => open({ kind: "cancel" })}>Cancel episode</Button>}
        {can("archive") && <Button size="xs" variant="default" onClick={() => open({ kind: "archive" })}>Archive episode</Button>}
        {detail.unarchive && (
          <Button size="xs" variant="default" onClick={() => open({ kind: "unarchive" })}>Un-archive: back to {detail.unarchive.label}</Button>
        )}
      </Group>
      {notice && <Text size="sm" c="orange.8" data-testid="status-notice">{notice}</Text>}
      {(e.status === "hq_not_complete" || e.status === "ready_for_review") && (
        <Text size="xs" c="dimmed">The status can be changed by hand once the HQ review is complete.</Text>
      )}

      {form && (
        <Card withBorder component="form" onSubmit={send} noValidate data-testid="status-form" bg="gray.0">
          <Stack gap="sm">
            <Title order={5}>{title}</Title>
            {form.kind === "reopen" && (
              <Text size="sm" c="dimmed">The episode goes back to Ready for review, and the clinician&apos;s answers can be changed again. The answers and validations stay, and the patient isn&apos;t asked again.</Text>
            )}
            {form.kind === "cancel" && (
              <Text size="sm" c="dimmed">Cancelling is final. The patient&apos;s link stops working, reminders stop, and their home shows the questionnaire as cancelled by the hospital.</Text>
            )}
            {form.kind === "archive" && (
              <Text size="sm" c="dimmed">An archived episode can&apos;t be changed, and leaves every worklist. It can still be found by searching, on the patient&apos;s page, and in the episode list with Show archived. Reminders stop, and the patient&apos;s questionnaire is closed. A super clinician can un-archive it.</Text>
            )}
            {form.kind === "unarchive" && (
              <Text size="sm" c="dimmed">The episode goes back to {detail.unarchive?.label}, the status it was archived from, and can be changed again. Reminders don&apos;t start again; resend the invite if the patient still needs it.</Text>
            )}
            {form.to?.needsDate && (
              <TextInput
                label={form.to.move === "discharge" ? "Discharge date" : "Admission date"}
                type="date"
                ref={dateRef}
                value={form.date}
                onChange={(ev) => setForm({ ...form, date: ev.currentTarget.value })}
                w={220}
              />
            )}
            {form.kind === "cancel" ? (
              <>
                <Radio.Group label="Reason" value={form.reason} onChange={(v) => setForm({ ...form, reason: v })}>
                  <Stack gap={6} mt={4}>
                    {CANCEL_REASONS.map((r) => <Radio key={r.value} value={r.value} label={r.label} />)}
                  </Stack>
                </Radio.Group>
                <TextInput
                  label={form.reason === "other" ? "Say why" : "Details (optional)"}
                  maxLength={MAX_REASON}
                  value={form.details}
                  onChange={(ev) => setForm({ ...form, details: ev.currentTarget.value })}
                />
              </>
            ) : (
              <TextInput
                label={form.kind === "pick" || form.kind === "archive" ? "Reason (optional)" : "Reason"}
                maxLength={MAX_REASON}
                value={form.reason}
                onChange={(ev) => setForm({ ...form, reason: ev.currentTarget.value })}
                data-autofocus
              />
            )}
            <SaveError error={error} stale={stale} onReload={() => { setError(""); setStale(false); onChanged(); }} />
            <Group gap="xs">
              <Button type="submit" loading={saving} color={form.kind === "cancel" ? "red" : undefined}>{confirm}</Button>
              <Button variant="default" onClick={close}>Back</Button>
            </Group>
          </Stack>
        </Card>
      )}
    </Stack>
  );
}

/** The status history (EPI-03): every change, with who, when and why; undone ones are marked. */
export function StatusHistory({ events }: { events: StatusEvent[] }) {
  if (events.length === 0) return <Text size="sm" c="dimmed">No status changes are recorded yet.</Text>;
  return (
    <Table verticalSpacing={6} fz="sm" data-testid="status-history">
      <Table.Thead>
        <Table.Tr>
          <Table.Th>When</Table.Th>
          <Table.Th>Change</Table.Th>
          <Table.Th>By</Table.Th>
          <Table.Th>Reason</Table.Th>
        </Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        {events.map((ev) => (
          <Table.Tr key={ev.id} data-testid="status-history-row">
            <Table.Td>{formatDateTime(ev.createdAt)}</Table.Td>
            <Table.Td>
              {historyText(ev, STATUS_LABELS)}
              {ev.undoneAt && <Text span size="xs" c="dimmed"> (undone)</Text>}
            </Table.Td>
            <Table.Td>{historyWho(ev)}</Table.Td>
            <Table.Td>{ev.reason || "—"}</Table.Td>
          </Table.Tr>
        ))}
      </Table.Tbody>
    </Table>
  );
}
