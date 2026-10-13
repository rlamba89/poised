// A patient's page (plan C2, S13): the trust's record (S08 answers: a hospital's staff see the
// patient's details) and their episodes at this hospital only. A patient with no episode here
// is "not found", even if another hospital of the trust has seen them.
import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router";
import { Alert, Anchor, Button, Card, Center, Group, Loader, SimpleGrid, Stack, Table, Text, Title } from "@mantine/core";
import { IconArrowLeft, IconPlus } from "@tabler/icons-react";
import { ageFrom } from "@poised/clinical";
import { api } from "@/lib/api";
import { useHospital } from "@/lib/auth";
import { nationalIdLabel } from "@/lib/config";
import { formatDate, formatDateTime } from "@/lib/format";
import { StatusBadge } from "./EpisodeList";
import { patientName, SEXES, type EpisodeStatus, type PatientRecord } from "./types";

type EpisodeHere = {
  id: string;
  status: EpisodeStatus;
  procedure: string;
  hospitalNumber: string;
  createdAt: string;
  questionnaireName: string;
  versionNo: number;
};

export function PatientPage({ base, patientId }: { base: string; patientId: string }) {
  const navigate = useNavigate();
  const { hospital } = useHospital();
  const [data, setData] = useState<{ patient: PatientRecord; episodes: EpisodeHere[] } | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ patient: PatientRecord; episodes: EpisodeHere[] }>(`${base}/patients/${patientId}`)
      .then(setData)
      .catch((e: Error) => setError(e.message));
  }, [base, patientId]);

  const back = (
    <Anchor component={Link} to={`${base}/episodes`} size="sm">
      <Group gap={4}><IconArrowLeft size={14} /> Episodes</Group>
    </Anchor>
  );
  if (!data) {
    if (!error) return <Center h="40vh"><Loader /></Center>;
    return (
      <Stack maw={960} mx="auto">
        {back}
        <Alert color="red">{error}</Alert>
      </Stack>
    );
  }
  const p = data.patient;

  return (
    <Stack maw={960} mx="auto">
      {back}
      <Title order={3}>{patientName(p)}</Title>
      <Card withBorder>
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="xs">
          <Detail label={nationalIdLabel()} value={p.nationalId ?? ""} />
          <Detail label="Date of birth" value={`${formatDate(p.dateOfBirth)} (age ${ageFrom(p.dateOfBirth)})`} />
          <Detail label="Sex" value={SEXES.find((s) => s.value === p.sex)?.label ?? p.sex} />
          <Detail label="Mobile" value={p.mobile} />
          <Detail label="Email" value={p.email} />
        </SimpleGrid>
      </Card>

      <Group justify="space-between">
        <Title order={4}>Episodes at {hospital.name}</Title>
        <Button leftSection={<IconPlus size={16} />} component={Link} to={`${base}/episodes/new?patient=${p.id}`}>
          New episode
        </Button>
      </Group>
      <Table highlightOnHover verticalSpacing="sm" aria-label="Episodes">
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Created</Table.Th>
            <Table.Th>HQ</Table.Th>
            <Table.Th>Procedure</Table.Th>
            <Table.Th>Hospital number</Table.Th>
            <Table.Th>Status</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {data.episodes.map((e) => (
            <Table.Tr key={e.id} onClick={() => navigate(`${base}/episodes/${e.id}`)} style={{ cursor: "pointer" }}>
              <Table.Td>{formatDateTime(e.createdAt)}</Table.Td>
              <Table.Td>{e.questionnaireName} v{e.versionNo}</Table.Td>
              <Table.Td>{e.procedure || "—"}</Table.Td>
              <Table.Td>{e.hospitalNumber || "—"}</Table.Td>
              <Table.Td><StatusBadge status={e.status} /></Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </Stack>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <Group gap="xs" wrap="nowrap" align="baseline">
      <Text size="sm" fw={600} w={170}>{label}</Text>
      <Text size="sm">{value || "—"}</Text>
    </Group>
  );
}
