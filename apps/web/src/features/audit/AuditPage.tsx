// The trust's audit log (plan C1, S11; STF-06): every request that read or changed patient data,
// newest first, with who, when, the hospital, what, and the patients it touched. Filters by person,
// days and patient, kept in the URL (filters.ts), and a CSV export of everything the filters match.
// Only the trust's whole-trust admins see it; the API (/api/o/{trust}/audit) refuses everyone else.
//
// Every load of this list, and every export, is itself written to the log (AUD-01), so the list
// is only fetched when the filters are applied, not on each keystroke.
import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { Alert, Badge, Button, Group, Loader, Select, Stack, Table, Text, TextInput, Title, Tooltip } from "@mantine/core";
import { IconDownload } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { canSeeAudit, useHospital } from "@/lib/auth";
import { formatDateTime } from "@/lib/format";
import { auditQuery, badDays, namesOf, readFilters, withoutBadDays, type AuditFilters } from "./filters";

/** One row, as GET /api/o/{trust}/audit sends it (internal/httpapi/audit.go, auditListRow). */
type Row = {
  id: number;
  at: string;
  actor: "staff" | "patient";
  userId: string | null;
  userName: string;
  userEmail: string;
  hospitalId: string | null;
  hospitalName: string; // "" for a trust route ("All hospitals", the audit log itself)
  action: string;
  actionLabel: string;
  route: string;
  status: number;
  patients: { id: string; name: string; hospitalNumber: string }[]; // the first ten
  patientCount: number; // every patient the request touched
  episodeIds: string[];
};
type Page = { rows: Row[]; more: boolean; limit: number; users: { id: string; name: string; email: string }[] };

const PATIENT = "patient"; // the "Who" filter's value for the patient's own link

export function AuditPage() {
  const { trust } = useHospital();
  const allowed = canSeeAudit(trust);
  const [params, setParams] = useSearchParams();
  const applied = readFilters(params);
  const key = params.toString(); // the applied filters, as one value for the effect below
  // The form's values: applied when Show is pressed (or Enter), which writes them to the URL.
  const [form, setForm] = useState<AuditFilters>(applied);
  const [page, setPage] = useState<Page | null>(null);
  // The people for the Who box, from the last list that loaded: kept when a later load is refused,
  // so the person chosen still shows by name (S11 test report 2).
  const [users, setUsers] = useState<Page["users"]>([]);
  const [error, setError] = useState("");
  const [failed, setFailed] = useState(false); // the applied filters were refused: nothing to export
  const [loading, setLoading] = useState(false);
  const [refresh, setRefresh] = useState(0); // Show with the same filters loads the list again
  // The filters this page put in the URL itself. Any others came from back, forward or a link,
  // and only those are copied into the form: otherwise a slow navigation could undo typing.
  const ownKey = useRef<string | null>(null);

  useEffect(() => {
    if (!allowed) return;
    const filters = readFilters(new URLSearchParams(key));
    if (key !== ownKey.current) {
      // A date box can't show a day that isn't real (30 February), so it starts empty instead.
      setForm(withoutBadDays(filters));
      ownKey.current = key;
    }
    // A day that isn't a real one (e.g. 30 February in a typed link) would otherwise be dropped
    // from the search without a word. The message names it, since the box can't show it.
    if (badDays(filters)) {
      const bad = filters.from !== withoutBadDays(filters).from ? filters.from : filters.to;
      setPage(null);
      setFailed(true);
      setError(`Enter valid dates. There's no ${bad}.`);
      return;
    }
    // Only the latest load may fill the page: a slower, older one is ignored when it lands.
    let current = true;
    setLoading(true);
    const q = auditQuery(filters).toString();
    api<Page>(`/o/${trust.id}/audit${q ? `?${q}` : ""}`)
      .then((p) => {
        if (!current) return;
        setPage(p);
        setUsers(p.users);
        setFailed(false);
        setError("");
      })
      .catch((e: Error) => {
        if (!current) return;
        setPage(null); // never leave the previous filters' rows under an error
        setFailed(true);
        setError(e.message);
      })
      .finally(() => {
        if (current) setLoading(false);
      });
    return () => {
      current = false;
    };
  }, [allowed, trust.id, key, refresh]);

  if (!allowed) {
    return <Alert color="red" title="Only the trust's admins can see the audit log" />;
  }

  const apply = (f: AuditFilters) => {
    if (badDays(f)) {
      setError("Enter valid dates.");
      return;
    }
    if (f.from && f.to && f.from > f.to) {
      setError("The start date must be before the end date.");
      return;
    }
    const next = new URLSearchParams();
    for (const [k, v] of Object.entries(f)) if (v.trim()) next.set(k, v.trim());
    setForm(f); // Clear empties the form too
    ownKey.current = next.toString();
    if (next.toString() === key) setRefresh((n) => n + 1);
    else setParams(next);
    setError("");
  };
  const csvHref = `/api/o/${trust.id}/audit?${auditQuery(applied, {
    csv: true,
    timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  })}`;
  const people = [
    { value: PATIENT, label: "Patients (their own link)" },
    ...users.map((u) => ({ value: u.id, label: `${u.name} (${u.email})` })),
  ];
  // A filter for someone no longer in the list (e.g. a deep link) still shows as chosen.
  if (applied.user && !people.some((p) => p.value === applied.user)) people.push({ value: applied.user, label: "Someone else" });

  return (
    <Stack>
      <Group justify="space-between">
        <div>
          <Title order={3}>Audit log</Title>
          <Text size="sm" c="dimmed">
            Every view and change of patient data in {trust.name}, newest first. Nothing here can be changed or deleted.
          </Text>
        </div>
        {/* Greyed out (and no link) while the applied filters are refused: it would download the error. */}
        <Button component="a" href={failed ? undefined : csvHref} download variant="default"
          leftSection={<IconDownload size={16} />} disabled={failed}>
          Export CSV
        </Button>
      </Group>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          apply(form);
        }}
      >
        <Group align="end" wrap="wrap">
          <Select
            label="Who"
            placeholder="Everyone"
            clearable
            searchable
            data={people}
            value={form.user || null}
            onChange={(v) => setForm({ ...form, user: v ?? "" })}
            w={280}
          />
          <TextInput label="From" type="date" value={form.from} onChange={(e) => setForm({ ...form, from: e.currentTarget.value })} />
          <TextInput label="To" type="date" value={form.to} onChange={(e) => setForm({ ...form, to: e.currentTarget.value })} />
          <TextInput
            label="Patient"
            placeholder="Name or hospital number"
            value={form.patient}
            onChange={(e) => setForm({ ...form, patient: e.currentTarget.value })}
            w={220}
          />
          <Button type="submit">Show</Button>
          <Button variant="subtle" onClick={() => apply({ user: "", from: "", to: "", patient: "" })}>
            Clear
          </Button>
        </Group>
      </form>
      {error && <Alert color="red">{error}</Alert>}
      {loading && !page && <Loader size="sm" aria-label="Loading the audit log" />}
      {page && (
        <Text size="sm" c="dimmed" aria-label="How many rows">
          {page.more
            ? `Showing the newest ${page.limit} rows. Narrow the filters, or export the CSV for all of them.`
            : `${page.rows.length} ${page.rows.length === 1 ? "row" : "rows"}`}
        </Text>
      )}
      {page && page.rows.length === 0 && <Text c="dimmed">Nothing matches these filters.</Text>}
      {page && page.rows.length > 0 && (
        <Table striped verticalSpacing="xs" aria-label="Audit log">
          <Table.Thead>
            <Table.Tr>
              <Table.Th>When</Table.Th>
              <Table.Th>Who</Table.Th>
              <Table.Th>What</Table.Th>
              <Table.Th>Patient</Table.Th>
              <Table.Th>Hospital</Table.Th>
              <Table.Th>Result</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {page.rows.map((r) => (
              <Table.Tr key={r.id}>
                <Table.Td style={{ whiteSpace: "nowrap" }}>{formatDateTime(r.at)}</Table.Td>
                <Table.Td>
                  {r.actor === "patient" ? (
                    <Text size="sm">Patient (their link)</Text>
                  ) : (
                    <>
                      <Text size="sm">{r.userName}</Text>
                      <Text size="xs" c="dimmed">{r.userEmail}</Text>
                    </>
                  )}
                </Table.Td>
                <Table.Td>
                  <Text size="sm">{r.actionLabel}</Text>
                </Table.Td>
                <Table.Td>
                  <Patients patients={r.patients} count={r.patientCount} />
                </Table.Td>
                <Table.Td>
                  <Text size="sm">{r.hospitalName || "Whole trust"}</Text>
                </Table.Td>
                <Table.Td>
                  <Result status={r.status} />
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}
    </Stack>
  );
}

/** One patient by name, or "12 patients" for a list screen, with the first ten names on hover. */
function Patients({ patients, count }: { patients: Row["patients"]; count: number }) {
  if (count === 0) return <Text size="sm" c="dimmed">—</Text>;
  const label = (p: Row["patients"][number]) => (p.hospitalNumber ? `${p.name} (${p.hospitalNumber})` : p.name);
  if (count === 1 && patients.length === 1) return <Text size="sm">{label(patients[0])}</Text>;
  return (
    <Tooltip label={namesOf(patients.map(label), count)} multiline w={320}>
      <Text size="sm" td="underline dotted" style={{ cursor: "help" }}>
        {count} patients
      </Text>
    </Tooltip>
  );
}

/** The request's outcome: the data was shown or changed, or it was refused (with the HTTP status). */
function Result({ status }: { status: number }) {
  if (status < 300) return <Badge color="green" variant="light">Done</Badge>;
  return (
    <Badge color={status >= 500 ? "red" : "orange"} variant="light">
      Refused ({status})
    </Badge>
  );
}
