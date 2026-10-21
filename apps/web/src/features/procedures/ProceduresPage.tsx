// The trust's procedure list (plan C3, S19; PRC-01), for its whole-trust super clinicians and
// admins (S19 answers 1); the API (/api/o/{trust}/procedures) refuses changes from anyone else.
// Add, edit, retire and bring back. A procedure is never deleted: a retired one isn't offered for
// new episodes, but stays on the episodes that have it (answers 5).
//
// The form is inline, not a modal: testers' browser tabs can hide Mantine's modals (decisions.md §4).
import { useCallback, useEffect, useRef, useState } from "react";
import { Alert, Badge, Button, Card, Group, Select, SimpleGrid, Stack, Table, Text, TextInput, Title } from "@mantine/core";
import { IconPlus } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { canManageProcedures, useHospital } from "@/lib/auth";
import { changedFields, type Procedure, type PublishedHQ } from "./procedures";

const EVERY = "every"; // the hospital Select's value for "every hospital of the trust"
const NONE = "none"; // the default HQ Select's value for "no default HQ"

type Form = { name: string; code: string; specialty: string; hospital: string; defaultHQ: string };
const EMPTY: Form = { name: "", code: "", specialty: "", hospital: EVERY, defaultHQ: NONE };

const formOf = (p: Procedure): Form => ({
  name: p.name, code: p.code, specialty: p.specialty, hospital: p.hospitalId ?? EVERY, defaultHQ: p.defaultQuestionnaireId ?? NONE,
});

/** The POST or PATCH body for a form: every field, with null for "every hospital" and "no default HQ". */
const bodyOf = (f: Form) => ({
  name: f.name, code: f.code, specialty: f.specialty,
  hospitalId: f.hospital === EVERY ? null : f.hospital,
  defaultQuestionnaireId: f.defaultHQ === NONE ? null : f.defaultHQ,
});

export function ProceduresPage() {
  const { trust, base } = useHospital();
  const allowed = canManageProcedures(trust);
  const path = `/o/${trust.id}/procedures`;
  const [list, setList] = useState<Procedure[] | null>(null);
  const [hqs, setHqs] = useState<PublishedHQ[]>([]);
  const [editing, setEditing] = useState<Procedure | "new" | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  // Bring a new message into view, after the list has reloaded too: Retire or Bring back on a row
  // further down leaves the page scrolled away from it (S19 test reports 2 and 3). The message is
  // at the top of the page, under the fixed header, so scroll to the top. Instant, since the
  // reload's change to the page cancels a smooth scroll part-way.
  useEffect(() => {
    if (notice) window.scrollTo({ top: 0 });
  }, [notice, list]);

  const load = useCallback(() => {
    api<Procedure[]>(path).then(setList).catch((e: Error) => setError(e.message));
  }, [path]);
  useEffect(() => {
    if (!allowed) return;
    load();
    // The trust's published questionnaires, from this hospital (they're the trust's, so every hospital has them).
    api<PublishedHQ[]>(`${base}/published-hqs`).then(setHqs).catch((e: Error) => setError(e.message));
  }, [allowed, load, base]);

  if (!allowed) return <Alert color="red" title="Only super clinicians and admins of the whole trust can manage its procedures" />;
  if (!list) return error ? <Alert color="red">{error}</Alert> : null;

  const saved = (text: string) => {
    setEditing(null);
    setError("");
    setNotice(text);
    load();
  };
  const setRetired = async (p: Procedure, retired: boolean) => {
    setNotice("");
    try {
      await api(`${path}/${p.id}`, { method: "PATCH", body: JSON.stringify({ retired }) });
      saved(retired ? `${p.name} is retired: it isn't offered for new episodes.` : `${p.name} is offered again.`);
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <Stack maw={1100}>
      <Group justify="space-between" align="flex-start">
        <div>
          <Title order={3}>Procedures</Title>
          <Text size="sm" c="dimmed">
            {trust.name}&apos;s procedures. Picking one on a new episode chooses its default HQ; without one, the hospital&apos;s default
            HQ. A retired procedure isn&apos;t offered for new episodes, but stays on the episodes that have it.
          </Text>
        </div>
        {editing === null && (
          <Button leftSection={<IconPlus size={16} />} onClick={() => { setEditing("new"); setNotice(""); }}>Add procedure</Button>
        )}
      </Group>
      {notice && <Alert color="green">{notice}</Alert>}
      {error && editing === null && <Alert color="red">{error}</Alert>}
      {editing !== null && (
        <ProcedureForm
          key={editing === "new" ? "new" : editing.id}
          path={path}
          procedure={editing === "new" ? null : editing}
          hospitals={trust.hospitals}
          hqs={hqs}
          onSaved={saved}
          onCancel={() => setEditing(null)}
        />
      )}
      {list.length === 0 ? (
        <Text c="dimmed">No procedures yet.</Text>
      ) : (
        <Table verticalSpacing="sm" aria-label="Procedures">
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Name</Table.Th>
              <Table.Th>Code</Table.Th>
              <Table.Th>Specialty</Table.Th>
              <Table.Th>Offered at</Table.Th>
              <Table.Th>Default HQ</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {list.map((p) => (
              <Table.Tr key={p.id} data-testid={`procedure-${p.name}`} c={p.retired ? "dimmed" : undefined}>
                <Table.Td>
                  <Group gap="xs">
                    <Text fw={600} size="sm">{p.name}</Text>
                    {p.retired && <Badge color="gray" variant="light">Retired</Badge>}
                  </Group>
                </Table.Td>
                <Table.Td>{p.code || "—"}</Table.Td>
                <Table.Td>{p.specialty || "—"}</Table.Td>
                <Table.Td>{p.hospitalName ?? "Every hospital"}</Table.Td>
                <Table.Td>{p.defaultQuestionnaireName ?? "The hospital's default"}</Table.Td>
                <Table.Td ta="right">
                  <Group gap="xs" justify="flex-end" wrap="nowrap">
                    <Button size="xs" variant="default" disabled={editing !== null} onClick={() => { setEditing(p); setNotice(""); }}>Edit</Button>
                    <Button size="xs" variant="default" color={p.retired ? undefined : "red"} disabled={editing !== null}
                      onClick={() => setRetired(p, !p.retired)}>
                      {p.retired ? "Bring back" : "Retire"}
                    </Button>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}
    </Stack>
  );
}

function ProcedureForm({ path, procedure, hospitals, hqs, onSaved, onCancel }: {
  path: string;
  procedure: Procedure | null; // null: a new one
  hospitals: { id: string; name: string }[];
  hqs: PublishedHQ[];
  onSaved: (notice: string) => void;
  onCancel: () => void;
}) {
  const [form, setForm] = useState<Form>(procedure ? formOf(procedure) : EMPTY);
  const [saving, setSaving] = useState(false);
  // Edit on a row further down opens the form at the top: bring it into view (S19 test report 1).
  const card = useRef<HTMLFormElement>(null);
  useEffect(() => {
    card.current?.scrollIntoView?.({ block: "nearest", behavior: "smooth" });
  }, []);
  const [error, setError] = useState("");
  const text = (key: "name" | "code" | "specialty") => ({
    value: form[key],
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: e.currentTarget.value }),
  });
  // The default HQ it has may no longer be published: keep it shown so saving doesn't drop it.
  const hqChoices = hqs.map((h) => ({ value: h.questionnaireId, label: h.name }));
  if (procedure?.defaultQuestionnaireId && !hqChoices.some((c) => c.value === procedure.defaultQuestionnaireId)) {
    hqChoices.push({ value: procedure.defaultQuestionnaireId, label: `${procedure.defaultQuestionnaireName} (not published now)` });
  }
  const hospitalChoices = [{ value: EVERY, label: "Every hospital of the trust" }, ...hospitals.map((h) => ({ value: h.id, label: `${h.name} only` }))];
  if (procedure?.hospitalId && !hospitals.some((h) => h.id === procedure.hospitalId)) {
    hospitalChoices.push({ value: procedure.hospitalId, label: `${procedure.hospitalName} only` });
  }

  const save = async (e?: React.FormEvent) => {
    e?.preventDefault(); // Enter in a field sends the form
    if (saving) return;
    setError(""); // a new attempt replaces the last message
    setSaving(true);
    try {
      const body = bodyOf(form);
      if (procedure) {
        // Send only what changed, so a default HQ that's no longer published isn't checked again.
        const changed = changedFields(bodyOf(formOf(procedure)), body);
        if (Object.keys(changed).length === 0) {
          onCancel();
          return;
        }
        await api(`${path}/${procedure.id}`, { method: "PATCH", body: JSON.stringify(changed) });
        onSaved(`${form.name.trim()} is saved.`);
      } else {
        await api(path, { method: "POST", body: JSON.stringify(body) });
        onSaved(`${form.name.trim()} is added.`);
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  // noValidate: the API's own messages, not the browser's "Please fill in this field." (S19 test report 2).
  return (
    <Card withBorder data-testid="procedure-form" ref={card} component="form" onSubmit={save} noValidate>
      <Title order={4} mb="sm">{procedure ? `Edit ${procedure.name}` : "Add a procedure"}</Title>
      <SimpleGrid cols={{ base: 1, sm: 2 }}>
        <TextInput label="Name" required {...text("name")} />
        <TextInput label="Code" description="Optional" {...text("code")} />
        <TextInput label="Specialty" description="Optional" {...text("specialty")} />
        <Select label="Offered at" data={hospitalChoices} value={form.hospital} allowDeselect={false}
          onChange={(v) => setForm({ ...form, hospital: v ?? EVERY })} />
        <Select
          label="Default HQ"
          description="Chosen first when this procedure is picked on a new episode. Without one, the hospital's default HQ is."
          data={[{ value: NONE, label: "None: use the hospital's default HQ" }, ...hqChoices]}
          value={form.defaultHQ}
          allowDeselect={false}
          onChange={(v) => setForm({ ...form, defaultHQ: v ?? NONE })}
        />
      </SimpleGrid>
      {error && <Alert color="red" mt="sm">{error}</Alert>}
      <Group justify="flex-end" mt="md">
        <Button variant="default" onClick={onCancel}>Cancel</Button>
        <Button type="submit" loading={saving}>{procedure ? "Save changes" : "Add procedure"}</Button>
      </Group>
    </Card>
  );
}
