"use client";
// FRM-01 list, FRM-04 create, FRM-06 delete.
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useDebouncedValue } from "@mantine/hooks";
import {
  Alert, Anchor, Badge, Button, Group, Modal, Pagination, Stack, Table, Text, Textarea, TextInput, Title,
} from "@mantine/core";
import { api } from "@/lib/api";
import { hasRole, useMe } from "@/lib/auth";
import { formatDateTime } from "@/lib/format";

type Row = {
  id: string;
  name: string;
  description: string;
  createdBy: string;
  createdAt: string;
  createdByName: string;
  versionNo: number;
  status: string;
  updatedAt: string;
  updatedByName: string;
};
type ListResponse = { items: Row[]; total: number; page: number; pageSize: number };

export function QuestionnaireList({ hospitalId }: { hospitalId: string }) {
  const me = useMe();
  const router = useRouter();
  const [search, setSearch] = useState("");
  const [debounced] = useDebouncedValue(search, 250);
  const [page, setPage] = useState(1);
  const [data, setData] = useState<ListResponse | null>(null);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<Row | null>(null);
  const isAuthor = hasRole(me, hospitalId, "author");
  const isAdmin = hasRole(me, hospitalId, "hospital_admin");

  const load = useCallback(() => {
    const params = new URLSearchParams({ q: debounced, page: String(page) });
    api<ListResponse>(`/h/${hospitalId}/questionnaires?${params}`)
      .then((d) => {
        setData(d);
        setError("");
      })
      .catch((e: Error) => setError(e.message));
  }, [hospitalId, debounced, page]);

  useEffect(load, [load]);
  useEffect(() => setPage(1), [debounced]);

  return (
    <Stack>
      <Group justify="space-between">
        <Title order={3}>Questionnaires</Title>
        {isAuthor && <Button onClick={() => setCreating(true)}>New questionnaire</Button>}
      </Group>
      <TextInput
        placeholder="Search by name or description"
        value={search}
        onChange={(e) => setSearch(e.currentTarget.value)}
        maw={400}
      />
      {error && <Alert color="red">{error}</Alert>}
      {data && data.items.length === 0 && <Text c="dimmed">No questionnaires found.</Text>}
      {data && data.items.length > 0 && (
        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Name</Table.Th>
              <Table.Th>Description</Table.Th>
              <Table.Th>Version</Table.Th>
              <Table.Th>Status</Table.Th>
              <Table.Th>Last changed</Table.Th>
              <Table.Th>Created by</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {data.items.map((q) => (
              <Table.Tr key={q.id}>
                <Table.Td>
                  <Anchor component={Link} href={`/h/${hospitalId}/questionnaires/${q.id}`}>{q.name}</Anchor>
                </Table.Td>
                <Table.Td>{q.description}</Table.Td>
                <Table.Td>{q.versionNo}</Table.Td>
                <Table.Td><Badge variant="light">{q.status}</Badge></Table.Td>
                <Table.Td>{formatDateTime(q.updatedAt)} by {q.updatedByName}</Table.Td>
                <Table.Td>{q.createdByName}</Table.Td>
                <Table.Td>
                  {q.status === "draft" && (q.createdBy === me.user.id || isAdmin) && (
                    <Button size="xs" variant="subtle" color="red" onClick={() => setDeleting(q)}>Delete</Button>
                  )}
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}
      {data && data.total > data.pageSize && (
        <Pagination total={Math.ceil(data.total / data.pageSize)} value={page} onChange={setPage} />
      )}

      <CreateModal
        opened={creating}
        onClose={() => setCreating(false)}
        hospitalId={hospitalId}
        onCreated={(id) => router.push(`/h/${hospitalId}/questionnaires/${id}`)}
      />
      <DeleteModal
        row={deleting}
        hospitalId={hospitalId}
        onClose={() => setDeleting(null)}
        onDeleted={() => {
          setDeleting(null);
          load();
        }}
      />
    </Stack>
  );
}

function CreateModal(props: { opened: boolean; onClose: () => void; hospitalId: string; onCreated: (id: string) => void }) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      const res = await api<{ id: string }>(`/h/${props.hospitalId}/questionnaires`, {
        method: "POST",
        body: JSON.stringify({ name, description }),
      });
      props.onCreated(res.id);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal opened={props.opened} onClose={props.onClose} title="New questionnaire">
      <form onSubmit={submit}>
        <Stack>
          {error && <Alert color="red">{error}</Alert>}
          <TextInput label="Name" required value={name} onChange={(e) => setName(e.currentTarget.value)} data-autofocus />
          <Textarea label="Description" required value={description} onChange={(e) => setDescription(e.currentTarget.value)} />
          <Group justify="flex-end">
            <Button variant="default" onClick={props.onClose}>Cancel</Button>
            <Button type="submit" loading={saving}>Create</Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  );
}

function DeleteModal(props: { row: Row | null; hospitalId: string; onClose: () => void; onDeleted: () => void }) {
  const [error, setError] = useState("");
  const confirm = async () => {
    if (!props.row) return;
    try {
      await api(`/h/${props.hospitalId}/questionnaires/${props.row.id}`, { method: "DELETE" });
      setError("");
      props.onDeleted();
    } catch (err) {
      setError((err as Error).message);
    }
  };
  return (
    <Modal opened={!!props.row} onClose={props.onClose} title="Delete questionnaire?">
      <Stack>
        {error && <Alert color="red">{error}</Alert>}
        <Text>
          “{props.row?.name}” and all its chapters will be deleted. This can&apos;t be undone.
        </Text>
        <Group justify="flex-end">
          <Button variant="default" onClick={props.onClose}>Cancel</Button>
          <Button color="red" onClick={confirm}>Delete</Button>
        </Group>
      </Stack>
    </Modal>
  );
}
