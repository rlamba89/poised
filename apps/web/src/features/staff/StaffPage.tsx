// The trust's staff (plan C1, S09; STF-01, STF-02, STF-04): who has access, invite someone, give
// someone more access, change or remove one of their accesses, or remove them from the trust.
// Only the trust's whole-trust admins see it; the API (/api/o/{trust}/staff…) refuses everyone else.
//
// A person can hold several accesses ("memberships") in one trust: the whole trust and/or some of
// its hospitals, each with its own role (Rahul, 13 Oct). The list has one row per person.
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { Alert, Badge, Button, Group, Modal, Select, Stack, Table, Text, TextInput, Title } from "@mantine/core";
import { IconPlus } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { canManageStaff, roleLabel, useHospital, useMe, type Role, type Trust } from "@/lib/auth";
import { formatShortDate } from "@/lib/format";

/** One access: the whole trust (hospitalId null) or one hospital, with a role. */
type Access = { hospitalId: string | null; hospitalName: string | null; role: Role };

/** One person, as GET /api/o/{trust}/staff lists them. */
type Person = {
  userId: string;
  name: string;
  email: string;
  access: Access[]; // the whole trust first, then by hospital name
  /** "invited" or "expired" while their account waits for its first sign-in. */
  status: "active" | "invited" | "expired";
  inviteExpiresAt: string | null;
};

/** The access pickers' value: a scope ("trust" or a hospital id) and a role. */
type Choice = { scope: string; role: Role };

const WHOLE_TRUST = "trust";
const roles: Role[] = ["clinician", "super_clinician", "admin"];

/** An access's scope as the API names it in URLs: "trust" or the hospital id. */
const scopeOf = (a: Access) => a.hospitalId ?? WHOLE_TRUST;
/** "Hospital A1", or "the whole trust". */
const placeOf = (a: Access) => a.hospitalName ?? "the whole trust";
/** The request body for a choice: hospitalId null means the whole trust. */
const choiceBody = (c: Choice) => ({ hospitalId: c.scope === WHOLE_TRUST ? null : c.scope, role: c.role });
/** The name of a choice's scope, for messages. */
const placeName = (trust: Trust, scope: string) =>
  scope === WHOLE_TRUST ? "the whole trust" : (trust.hospitals.find((h) => h.id === scope)?.name ?? "that hospital");

/** What a dialog is open for. */
type Dialog =
  | { kind: "invite" }
  | { kind: "add"; person: Person }
  | { kind: "change"; person: Person; access: Access }
  | { kind: "removeAccess"; person: Person; access: Access }
  | { kind: "removePerson"; person: Person };

export function StaffPage() {
  const me = useMe();
  const { trust } = useHospital();
  const navigate = useNavigate();
  const path = `/o/${trust.id}/staff`;
  const allowed = canManageStaff(trust);
  const [people, setPeople] = useState<Person[] | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [dialog, setDialog] = useState<Dialog | null>(null);
  // The last dialog stays rendered while the modal fades out, instead of going blank.
  const [shown, setShown] = useState<Dialog | null>(null);
  // Bumped each time a dialog opens, so it starts afresh rather than with the last one's entries
  // (S09 test report 1, F-1).
  const [opening, setOpening] = useState(0);

  const load = useCallback(() => {
    api<Person[]>(path)
      .then((r) => {
        setPeople(r);
        setError("");
      })
      .catch((e: Error) => setError(e.message));
  }, [path]);
  useEffect(() => {
    if (allowed) load();
  }, [allowed, load]);

  const open = (d: Dialog) => {
    setOpening((n) => n + 1);
    setDialog(d);
    setShown(d);
  };
  const close = () => setDialog(null);
  // After a change: show what happened and reload. A change to your own access may end your right
  // to manage staff, so it goes back to the start.
  const done = (who: Person | null, message: string) => {
    setDialog(null);
    if (who?.userId === me.user.id) {
      navigate("/");
      return;
    }
    setNotice(message);
    load();
  };

  const resend = async (p: Person) => {
    setError("");
    try {
      const r = await api<Person>(`${path}/invites`, {
        method: "POST",
        body: JSON.stringify({ email: p.email, hospitalId: p.access[0].hospitalId, role: p.access[0].role }),
      });
      done(null, `Invitation sent again to ${r.email}. It expires on ${formatShortDate(r.inviteExpiresAt!)}.`);
    } catch (e) {
      setError((e as Error).message);
    }
  };

  if (!allowed) {
    return <Alert color="red" title="Only the trust's admins can manage staff" />;
  }
  return (
    <Stack>
      <Group justify="space-between">
        <div>
          <Title order={3}>Staff</Title>
          <Text size="sm" c="dimmed">Everyone with access to {trust.name}.</Text>
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={() => open({ kind: "invite" })}>Invite</Button>
      </Group>
      {notice && (
        <Alert color="green" withCloseButton onClose={() => setNotice("")} role="status">
          {notice}
        </Alert>
      )}
      {error && <Alert color="red">{error}</Alert>}
      {people && (
        <Table.ScrollContainer minWidth={960}>
          <Table verticalSpacing="sm">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Name</Table.Th>
                <Table.Th>Email</Table.Th>
                <Table.Th>Access and role</Table.Th>
                <Table.Th>Status</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {people.map((p) => (
                <Table.Tr key={p.userId} aria-label={p.name} style={{ verticalAlign: "top" }}>
                  <Table.Td>
                    <Text fw={500}>
                      {p.name}
                      {p.userId === me.user.id && <Text span c="dimmed"> (you)</Text>}
                    </Text>
                  </Table.Td>
                  <Table.Td>{p.email}</Table.Td>
                  <Table.Td>
                    <Stack gap={4}>
                      {p.access.map((a) => (
                        <Group key={scopeOf(a)} gap="xs" wrap="nowrap" justify="space-between">
                          <Text size="sm" style={{ whiteSpace: "nowrap" }}>{a.hospitalName ?? "Whole trust"} · {roleLabel[a.role]}</Text>
                          <Group gap={4} wrap="nowrap">
                            <Button size="compact-xs" variant="default" aria-label={`Change ${p.name}'s access at ${placeOf(a)}`}
                              onClick={() => open({ kind: "change", person: p, access: a })}>
                              Change
                            </Button>
                            <Button size="compact-xs" variant="subtle" color="red" aria-label={`Remove ${p.name}'s access at ${placeOf(a)}`}
                              onClick={() => open({ kind: "removeAccess", person: p, access: a })}>
                              Remove
                            </Button>
                          </Group>
                        </Group>
                      ))}
                    </Stack>
                  </Table.Td>
                  <Table.Td><StatusBadge person={p} /></Table.Td>
                  <Table.Td>
                    <Stack gap={4} align="flex-end">
                      <Button size="compact-xs" variant="subtle" aria-label={`Give ${p.name} more access`} onClick={() => open({ kind: "add", person: p })}>
                        Add access
                      </Button>
                      {p.status !== "active" && (
                        <Button size="compact-xs" variant="subtle" onClick={() => resend(p)} aria-label={`Send ${p.name} a new invitation`}>
                          Resend invite
                        </Button>
                      )}
                      <Button size="compact-xs" variant="subtle" color="red" aria-label={`Remove ${p.name} from ${trust.name}`}
                        onClick={() => open({ kind: "removePerson", person: p })}>
                        Remove from trust
                      </Button>
                    </Stack>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      )}

      <Modal opened={!!dialog} onClose={close} title={shown && titleOf(shown, trust)}>
        {shown?.kind === "invite" && <InviteForm key={opening} trust={trust} path={path} onCancel={close} onDone={done} />}
        {shown?.kind === "add" && <InviteForm key={opening} trust={trust} path={path} person={shown.person} onCancel={close} onDone={done} />}
        {shown?.kind === "change" && (
          <ChangeForm key={opening} trust={trust} path={path} person={shown.person} access={shown.access} onCancel={close} onDone={done} />
        )}
        {(shown?.kind === "removeAccess" || shown?.kind === "removePerson") && (
          <RemoveForm key={opening} trust={trust} path={path} person={shown.person}
            access={shown.kind === "removeAccess" ? shown.access : null} onCancel={close} onDone={done} />
        )}
      </Modal>
    </Stack>
  );
}

function titleOf(d: Dialog, trust: Trust): string {
  switch (d.kind) {
    case "invite":
      return `Invite someone to ${trust.name}`;
    case "add":
      return `Give ${d.person.name} more access`;
    case "change":
      return `Change ${d.person.name}'s access at ${placeOf(d.access)}`;
    case "removeAccess":
      return `Remove ${d.person.name}'s access at ${placeOf(d.access)}?`;
    case "removePerson":
      return `Remove ${d.person.name} from ${trust.name}?`;
  }
}

function StatusBadge({ person }: { person: Person }) {
  if (person.status === "invited") {
    return (
      <Badge variant="light" color="blue" style={{ overflow: "visible" }} styles={{ label: { overflow: "visible" } }}>
        Invited · expires {formatShortDate(person.inviteExpiresAt!)}
      </Badge>
    );
  }
  if (person.status === "expired") return <Badge variant="light" color="red">Invite expired</Badge>;
  return <Badge variant="light" color="green">Active</Badge>;
}

/** The scope pickers' options: the whole trust and each hospital, leaving out `taken` scopes. */
function scopeOptions(trust: Trust, taken: string[]) {
  return [{ value: WHOLE_TRUST, label: "Whole trust" }, ...trust.hospitals.map((h) => ({ value: h.id, label: h.name }))].filter(
    (o) => !taken.includes(o.value),
  );
}

/** The scope (the whole trust or one of its hospitals) and role pickers. */
function ChoiceFields(props: { options: { value: string; label: string }[]; value: Choice; onChange: (c: Choice) => void }) {
  const { value, onChange } = props;
  return (
    <>
      <Select
        label="Access"
        description="The whole trust, or one hospital only"
        data={props.options}
        value={value.scope}
        onChange={(v) => v && onChange({ ...value, scope: v })}
        allowDeselect={false}
        comboboxProps={{ withinPortal: false }}
      />
      <Select
        label="Role"
        data={roles.map((r) => ({ value: r, label: roleLabel[r] }))}
        value={value.role}
        onChange={(v) => v && onChange({ ...value, role: v as Role })}
        allowDeselect={false}
        comboboxProps={{ withinPortal: false }}
      />
    </>
  );
}

type FormProps = { trust: Trust; path: string; onCancel: () => void; onDone: (who: Person | null, message: string) => void };

/** Invites someone new by email, or (with `person`) gives someone on the staff another access. */
function InviteForm(props: FormProps & { person?: Person }) {
  const { trust, person } = props;
  const options = scopeOptions(trust, person ? person.access.map(scopeOf) : []);
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  // Starts on the first hospital on offer (else the whole trust), as a clinician.
  const first = options.find((o) => o.value !== WHOLE_TRUST) ?? options[0];
  const [choice, setChoice] = useState<Choice>({ scope: first?.value ?? WHOLE_TRUST, role: "clinician" });
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      const r = await api<Person & { existingAccount: boolean }>(`${props.path}/invites`, {
        method: "POST",
        body: JSON.stringify({ email: person?.email ?? email, name, ...choiceBody(choice) }),
      });
      const where = placeName(trust, choice.scope);
      props.onDone(
        person ?? null,
        person
          ? `${r.name} now also has access to ${where} as ${roleLabel[choice.role]}.`
          : r.existingAccount
            ? `${r.name} already had an account, so they were given access to ${where} as ${roleLabel[choice.role]}. No new account was made.`
            : `Invitation sent to ${r.email}. It expires on ${formatShortDate(r.inviteExpiresAt!)}.`,
      );
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return (
    // noValidate: the API's plain-language message explains a bad email, not the browser's.
    <form onSubmit={submit} noValidate>
      <Stack>
        {error && <Alert color="red">{error}</Alert>}
        {person ? (
          <Text size="sm">{person.email}</Text>
        ) : (
          <>
            <TextInput label="Email" type="email" required value={email} onChange={(e) => setEmail(e.currentTarget.value)} data-autofocus />
            <TextInput
              label="Name"
              description="Optional. Used until they sign in for the first time."
              value={name}
              onChange={(e) => setName(e.currentTarget.value)}
            />
          </>
        )}
        {options.length === 0 ? (
          <Text size="sm" c="dimmed">They already have access to the whole trust and every hospital. Change one of their accesses instead.</Text>
        ) : (
          <ChoiceFields options={options} value={choice} onChange={setChoice} />
        )}
        {!person && <Text size="sm" c="dimmed">The invitation expires after 7 days.</Text>}
        <Group justify="flex-end">
          <Button variant="default" onClick={props.onCancel}>Cancel</Button>
          <Button type="submit" loading={saving} disabled={options.length === 0}>{person ? "Add access" : "Send invite"}</Button>
        </Group>
      </Stack>
    </form>
  );
}

/** Changes one access: its scope, its role, or both. */
function ChangeForm(props: FormProps & { person: Person; access: Access }) {
  const { trust, person, access } = props;
  // Its own scope, plus any the person doesn't already have.
  const options = scopeOptions(trust, person.access.map(scopeOf).filter((s) => s !== scopeOf(access)));
  const [choice, setChoice] = useState<Choice>({ scope: scopeOf(access), role: access.role });
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      await api<Person>(`${props.path}/${person.userId}/access/${scopeOf(access)}`, { method: "PATCH", body: JSON.stringify(choiceBody(choice)) });
      const where = choice.scope === WHOLE_TRUST ? "whole trust" : placeName(trust, choice.scope);
      props.onDone(person, `${person.name} is now ${roleLabel[choice.role]}, ${where}.`);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={submit}>
      <Stack>
        {error && <Alert color="red">{error}</Alert>}
        <ChoiceFields options={options} value={choice} onChange={setChoice} />
        <Text size="sm" c="dimmed">The change takes effect on their next click.</Text>
        <Group justify="flex-end">
          <Button variant="default" onClick={props.onCancel}>Cancel</Button>
          <Button type="submit" loading={saving}>Save</Button>
        </Group>
      </Stack>
    </form>
  );
}

/** Removes one access (`access`), or every access the person has in the trust (`access` null). */
function RemoveForm(props: FormProps & { person: Person; access: Access | null }) {
  const { trust, person, access } = props;
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const lastOne = !access || person.access.length === 1;

  const confirm = async () => {
    setSaving(true);
    setError("");
    try {
      await api(`${props.path}/${person.userId}${access ? `/access/${scopeOf(access)}` : ""}`, { method: "DELETE" });
      props.onDone(person, `${person.name} no longer has access to ${access && !lastOne ? placeOf(access) : trust.name}.`);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Stack>
      {error && <Alert color="red">{error}</Alert>}
      <Text>
        {lastOne
          ? `They lose access to ${trust.name} at once, at every hospital.`
          : `They lose access to ${placeOf(access!)} at once. Their other access stays.`}{" "}
        Their account and everything they did are kept.
      </Text>
      <Group justify="flex-end">
        <Button variant="default" onClick={props.onCancel}>Cancel</Button>
        <Button color="red" onClick={confirm} loading={saving}>Remove</Button>
      </Group>
    </Stack>
  );
}
