// Step 2 of plan-workflow.md: the hospital's episodes, and creating one for a patient with a
// published HQ. Made-up patients only (NFR-03). Trust-wide staff can also list every hospital's
// episodes ("All hospitals", STF-03), kept in the URL as ?hospitals=all.
import { useCallback, useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import {
  Alert, Badge, Button, Drawer, Group, SegmentedControl, Select, Stack, Table, Text, TextInput, Title,
} from "@mantine/core";
import { IconPlus } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { hospitalBase, useHospital } from "@/lib/auth";
import { formatDateTime } from "@/lib/format";
import { patientName, STATUS_COLORS, STATUS_LABELS, type EpisodeRow, type EpisodeStatus, type Patient } from "./types";

type ListRow = EpisodeRow & { hospitalId: string; hospitalName: string };

export function EpisodeList({ base }: { base: string }) {
  const navigate = useNavigate();
  const { trust, hospital } = useHospital();
  const [params, setParams] = useSearchParams();
  // "All hospitals" is for trust-wide staff only; anyone else gets this hospital's list.
  const trustWide = trust.role !== null;
  const all = trustWide && params.get("hospitals") === "all";
  const [status, setStatus] = useState<string | null>(null);
  const [rows, setRows] = useState<ListRow[] | null>(null);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);

  const load = useCallback(() => {
    const from = all ? `/o/${trust.id}` : base;
    api<ListRow[]>(`${from}/episodes?${new URLSearchParams({ status: status ?? "" })}`)
      .then((r) => {
        setRows(r);
        setError("");
      })
      .catch((e: Error) => setError(e.message));
  }, [all, trust.id, base, status]);
  useEffect(load, [load]);

  return (
    <Stack>
      <Group justify="space-between">
        <Title order={3}>Episodes</Title>
        <Button leftSection={<IconPlus size={16} />} onClick={() => setCreating(true)}>New episode</Button>
      </Group>
      {trustWide && (
        <SegmentedControl
          aria-label="Hospitals"
          w="fit-content"
          data={[
            { value: "one", label: hospital.name },
            { value: "all", label: "All hospitals" },
          ]}
          value={all ? "all" : "one"}
          onChange={(v) => setParams(v === "all" ? { hospitals: "all" } : {})}
        />
      )}
      <Select
        aria-label="Status"
        placeholder="All statuses"
        clearable
        data={Object.entries(STATUS_LABELS).map(([value, label]) => ({ value, label }))}
        value={status}
        onChange={setStatus}
        maw={260}
      />
      {error && <Alert color="red">{error}</Alert>}
      {rows && rows.length === 0 && <Text c="dimmed">No episodes yet.</Text>}
      {rows && rows.length > 0 && (
        <Table highlightOnHover verticalSpacing="sm">
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Patient</Table.Th>
              {all && <Table.Th>Hospital</Table.Th>}
              <Table.Th>Procedure</Table.Th>
              <Table.Th>HQ</Table.Th>
              <Table.Th>Status</Table.Th>
              <Table.Th>Created</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {rows.map((e) => (
              <Table.Tr key={e.id} onClick={() => navigate(`${hospitalBase(trust.id, e.hospitalId)}/episodes/${e.id}`)} style={{ cursor: "pointer" }}>
                <Table.Td>
                  <Text fw={700}>{patientName(e)}</Text>
                  {e.hospitalNumber && <Text size="sm" c="dimmed">{e.hospitalNumber}</Text>}
                </Table.Td>
                {all && <Table.Td>{e.hospitalName}</Table.Td>}
                <Table.Td>{e.procedure || "—"}</Table.Td>
                <Table.Td>{e.questionnaireName} v{e.versionNo}</Table.Td>
                <Table.Td>
                  <StatusBadge status={e.status} />
                </Table.Td>
                <Table.Td>{formatDateTime(e.createdAt)}</Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}
      <NewEpisode
        base={base}
        opened={creating}
        onClose={() => setCreating(false)}
        onCreated={(id) => navigate(`${base}/episodes/${id}`)}
      />
    </Stack>
  );
}

type Published = { versionId: string; versionNo: number; name: string };
const EMPTY_PATIENT = { firstName: "", lastName: "", dateOfBirth: "", sex: "", hospitalNumber: "", phone: "" };
const ANAESTHETICS = ["General", "Regional", "Local", "Sedation"];

function NewEpisode(props: { base: string; opened: boolean; onClose: () => void; onCreated: (id: string) => void }) {
  const { base } = props;
  const [patients, setPatients] = useState<Patient[]>([]);
  const [hqs, setHqs] = useState<Published[]>([]);
  const [mode, setMode] = useState<"existing" | "new">("existing");
  const [patientId, setPatientId] = useState<string | null>(null);
  const [patient, setPatient] = useState(EMPTY_PATIENT);
  const [versionId, setVersionId] = useState<string | null>(null);
  const [details, setDetails] = useState({ procedure: "", anaesthetic: "", consultant: "" });
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!props.opened) return;
    api<Patient[]>(`${base}/patients`).then(setPatients).catch((e: Error) => setError(e.message));
    api<Published[]>(`${base}/published-hqs`).then(setHqs).catch((e: Error) => setError(e.message));
  }, [props.opened, base]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      let pid = patientId;
      if (mode === "new") pid = (await api<{ id: string }>(`${base}/patients`, { method: "POST", body: JSON.stringify(patient) })).id;
      const res = await api<{ id: string }>(`${base}/episodes`, {
        method: "POST",
        body: JSON.stringify({ patientId: pid, versionId, ...details }),
      });
      props.onCreated(res.id);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const field = (key: keyof typeof EMPTY_PATIENT) => ({
    value: patient[key],
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => setPatient({ ...patient, [key]: e.currentTarget.value }),
  });

  return (
    <Drawer opened={props.opened} onClose={props.onClose} position="right" title="New episode" size="md">
      <form onSubmit={submit}>
        <Stack>
          {error && <Alert color="red">{error}</Alert>}
          <SegmentedControl
            value={mode}
            onChange={(v) => setMode(v as "existing" | "new")}
            data={[{ value: "existing", label: "Existing patient" }, { value: "new", label: "New patient" }]}
          />
          {mode === "existing" ? (
            <Select
              label="Patient"
              required
              searchable
              data={patients.map((p) => ({ value: p.id, label: `${patientName(p)}${p.hospitalNumber ? ` · ${p.hospitalNumber}` : ""}` }))}
              value={patientId}
              onChange={setPatientId}
            />
          ) : (
            <>
              <Text size="xs" c="dimmed">Made-up details only. Never enter a real patient.</Text>
              <Group grow>
                <TextInput label="First name" required {...field("firstName")} />
                <TextInput label="Last name" required {...field("lastName")} />
              </Group>
              <Group grow>
                <TextInput label="Date of birth" type="date" required {...field("dateOfBirth")} />
                <Select
                  label="Sex"
                  required
                  data={[
                    { value: "female", label: "Female" },
                    { value: "male", label: "Male" },
                    { value: "other", label: "Other" },
                    { value: "unknown", label: "Not known" },
                  ]}
                  value={patient.sex || null}
                  onChange={(v) => setPatient({ ...patient, sex: v ?? "" })}
                />
              </Group>
              <Group grow>
                <TextInput label="Hospital number" {...field("hospitalNumber")} />
                <TextInput label="Phone" {...field("phone")} />
              </Group>
            </>
          )}
          <Select
            label="Health questionnaire"
            description="Only published versions can be given to patients."
            required
            data={hqs.map((h) => ({ value: h.versionId, label: `${h.name} v${h.versionNo}` }))}
            value={versionId}
            onChange={setVersionId}
            nothingFoundMessage="Nothing published yet"
          />
          <TextInput label="Procedure" value={details.procedure} onChange={(e) => setDetails({ ...details, procedure: e.currentTarget.value })} />
          <Select
            label="Anaesthetic"
            data={ANAESTHETICS}
            clearable
            value={details.anaesthetic || null}
            onChange={(v) => setDetails({ ...details, anaesthetic: v ?? "" })}
          />
          <TextInput label="Consultant" value={details.consultant} onChange={(e) => setDetails({ ...details, consultant: e.currentTarget.value })} />
          <Group justify="flex-end">
            <Button variant="default" onClick={props.onClose}>Cancel</Button>
            <Button type="submit" loading={saving}>Create episode</Button>
          </Group>
        </Stack>
      </form>
    </Drawer>
  );
}

export function StatusBadge({ status }: { status: EpisodeStatus }) {
  return <Badge variant="light" color={STATUS_COLORS[status]}>{STATUS_LABELS[status]}</Badge>;
}
