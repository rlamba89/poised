// The platform admin's list of every trust (S10, ORG-01). No patient data (OPS-06).
import { useEffect, useState } from "react";
import { Link } from "react-router";
import { Alert, Anchor, Badge, Button, Group, Stack, Table, Text, Title } from "@mantine/core";
import { IconPlus } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { formatShortDate } from "@/lib/format";
import type { Status, TrustRow } from "./types";

export function StatusBadge({ status }: { status: Status }) {
  return status === "active" ? <Badge color="green" variant="light">Active</Badge> : <Badge color="red" variant="light">Suspended</Badge>;
}

export function TrustList() {
  const [trusts, setTrusts] = useState<TrustRow[] | null>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    api<TrustRow[]>("/admin/orgs").then(setTrusts).catch((e: Error) => setError(e.message));
  }, []);

  return (
    <Stack>
      <Group justify="space-between">
        <div>
          <Title order={3}>Trusts</Title>
          <Text size="sm" c="dimmed">Every trust using Preopsis in this country.</Text>
        </div>
        <Button component={Link} to="/admin/trusts/new" leftSection={<IconPlus size={16} />}>New trust</Button>
      </Group>
      {error && <Alert color="red">{error}</Alert>}
      {trusts && trusts.length === 0 && <Text c="dimmed">No trusts yet.</Text>}
      {trusts && trusts.length > 0 && (
        <Table.ScrollContainer minWidth={640}>
          <Table verticalSpacing="sm">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Name</Table.Th>
                <Table.Th>Code</Table.Th>
                <Table.Th>Hospitals</Table.Th>
                <Table.Th>Status</Table.Th>
                <Table.Th>Created</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {trusts.map((t) => (
                <Table.Tr key={t.id} aria-label={t.name}>
                  <Table.Td>
                    <Anchor component={Link} to={`/admin/trusts/${t.id}`} fw={500}>{t.name}</Anchor>
                  </Table.Td>
                  <Table.Td>{t.code}</Table.Td>
                  <Table.Td>{t.hospitalCount}</Table.Td>
                  <Table.Td><StatusBadge status={t.status} /></Table.Td>
                  <Table.Td>{formatShortDate(t.createdAt)}</Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      )}
    </Stack>
  );
}
