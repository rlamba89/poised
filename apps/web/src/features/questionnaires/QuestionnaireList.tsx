"use client";
// FRM-01 list, FRM-04 create, FRM-06 delete. Laid out like the Lifebox Author list.
import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useDebouncedValue } from "@mantine/hooks";
import {
  ActionIcon, Alert, Badge, Button, Drawer, Group, Menu, Modal, Pagination, Stack, Table, Text, Textarea, TextInput, Title,
} from "@mantine/core";
import { IconDots, IconPencil, IconPlus, IconSearch, IconTrash } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { atLeast, useMe } from "@/lib/auth";
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

export function QuestionnaireList({ hospitalId, base }: { hospitalId: string; base: string }) {
  const me = useMe();
  const router = useRouter();
  const [search, setSearch] = useState("");
  const [debounced] = useDebouncedValue(search, 250);
  const [page, setPage] = useState(1);
  const [data, setData] = useState<ListResponse | null>(null);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<Row | null>(null);
  const canAuthor = atLeast(me, hospitalId, "super_clinician");
  const isAdmin = atLeast(me, hospitalId, "admin");

  const load = useCallback(() => {
    const params = new URLSearchParams({ q: debounced, page: String(page) });
    api<ListResponse>(`${base}/questionnaires?${params}`)
      .then((d) => {
        setData(d);
        setError("");
      })
      .catch((e: Error) => setError(e.message));
  }, [base, debounced, page]);

  useEffect(load, [load]);
  useEffect(() => setPage(1), [debounced]);

  return (
    <Stack>
      <Group justify="space-between">
        <Title order={3}>Questionnaires</Title>
        {canAuthor && <Button leftSection={<IconPlus size={16} />} onClick={() => setCreating(true)}>New Questionnaire</Button>}
      </Group>
      <TextInput
        leftSection={<IconSearch size={16} />}
        placeholder="Search by name or description"
        value={search}
        onChange={(e) => setSearch(e.currentTarget.value)}
        maw={400}
      />
      {error && <Alert color="red">{error}</Alert>}
      {data && data.items.length === 0 && <Text c="dimmed">No questionnaires found.</Text>}
      {data && data.items.length > 0 && (
        <Table highlightOnHover verticalSpacing="sm">
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Name</Table.Th>
              <Table.Th>Version</Table.Th>
              <Table.Th>Last Modified</Table.Th>
              <Table.Th>State</Table.Th>
              <Table.Th>Created By</Table.Th>
              <Table.Th w={50} />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {data.items.map((q) => {
              const open = () => router.push(`${base}/questionnaires/${q.id}`);
              const canDelete = q.status === "draft" && (q.createdBy === me.user.id || isAdmin);
              return (
                <Table.Tr key={q.id} onClick={open} style={{ cursor: "pointer" }}>
                  <Table.Td>
                    <Text fw={700}>{q.name}</Text>
                    {q.description && <Text size="sm" c="dimmed">{q.description}</Text>}
                  </Table.Td>
                  <Table.Td>{q.versionNo}</Table.Td>
                  <Table.Td>{formatDateTime(q.updatedAt)} by {q.updatedByName}</Table.Td>
                  <Table.Td>
                    <Badge variant="light" color={q.status === "draft" ? "gray" : "green"}>{q.status}</Badge>
                  </Table.Td>
                  <Table.Td>{q.createdByName}</Table.Td>
                  <Table.Td onClick={(e) => e.stopPropagation()}>
                    <Menu position="bottom-end">
                      <Menu.Target>
                        <ActionIcon variant="subtle" color="gray" aria-label={`${q.name} options`}>
                          <IconDots size={16} />
                        </ActionIcon>
                      </Menu.Target>
                      <Menu.Dropdown>
                        <Menu.Item leftSection={<IconPencil size={14} />} onClick={open}>
                          {q.status === "draft" && canAuthor ? "Edit" : "View"}
                        </Menu.Item>
                        {canDelete && (
                          <Menu.Item color="red" leftSection={<IconTrash size={14} />} onClick={() => setDeleting(q)}>
                            Delete
                          </Menu.Item>
                        )}
                      </Menu.Dropdown>
                    </Menu>
                  </Table.Td>
                </Table.Tr>
              );
            })}
          </Table.Tbody>
        </Table>
      )}
      {data && data.total > data.pageSize && (
        <Pagination total={Math.ceil(data.total / data.pageSize)} value={page} onChange={setPage} />
      )}

      <CreateModal
        opened={creating}
        onClose={() => setCreating(false)}
        base={base}
        onCreated={(id) => router.push(`${base}/questionnaires/${id}`)}
      />
      <DeleteModal
        row={deleting}
        base={base}
        onClose={() => setDeleting(null)}
        onDeleted={() => {
          setDeleting(null);
          load();
        }}
      />
    </Stack>
  );
}

function CreateModal(props: { opened: boolean; onClose: () => void; base: string; onCreated: (id: string) => void }) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      const res = await api<{ id: string }>(`${props.base}/questionnaires`, {
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
    <Drawer opened={props.opened} onClose={props.onClose} position="right" title="New Questionnaire">
      <form onSubmit={submit}>
        <Stack>
          {error && <Alert color="red">{error}</Alert>}
          <TextInput label="Name" required value={name} onChange={(e) => setName(e.currentTarget.value)} data-autofocus />
          <Textarea label="Description" required value={description} onChange={(e) => setDescription(e.currentTarget.value)} />
          <Group justify="flex-end">
            <Button variant="default" onClick={props.onClose}>Cancel</Button>
            <Button type="submit" loading={saving}>Save</Button>
          </Group>
        </Stack>
      </form>
    </Drawer>
  );
}

function DeleteModal(props: { row: Row | null; base: string; onClose: () => void; onDeleted: () => void }) {
  const [error, setError] = useState("");
  const confirm = async () => {
    if (!props.row) return;
    try {
      await api(`${props.base}/questionnaires/${props.row.id}`, { method: "DELETE" });
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
          “{props.row?.name}” and all its Question Sets will be deleted. This can&apos;t be undone.
        </Text>
        <Group justify="flex-end">
          <Button variant="default" onClick={props.onClose}>Cancel</Button>
          <Button color="red" onClick={confirm}>Delete</Button>
        </Group>
      </Stack>
    </Modal>
  );
}
