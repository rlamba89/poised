// Step 2 of plan-workflow.md: the hospital's episodes, the clinician home. "New episode" opens its
// own page (S13, NewEpisodePage). Trust-wide staff can also list every hospital's episodes ("All
// hospitals", STF-03), kept in the URL as ?hospitals=all.
//
// The worklists (S21, EPI-02) are tabs beside "All": built-in saved filters, each with its count.
// Their labels, help lines and counts come from Go (GET …/worklists), and each one's list is
// GET …/episodes?worklist=<key>, so the rules are Go's (the database's in_worklist). The tab is kept
// in the URL as ?worklist=<key>, so a reload, a deep link and Back keep it. The status filter
// belongs to "All".
//
// Archived episodes (S22) are in no worklist, and "All" leaves them out unless "Show archived" is
// ticked (?archived=1 in the URL) or the status filter asks for Archived.
import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { Alert, Badge, Button, Checkbox, Group, SegmentedControl, Select, Stack, Table, Tabs, Text, Title } from "@mantine/core";
import { IconPlus } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { hospitalBase, useHospital } from "@/lib/auth";
import { formatDateTime, formatOptionalDate } from "@/lib/format";
import { openTaskCount } from "./tasks";
import { patientName, STATUS_COLORS, STATUS_LABELS, type EpisodeRow, type EpisodeStatus } from "./types";

/** openTasks: how many of its tasks are open (S21). */
type ListRow = EpisodeRow & { hospitalId: string; hospitalName: string; openTasks: number };

/** One worklist tab (Go: worklistCount in internal/httpapi/worklists.go). */
type WorklistTab = { key: string; label: string; help: string; count: number };

export function EpisodeList({ base }: { base: string }) {
  const navigate = useNavigate();
  const { trust, hospital } = useHospital();
  const [params, setParams] = useSearchParams();
  // "All hospitals" is for trust-wide staff only; anyone else gets this hospital's list.
  const trustWide = trust.role !== null;
  const all = trustWide && params.get("hospitals") === "all";
  const worklist = params.get("worklist") ?? "";
  const showArchived = params.get("archived") === "1";
  const [status, setStatus] = useState<string | null>(null);
  const [rows, setRows] = useState<ListRow[] | null>(null);
  const [tabs, setTabs] = useState<WorklistTab[] | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    const from = all ? `/o/${trust.id}` : base;
    const query: Record<string, string> = worklist ? { worklist } : { status: status ?? "", ...(showArchived ? { archived: "true" } : {}) };
    api<ListRow[]>(`${from}/episodes?${new URLSearchParams(query)}`)
      .then((r) => {
        setRows(r);
        setError("");
      })
      .catch((e: Error) => setError(e.message));
    // The counts with every list, so they match what the lists hold now.
    api<WorklistTab[]>(`${from}/worklists`).then(setTabs).catch((e: Error) => setError(e.message));
  }, [all, trust.id, base, status, worklist, showArchived]);
  useEffect(load, [load]);

  // Changes one URL parameter, keeping the others ("" removes it).
  const setParam = (name: string, value: string) => {
    const next = new URLSearchParams(params);
    if (value) next.set(name, value);
    else next.delete(name);
    setParams(next);
  };
  const tab = tabs?.find((t) => t.key === worklist);

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
          onChange={(v) => setParam("hospitals", v === "all" ? "all" : "")}
        />
      )}
      <Tabs value={worklist || "all"} onChange={(v) => setParam("worklist", v && v !== "all" ? v : "")} keepMounted={false}>
        <Tabs.List aria-label="Worklists">
          <Tabs.Tab value="all" data-testid="worklist-all">All</Tabs.Tab>
          {tabs?.map((t) => (
            <Tabs.Tab key={t.key} value={t.key} data-testid={`worklist-${t.key}`} data-count={t.count} title={t.help}
              rightSection={<Badge size="sm" variant={t.count > 0 ? "filled" : "light"} color={t.count > 0 ? "brand" : "gray"} circle={t.count < 10}>{t.count}</Badge>}>
              {t.label}
            </Tabs.Tab>
          ))}
        </Tabs.List>
      </Tabs>
      {worklist ? (
        <Text size="sm" c="dimmed" data-testid="worklist-help">{tab?.help}</Text>
      ) : (
        <Group gap="md" align="center">
          <Select
            aria-label="Status"
            placeholder="All statuses"
            clearable
            data={Object.entries(STATUS_LABELS).map(([value, label]) => ({ value, label }))}
            value={status}
            onChange={setStatus}
            maw={260}
          />
          <Checkbox label="Show archived" checked={showArchived} onChange={(ev) => setParam("archived", ev.currentTarget.checked ? "1" : "")} />
        </Group>
      )}
      {error && <Alert color="red">{error}</Alert>}
      {rows && rows.length === 0 && <Text c="dimmed">{worklist ? "No episodes in this worklist." : "No episodes yet."}</Text>}
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
                  {!!openTaskCount(e.status, e.openTasks) && (
                    <Text size="xs" c="dimmed" mt={2}>{e.openTasks === 1 ? "1 open task" : `${e.openTasks} open tasks`}</Text>
                  )}
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
