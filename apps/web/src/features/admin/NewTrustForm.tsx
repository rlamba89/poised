// New trust (S10, ORG-02): name, code, contact, logo address, the first admin (invited as on the
// Staff screen, S09) and the opt-ins (ORG-05). The trust starts with no hospitals; they're added
// on the trust's page next.
import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router";
import { Alert, Anchor, Button, Checkbox, Group, Paper, SimpleGrid, Stack, Text, TextInput, Title } from "@mantine/core";
import { api } from "@/lib/api";
import type { AdminTrust, OptIns, SettingDef } from "./types";

const empty = { name: "", code: "", contactName: "", contactEmail: "", contactPhone: "", logoUrl: "", adminEmail: "", adminName: "" };

export function NewTrustForm() {
  const navigate = useNavigate();
  const [form, setForm] = useState(empty);
  const [defs, setDefs] = useState<SettingDef[]>([]);
  const [optIns, setOptIns] = useState<OptIns>({});
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  // Each opt-in starts at the platform's default.
  useEffect(() => {
    api<SettingDef[]>("/admin/opt-ins")
      .then((d) => {
        setDefs(d);
        setOptIns(Object.fromEntries(d.map((x) => [x.key, x.default === true])));
      })
      .catch((e: Error) => setError(e.message));
  }, []);

  const field = (key: keyof typeof empty) => ({
    value: form[key],
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: e.currentTarget.value }),
  });

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      const t = await api<AdminTrust>("/admin/orgs", { method: "POST", body: JSON.stringify({ ...form, optIns }) });
      const admin = t.admins[0];
      const notice = t.existingAccount
        ? `${t.name} was created. ${admin.email} already had an account, so they were made its admin.`
        : `${t.name} was created. An invitation was sent to ${admin.email}. Next, add its hospitals.`;
      navigate(`/admin/trusts/${t.id}`, { state: { notice } });
    } catch (err) {
      setError((err as Error).message);
      setSaving(false);
    }
  };

  return (
    <Stack maw={760}>
      <Anchor component={Link} to="/admin" size="sm">← Trusts</Anchor>
      <Title order={3}>New trust</Title>
      <form onSubmit={submit} noValidate>
        <Stack>
          <Paper withBorder p="md">
            <Title order={5} mb="sm">The trust</Title>
            <SimpleGrid cols={{ base: 1, sm: 2 }}>
              <TextInput label="Trust name" required {...field("name")} />
              <TextInput label="Code" description="2 to 12 letters, digits or hyphens, e.g. RXH" required {...field("code")} />
              <TextInput label="Logo web address" description="Optional. A link to the logo image." {...field("logoUrl")} />
            </SimpleGrid>
          </Paper>
          <Paper withBorder p="md">
            <Title order={5} mb="sm">Contact at the trust</Title>
            <Text size="sm" c="dimmed" mb="sm">Who Poised deals with. Patients don't see it.</Text>
            <SimpleGrid cols={{ base: 1, sm: 3 }}>
              <TextInput label="Contact name" required {...field("contactName")} />
              <TextInput label="Contact email" type="email" required {...field("contactEmail")} />
              <TextInput label="Contact phone" {...field("contactPhone")} />
            </SimpleGrid>
          </Paper>
          <Paper withBorder p="md">
            <Title order={5} mb="sm">First admin</Title>
            <Text size="sm" c="dimmed" mb="sm">They're invited by email as an admin of the whole trust, and can then invite the trust's staff.</Text>
            <SimpleGrid cols={{ base: 1, sm: 2 }}>
              <TextInput label="Admin's email" type="email" required {...field("adminEmail")} />
              <TextInput label="Admin's name" description="Optional" {...field("adminName")} />
            </SimpleGrid>
          </Paper>
          <Paper withBorder p="md">
            <Title order={5} mb="sm">Features and opt-ins</Title>
            <Text size="sm" c="dimmed" mb="sm">The trust's default. Each hospital follows it unless you set the hospital on its own.</Text>
            <SimpleGrid cols={{ base: 1, sm: 2 }}>
              {defs.map((d) => (
                <Checkbox key={d.key} label={d.label} checked={!!optIns[d.key]}
                  onChange={(e) => setOptIns({ ...optIns, [d.key]: e.currentTarget.checked })} />
              ))}
            </SimpleGrid>
          </Paper>
          {error && <Alert color="red">{error}</Alert>}
          <Group justify="flex-end">
            <Button variant="default" component={Link} to="/admin">Cancel</Button>
            <Button type="submit" loading={saving}>Create trust</Button>
          </Group>
        </Stack>
      </form>
    </Stack>
  );
}
