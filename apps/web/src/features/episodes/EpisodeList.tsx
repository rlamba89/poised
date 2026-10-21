// Step 2 of plan-workflow.md: the hospital's episodes. "New episode" opens its own page (S13,
// NewEpisodePage). Trust-wide staff can also list every hospital's episodes ("All hospitals",
// STF-03), kept in the URL as ?hospitals=all.
import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { Alert, Badge, Button, Group, SegmentedControl, Select, Stack, Table, Text, Title } from "@mantine/core";
import { IconPlus } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { hospitalBase, useHospital } from "@/lib/auth";
import { formatDateTime, formatOptionalDate } from "@/lib/format";
import { patientName, STATUS_COLORS, STATUS_LABELS, type EpisodeRow, type EpisodeStatus } from "./types";

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
        <Button leftSection={<IconPlus size={16} />} component={Link} to={`${base}/episodes/new`}>New episode</Button>
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
              <Table.Th>Planned date</Table.Th>
              <Table.Th>POA date</Table.Th>
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
                <Table.Td>{formatOptionalDate(e.plannedDate)}</Table.Td>
                <Table.Td>{formatOptionalDate(e.poaDate)}</Table.Td>
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
    </Stack>
  );
}

export function StatusBadge({ status }: { status: EpisodeStatus }) {
  return <Badge variant="light" color={STATUS_COLORS[status]}>{STATUS_LABELS[status]}</Badge>;
}
