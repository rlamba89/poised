"use client";
// The trust's staff and invites (STF-01, STF-02), for the trust's admins. An invite creates the
// person in Cognito, which emails them a temporary password; someone who already has an
// account only gains the new access.
import { useCallback, useEffect, useState } from "react";
import { Alert, Badge, Button, Card, Group, Select, Stack, Table, Text, TextInput, Title } from "@mantine/core";
import { api } from "@/lib/api";
import { useMe, type Role } from "@/lib/auth";

type StaffRow = {
  userId: string; name: string; email: string; hasAccount: boolean;
  hospitalId: string | null; hospitalName: string | null; role: Role;
};

const ROLE_LABELS: Record<Role, string> = { clinician: "Clinician", super_clinician: "Super clinician", admin: "Admin" };

export function StaffPage({ orgId, hospitalId }: { orgId: string; hospitalId: string }) {
  const me = useMe();
  const hospital = me.hospitals.find((h) => h.id === hospitalId);
  const base = `/o/${orgId}/staff`;
  const [rows, setRows] = useState<StaffRow[] | null>(null);
  const [error, setError] = useState("");

  const reload = useCallback(
    () => api<StaffRow[]>(base).then(setRows).catch((e: Error) => setError(e.message)),
    [base],
  );
  useEffect(() => {
    reload();
  }, [reload]);

  return (
    <Stack maw={960} mx="auto">
      <Title order={3}>Staff of {hospital?.orgName ?? "this trust"}</Title>
      {error && <Alert color="red">{error}</Alert>}
      {rows && (
        <>
          <Invite base={base} hospitalId={hospitalId} hospitalName={hospital?.name ?? "This hospital"} onInvited={reload} />
          <Table striped>
            <Table.Thead>
              <Table.Tr><Table.Th>Name</Table.Th><Table.Th>Email</Table.Th><Table.Th>Where</Table.Th><Table.Th>Role</Table.Th></Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {rows.map((r) => (
                <Table.Tr key={`${r.userId}-${r.hospitalId ?? "trust"}`}>
                  <Table.Td>{r.name} {!r.hasAccount && <Badge size="xs" variant="light" color="gray">no Cognito account</Badge>}</Table.Td>
                  <Table.Td>{r.email}</Table.Td>
                  <Table.Td>{r.hospitalName ?? "Whole trust"}</Table.Td>
                  <Table.Td>{ROLE_LABELS[r.role] ?? r.role}</Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </>
      )}
    </Stack>
  );
}

function Invite({ base, hospitalId, hospitalName, onInvited }: { base: string; hospitalId: string; hospitalName: string; onInvited: () => void }) {
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [role, setRole] = useState<Role>("clinician");
  const [scope, setScope] = useState<"hospital" | "trust">("hospital");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null);

  const invite = async (ev: React.FormEvent) => {
    ev.preventDefault();
    setBusy(true);
    setMessage(null);
    try {
      await api(`${base}/invites`, {
        method: "POST",
        body: JSON.stringify({ email, name, role, ...(scope === "hospital" ? { hospitalId } : {}) }),
      });
      setMessage({ ok: true, text: `Invited ${email}. If they are new, Cognito has emailed them a temporary password.` });
      setEmail("");
      setName("");
      onInvited();
    } catch (e) {
      setMessage({ ok: false, text: (e as Error).message });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card withBorder>
      <form onSubmit={invite}>
        <Text fw={600} mb="xs">Invite someone</Text>
        {message && <Alert color={message.ok ? "green" : "red"} mb="xs">{message.text}</Alert>}
        <Group align="flex-end" gap="sm">
          <TextInput label="Work email" type="email" required value={email} onChange={(e) => setEmail(e.currentTarget.value)} w={240} />
          <TextInput label="Name" required value={name} onChange={(e) => setName(e.currentTarget.value)} w={180} />
          <Select label="Role" value={role} allowDeselect={false} w={160}
            data={(Object.keys(ROLE_LABELS) as Role[]).map((r) => ({ value: r, label: ROLE_LABELS[r] }))}
            onChange={(v) => v && setRole(v as Role)} />
          <Select label="Where" value={scope} allowDeselect={false} w={180}
            data={[{ value: "hospital", label: hospitalName }, { value: "trust", label: "Whole trust" }]}
            onChange={(v) => v && setScope(v as "hospital" | "trust")} />
          <Button type="submit" loading={busy}>Invite</Button>
        </Group>
      </form>
    </Card>
  );
}
