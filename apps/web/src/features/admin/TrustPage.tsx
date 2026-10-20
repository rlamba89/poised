// One trust, for the platform admin (S10): its details and admins, its hospitals (add, suspend,
// re-activate), and the opt-ins (ORG-05): the trust's default, and each hospital following it or
// set on its own. Suspending (ORG-03) stops sign-in and patients' links; nothing is deleted.
// No patient data here (OPS-06).
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useLocation } from "react-router";
import {
  Alert, Anchor, Badge, Button, Group, Image, Modal, Paper, Select, SimpleGrid, Stack, Switch, Table, Text, TextInput, Textarea, Title,
} from "@mantine/core";
import { IconPlus } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { deploymentTimeZone } from "@/lib/config";
import { formatShortDate } from "@/lib/format";
import { StatusBadge } from "./TrustList";
import type { AdminHospital, AdminTrust } from "./types";

/** What a confirmation or form dialog is open for. */
type Dialog =
  | { kind: "suspendTrust" }
  | { kind: "editTrust" }
  | { kind: "suspendHospital"; hospital: AdminHospital }
  | { kind: "addHospital" }
  | { kind: "editHospital"; hospital: AdminHospital }
  | { kind: "inviteAdmin" };

/**
 * Where a change's message is shown: by the part of the page it came from, so it's where the user
 * is looking (S10 test report 1, F2).
 */
type Place = "top" | "hospitals" | "optIns";
type Notice = { text: string; at: Place };

export function TrustPage({ orgId }: { orgId: string }) {
  const location = useLocation();
  const path = `/admin/orgs/${orgId}`;
  const [trust, setTrust] = useState<AdminTrust | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState<Notice | null>(() => {
    const text = (location.state as { notice?: string } | null)?.notice;
    return text ? { text, at: "top" } : null;
  });
  const [dialog, setDialog] = useState<Dialog | null>(null);
  // The last dialog stays rendered while the modal fades out; `opening` starts each form afresh.
  const [shown, setShown] = useState<Dialog | null>(null);
  const [opening, setOpening] = useState(0);

  const load = useCallback(() => {
    api<AdminTrust>(path).then(setTrust).catch((e: Error) => setError(e.message));
  }, [path]);
  useEffect(load, [load]);

  const open = (d: Dialog) => {
    setOpening((n) => n + 1);
    setDialog(d);
    setShown(d);
  };
  const close = () => setDialog(null);

  /**
   * Sends a change; on success shows the trust as it is now, with a message. A refusal is returned
   * to a form (which shows it and stays open), or else shown on the page.
   */
  const change: Change = async (method, url, body, message, inForm = false, at = "top") => {
    setError("");
    try {
      const t = await api<AdminTrust>(url, { method, body: JSON.stringify(body) });
      setTrust(t);
      setNotice({ text: typeof message === "string" ? message : message(t), at });
      setDialog(null);
      return "";
    } catch (e) {
      const msg = (e as Error).message;
      if (!inForm) {
        setDialog(null);
        setError(msg);
      }
      return msg;
    }
  };

  /** The change's message, if it belongs at this place. */
  const noticeAt = (at: Place) =>
    notice?.at === at && (
      <Alert color="green" withCloseButton onClose={() => setNotice(null)} role="status">{notice.text}</Alert>
    );

  if (!trust) return error ? <Alert color="red">{error}</Alert> : null;
  const hospitalPath = (h: AdminHospital) => `${path}/hospitals/${h.id}`;

  return (
    <Stack>
      <Anchor component={Link} to="/admin" size="sm">← Trusts</Anchor>
      <Group justify="space-between" align="flex-start">
        <div>
          <Group gap="sm">
            <Title order={3}>{trust.name}</Title>
            <StatusBadge status={trust.status} />
          </Group>
          <Text size="sm" c="dimmed">Code {trust.code} · created {formatShortDate(trust.createdAt)}</Text>
        </div>
        <Group gap="xs">
          <Button variant="default" onClick={() => open({ kind: "editTrust" })}>Edit details</Button>
          {trust.status === "active" ? (
            <Button color="red" variant="light" onClick={() => open({ kind: "suspendTrust" })}>Suspend trust</Button>
          ) : (
            <Button onClick={() => change("PATCH", path, { status: "active" }, `${trust.name} is active again. Its staff can sign in.`)}>
              Re-activate trust
            </Button>
          )}
        </Group>
      </Group>
      {trust.status === "suspended" && (
        <Alert color="red" title="This trust is suspended">Its staff can't sign in and patients can't open their links. Nothing has been deleted.</Alert>
      )}
      {noticeAt("top")}
      {error && <Alert color="red">{error}</Alert>}

      <SimpleGrid cols={{ base: 1, md: 2 }}>
        <Paper withBorder p="md">
          <Title order={5} mb="xs">Contact at the trust</Title>
          <Text size="sm">{trust.contactName}</Text>
          <Text size="sm">{trust.contactEmail}</Text>
          {trust.contactPhone && <Text size="sm">{trust.contactPhone}</Text>}
          {trust.logoUrl && (
            <Group mt="sm" gap="sm">
              <Image src={trust.logoUrl} alt={`${trust.name} logo`} h={40} w="auto" fit="contain" />
              <Text size="xs" c="dimmed" style={{ wordBreak: "break-all" }}>{trust.logoUrl}</Text>
            </Group>
          )}
        </Paper>
        <Paper withBorder p="md">
          <Group justify="space-between" mb="xs">
            <Title order={5}>Admins of the whole trust</Title>
            <Button size="compact-sm" variant="light" onClick={() => open({ kind: "inviteAdmin" })}>Invite an admin</Button>
          </Group>
          {trust.admins.length === 0 && <Text size="sm" c="dimmed">None. Invite one.</Text>}
          <Stack gap={4}>
            {trust.admins.map((a) => (
              <Group key={a.userId} justify="space-between" wrap="nowrap" aria-label={`Admin ${a.name}`}>
                <Text size="sm">{a.name} <Text span c="dimmed">{a.email}</Text></Text>
                {a.status === "active" ? (
                  <Badge variant="light" color="green">Active</Badge>
                ) : a.status === "invited" ? (
                  <Badge variant="light">Invited · expires {formatShortDate(a.inviteExpiresAt!)}</Badge>
                ) : (
                  <Badge variant="light" color="orange">Invite expired</Badge>
                )}
              </Group>
            ))}
          </Stack>
        </Paper>
      </SimpleGrid>

      <Group justify="space-between" mt="md">
        <Title order={4}>Hospitals</Title>
        <Button leftSection={<IconPlus size={16} />} onClick={() => open({ kind: "addHospital" })}>Add hospital</Button>
      </Group>
      {noticeAt("hospitals")}
      {trust.hospitals.length === 0 ? (
        <Text c="dimmed">No hospitals yet. Its staff can't work in Preopsis until it has one.</Text>
      ) : (
        <Table.ScrollContainer minWidth={900}>
          <Table verticalSpacing="sm">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Name</Table.Th>
                <Table.Th>Address</Table.Th>
                <Table.Th>For patients</Table.Th>
                <Table.Th>Time zone</Table.Th>
                <Table.Th>Status</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {trust.hospitals.map((h) => (
                <Table.Tr key={h.id} aria-label={h.name} style={{ verticalAlign: "top" }}>
                  <Table.Td fw={500}>{h.name}</Table.Td>
                  <Table.Td style={{ whiteSpace: "pre-line" }}>{h.address}</Table.Td>
                  <Table.Td>
                    <Text size="sm">{h.phone || <Text span c="dimmed">Trust's phone</Text>}</Text>
                    <Text size="sm">{h.email || <Text span c="dimmed">Trust's email</Text>}</Text>
                  </Table.Td>
                  <Table.Td>{h.timeZone}{!h.ownTimeZone && <Text span c="dimmed" size="sm"> (default)</Text>}</Table.Td>
                  <Table.Td><StatusBadge status={h.status} /></Table.Td>
                  <Table.Td>
                    <Group gap={4} wrap="nowrap">
                      <Button size="compact-sm" variant="subtle" aria-label={`Edit ${h.name}`} onClick={() => open({ kind: "editHospital", hospital: h })}>
                        Edit
                      </Button>
                      {h.status === "active" ? (
                        <Button size="compact-sm" variant="subtle" color="red" aria-label={`Suspend ${h.name}`}
                          onClick={() => open({ kind: "suspendHospital", hospital: h })}>
                          Suspend
                        </Button>
                      ) : (
                        <Button size="compact-sm" variant="subtle" aria-label={`Re-activate ${h.name}`}
                          onClick={() => change("PATCH", hospitalPath(h), { status: "active" }, `${h.name} is active again.`, false, "hospitals")}>
                          Re-activate
                        </Button>
                      )}
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      )}

      <OptInTable trust={trust} change={change} notice={noticeAt("optIns")} />

      <Modal opened={!!dialog} onClose={close} title={shown && titleOf(shown, trust)} size={shown?.kind === "addHospital" || shown?.kind === "editHospital" || shown?.kind === "editTrust" ? "lg" : "md"}>
        {shown?.kind === "suspendTrust" && (
          <Confirm onCancel={close} label="Suspend trust"
            onConfirm={() => change("PATCH", path, { status: "suspended" }, `${trust.name} is suspended. Its staff can't sign in.`)}>
            Its staff won't be able to sign in, and its patients won't be able to open their links, until you re-activate it.
            Nothing is deleted.
          </Confirm>
        )}
        {shown?.kind === "suspendHospital" && (
          <Confirm onCancel={close} label="Suspend hospital"
            onConfirm={() => change("PATCH", hospitalPath(shown.hospital), { status: "suspended" }, `${shown.hospital.name} is suspended.`, false, "hospitals")}>
            Nobody will be able to work at {shown.hospital.name} or open its patients' links until you re-activate it. Staff of the
            whole trust keep its other hospitals. Nothing is deleted.
          </Confirm>
        )}
        {shown?.kind === "addHospital" && (
          <HospitalForm key={opening} onCancel={close} submit="Add hospital"
            onSave={(body, name) => change("POST", `${path}/hospitals`, body, `${name} was added to ${trust.name}.`, true, "hospitals")} />
        )}
        {shown?.kind === "editHospital" && (
          <HospitalForm key={opening} onCancel={close} submit="Save" initial={shown.hospital}
            onSave={(body, name) => change("PATCH", hospitalPath(shown.hospital), body, `${name}'s details were saved.`, true, "hospitals")} />
        )}
        {shown?.kind === "editTrust" && (
          <TrustForm key={opening} trust={trust} onCancel={close}
            onSave={(body) => change("PATCH", path, body, (t) => `${t.name}'s details were saved.`, true)} />
        )}
        {shown?.kind === "inviteAdmin" && (
          <InviteAdminForm key={opening} onCancel={close}
            onSave={(body) => change("POST", `${path}/admins`, body, (t) => t.existingAccount
              ? `${body.email.trim()} already had an account, so they were made an admin of ${t.name}.`
              : `An invitation was sent to ${body.email.trim()}.`, true)} />
        )}
      </Modal>
    </Stack>
  );
}

function titleOf(d: Dialog, trust: AdminTrust): string {
  switch (d.kind) {
    case "suspendTrust": return `Suspend ${trust.name}?`;
    case "editTrust": return `Edit ${trust.name}`;
    case "editHospital": return `Edit ${d.hospital.name}`;
    case "suspendHospital": return `Suspend ${d.hospital.name}?`;
    case "addHospital": return `Add a hospital to ${trust.name}`;
    case "inviteAdmin": return `Invite an admin of ${trust.name}`;
  }
}

function Confirm({ children, label, onConfirm, onCancel }: { children: React.ReactNode; label: string; onConfirm: () => void; onCancel: () => void }) {
  return (
    <Stack>
      <Text size="sm">{children}</Text>
      <Group justify="flex-end">
        <Button variant="default" onClick={onCancel}>Cancel</Button>
        <Button color="red" onClick={onConfirm}>{label}</Button>
      </Group>
    </Stack>
  );
}

type Change = (
  method: string, url: string, body: unknown, message: string | ((t: AdminTrust) => string), inForm?: boolean, at?: Place,
) => Promise<string>;

/**
 * The opt-ins (ORG-05): one row per feature or instrument. The trust's switch is its default; each
 * hospital either follows it or is on or off on its own. Each change is saved at once.
 */
function OptInTable({ trust, change, notice }: { trust: AdminTrust; change: Change; notice: React.ReactNode }) {
  const path = `/admin/orgs/${trust.id}`;
  const onOff = (on: boolean) => (on ? "On" : "Off");
  return (
    <Stack gap="xs" mt="md">
      <Title order={4}>Features and opt-ins</Title>
      <Text size="sm" c="dimmed">The trust sets the default; each hospital follows it, or is switched on or off on its own.</Text>
      {notice}
      <Table.ScrollContainer minWidth={480 + 200 * trust.hospitals.length}>
        <Table verticalSpacing="xs">
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Feature</Table.Th>
              <Table.Th>Trust default</Table.Th>
              {trust.hospitals.map((h) => <Table.Th key={h.id}>{h.name}</Table.Th>)}
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {trust.optInDefs.map((d) => {
              const trustOn = !!trust.effectiveOptIns[d.key];
              return (
                <Table.Tr key={d.key} aria-label={d.label}>
                  <Table.Td>{d.label}</Table.Td>
                  <Table.Td>
                    <Switch aria-label={`${d.label}: trust default`} checked={trustOn} label={onOff(trustOn)}
                      onChange={(e) => change("PATCH", path, { optIns: { [d.key]: e.currentTarget.checked } },
                        `${d.label} is now ${onOff(e.currentTarget.checked).toLowerCase()} by default for ${trust.name}.`, false, "optIns")} />
                  </Table.Td>
                  {trust.hospitals.map((h) => {
                    const own = h.optIns[d.key];
                    return (
                      <Table.Td key={h.id}>
                        <Select aria-label={`${d.label} at ${h.name}`} allowDeselect={false} w={180} size="xs"
                          value={own === undefined ? "follow" : own ? "on" : "off"}
                          data={[
                            { value: "follow", label: `Follow trust (${onOff(trustOn)})` },
                            { value: "on", label: "On" },
                            { value: "off", label: "Off" },
                          ]}
                          onChange={(v) => {
                            const value = v === "follow" ? null : v === "on";
                            const what = v === "follow" ? "follows the trust" : `is ${v}`;
                            change("PATCH", `${path}/hospitals/${h.id}`, { optIns: { [d.key]: value } }, `${d.label} at ${h.name} ${what}.`, false, "optIns");
                          }} />
                      </Table.Td>
                    );
                  })}
                </Table.Tr>
              );
            })}
          </Table.Tbody>
        </Table>
      </Table.ScrollContainer>
    </Stack>
  );
}

/**
 * Add a hospital, or edit one's details at any time (ORG-02; S10 answers 3): name, address, the
 * patients' phone and email (empty: the trust's), time zone, logo address.
 */
function HospitalForm({ initial, submit, onSave, onCancel }: {
  initial?: AdminHospital; submit: string; onSave: (body: Record<string, string>, name: string) => Promise<string>; onCancel: () => void;
}) {
  const [form, setForm] = useState({
    name: initial?.name ?? "", address: initial?.address ?? "", phone: initial?.phone ?? "", email: initial?.email ?? "",
    timeZone: initial?.ownTimeZone ?? "", logoUrl: initial?.logoUrl ?? "",
  });
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  // Every IANA zone the browser knows; "" is the deployment's (A-20: none is written in the code).
  const zones = useMemo(() => {
    const deployment = deploymentTimeZone();
    const all = Intl.supportedValuesOf("timeZone");
    if (form.timeZone && !all.includes(form.timeZone)) all.push(form.timeZone); // one this browser doesn't list
    return [{ value: "", label: deployment ? `Default (${deployment})` : "Default" }, ...all.map((z) => ({ value: z, label: z }))];
  }, []); // once: the form's first zone is enough to add a missing one
  const set = (key: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setForm({ ...form, [key]: e.currentTarget.value });
  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError(await onSave(form, form.name.trim()));
    setSaving(false);
  };
  return (
    <form onSubmit={save} noValidate>
      <Stack>
        <TextInput label="Hospital name" withAsterisk value={form.name} onChange={set("name")} data-autofocus />
        <Textarea label="Address" withAsterisk autosize minRows={2} value={form.address} onChange={set("address")} />
        <Text size="sm" c="dimmed">Leave the phone and email empty to use the trust's.</Text>
        <SimpleGrid cols={{ base: 1, sm: 2 }}>
          <TextInput label="Phone for patients" value={form.phone} onChange={set("phone")} />
          <TextInput label="Email for patients" value={form.email} onChange={set("email")} />
        </SimpleGrid>
        <Select label="Time zone" searchable allowDeselect={false} data={zones} value={form.timeZone}
          onChange={(v) => setForm({ ...form, timeZone: v ?? "" })} />
        <TextInput label="Logo web address" description="Optional; leave empty to use the trust's." value={form.logoUrl} onChange={set("logoUrl")} />
        {error && <Alert color="red">{error}</Alert>}
        <Group justify="flex-end">
          <Button variant="default" onClick={onCancel}>Cancel</Button>
          <Button type="submit" loading={saving}>{submit}</Button>
        </Group>
      </Stack>
    </form>
  );
}

/** Edit a trust's details at any time (S10 answers 3): name, code, contact and logo address. */
function TrustForm({ trust, onSave, onCancel }: { trust: AdminTrust; onSave: (body: Record<string, string>) => Promise<string>; onCancel: () => void }) {
  const [form, setForm] = useState({
    name: trust.name, code: trust.code, contactName: trust.contactName, contactEmail: trust.contactEmail,
    contactPhone: trust.contactPhone, logoUrl: trust.logoUrl,
  });
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const set = (key: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: e.currentTarget.value });
  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError(await onSave(form));
    setSaving(false);
  };
  return (
    <form onSubmit={save} noValidate>
      <Stack>
        <TextInput label="Trust name" withAsterisk value={form.name} onChange={set("name")} data-autofocus />
        <TextInput label="Code" withAsterisk description="2 to 12 letters, digits or hyphens" value={form.code} onChange={set("code")} />
        <TextInput label="Logo web address" description="Optional" value={form.logoUrl} onChange={set("logoUrl")} />
        <SimpleGrid cols={{ base: 1, sm: 3 }}>
          <TextInput label="Contact name" withAsterisk value={form.contactName} onChange={set("contactName")} />
          <TextInput label="Contact email" withAsterisk value={form.contactEmail} onChange={set("contactEmail")} />
          <TextInput label="Contact phone" value={form.contactPhone} onChange={set("contactPhone")} />
        </SimpleGrid>
        {error && <Alert color="red">{error}</Alert>}
        <Group justify="flex-end">
          <Button variant="default" onClick={onCancel}>Cancel</Button>
          <Button type="submit" loading={saving}>Save</Button>
        </Group>
      </Stack>
    </form>
  );
}

/** Invite an admin of the whole trust (S09's invite): e.g. when the first admin's invitation expired. */
function InviteAdminForm({ onSave, onCancel }: { onSave: (body: { email: string; name: string }) => Promise<string>; onCancel: () => void }) {
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError(await onSave({ email, name }));
    setSaving(false);
  };
  return (
    <form onSubmit={submit} noValidate>
      <Stack>
        <TextInput label="Email" withAsterisk value={email} onChange={(e) => setEmail(e.currentTarget.value)} data-autofocus />
        <TextInput label="Name" description="Optional" value={name} onChange={(e) => setName(e.currentTarget.value)} />
        {error && <Alert color="red">{error}</Alert>}
        <Group justify="flex-end">
          <Button variant="default" onClick={onCancel}>Cancel</Button>
          <Button type="submit" loading={saving}>Send invite</Button>
        </Group>
      </Stack>
    </form>
  );
}
