// The trust's settings (plan C1, S10; ORG-04), for its whole-trust admins only; the API
// (/api/o/{trust}/settings) refuses everyone else. Pick "the whole trust" to set the defaults every
// hospital follows, or a hospital to give it its own values where a hospital may override.
// Trust-only settings and the opt-ins (set by Preopsis's platform admin, PRM-08) are shown, not edited.
//
// Every setting's label, group, kind and limits come from the API (Go's settings.Defs), so this
// page draws one field per kind and has no list of settings of its own.
import { useCallback, useEffect, useState } from "react";
import {
  Alert, Badge, Button, Checkbox, Group, NumberInput, Paper, Select, Stack, Switch, Text, TextInput, Textarea, Title,
} from "@mantine/core";
import { api } from "@/lib/api";
import { canManageSettings, useHospital } from "@/lib/auth";
import type { SettingDef } from "@/features/admin/types";

type Values = Record<string, unknown>;
type HospitalSettings = { id: string; name: string; status: "active" | "suspended"; values: Values; effective: Values };
/** GET /api/o/{trust}/settings (Go: settingsView in internal/httpapi/trust_settings.go). */
type View = {
  defs: SettingDef[];
  trust: Values; // set for the trust
  effective: Values; // the trust's, with Preopsis's defaults
  hospitals: HospitalSettings[];
  questionnaires: { id: string; name: string }[];
};

const TRUST = "trust";
const NOTHING = "Nothing has changed.";

export function SettingsPage() {
  const { trust } = useHospital();
  const allowed = canManageSettings(trust);
  const path = `/o/${trust.id}/settings`;
  const [view, setView] = useState<View | null>(null);
  const [scope, setScope] = useState(TRUST);
  const [saved, setSaved] = useState(0); // bumped after a save, so the form starts from what was saved
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const load = useCallback(() => {
    api<View>(path).then(setView).catch((e: Error) => setError(e.message));
  }, [path]);
  useEffect(() => {
    if (allowed) load();
  }, [allowed, load]);

  if (!allowed) return <Alert color="red" title="Only the trust's admins can change its settings" />;
  if (!view) return error ? <Alert color="red">{error}</Alert> : null;
  const hospital = view.hospitals.find((h) => h.id === scope);

  return (
    <Stack maw={900}>
      <div>
        <Title order={3}>Settings</Title>
        <Text size="sm" c="dimmed">
          {trust.name}'s settings. Every hospital follows the trust's, except where you give a hospital its own.
        </Text>
      </div>
      <Select label="Settings for" allowDeselect={false} w={360} value={scope}
        onChange={(v) => {
          setScope(v ?? TRUST);
          setNotice("");
        }}
        data={[
          { value: TRUST, label: "The whole trust (every hospital's default)" },
          ...view.hospitals.map((h) => ({ value: h.id, label: h.status === "suspended" ? `${h.name} (suspended)` : h.name })),
        ]} />
      <SettingsForm key={`${scope}:${saved}`} view={view} hospital={hospital} path={path} notice={notice} setNotice={setNotice}
        onSaved={(v) => {
          setView(v);
          setSaved((n) => n + 1);
          setNotice(`${hospital ? hospital.name : trust.name}'s settings were saved.`);
        }} />
    </Stack>
  );
}

/** One setting being edited: following the level above, or its own value. */
type Draft = { follow: boolean; value: unknown };

const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);

/** "2, 5" for [2, 5], and back; anything that isn't a list of numbers is sent as typed, for the API to refuse. */
const daysText = (v: unknown) => (Array.isArray(v) ? v.join(", ") : String(v ?? ""));
const parseDays = (s: string): unknown =>
  /^\s*(\d+\s*(,\s*\d+\s*)*)?$/.test(s) ? s.split(",").map((x) => x.trim()).filter(Boolean).map(Number) : s;

/**
 * One level's settings. `notice` (the result of the last Save) is shown beside the Save button, where
 * the user is looking (S10 test report 1, F2); it's kept in SettingsPage because the form starts
 * afresh after each save.
 */
function SettingsForm({ view, hospital, path, notice, setNotice, onSaved }: {
  view: View; hospital?: HospitalSettings; path: string; notice: string; setNotice: (s: string) => void; onSaved: (v: View) => void;
}) {
  // What this level has set, what applies here, and what it follows when it sets nothing.
  const own = hospital ? hospital.values : view.trust;
  const effective = hospital ? hospital.effective : view.effective;
  const parent = (d: SettingDef) => (hospital ? view.effective[d.key] : d.default);
  const editable = (d: SettingDef) => !d.optIn && (!hospital || d.level === "hospital");

  const [drafts, setDrafts] = useState<Record<string, Draft>>(() =>
    Object.fromEntries(view.defs.map((d) => [d.key, { follow: !(d.key in own), value: d.kind === "dayList" ? daysText(effective[d.key]) : effective[d.key] }])),
  );
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const set = (key: string, d: Partial<Draft>) => setDrafts({ ...drafts, [key]: { ...drafts[key], ...d } });

  const save = async () => {
    setError("");
    setNotice("");
    const values: Values = {};
    for (const d of view.defs.filter(editable)) {
      const draft = drafts[d.key];
      const value = d.kind === "dayList" ? parseDays(String(draft.value)) : draft.value;
      if (draft.follow) {
        if (d.key in own) values[d.key] = null; // back to following
      } else if (!(d.key in own) || !same(value, own[d.key])) {
        values[d.key] = value;
      }
    }
    if (Object.keys(values).length === 0) {
      setNotice(NOTHING);
      return;
    }
    setSaving(true);
    try {
      const v = await api<View>(path, { method: "PATCH", body: JSON.stringify({ hospitalId: hospital?.id ?? null, values }) });
      onSaved(v);
    } catch (e) {
      setError((e as Error).message);
      setSaving(false);
    }
  };

  const groups = [...new Set(view.defs.map((d) => d.group))];
  const followLabel = hospital ? "Use the trust's setting" : "Use Preopsis's default";
  return (
    <Stack>
      {hospital && (
        <Alert color="blue">
          {hospital.name} follows the trust's settings unless you untick "Use the trust's setting" and give it its own.
        </Alert>
      )}
      {groups.map((g) => {
        // At a hospital, trust-only settings are shown too, as "Set for the whole trust" (S10 test report 1, F1).
        const defs = view.defs.filter((d) => d.group === g);
        return (
          <Paper key={g} withBorder p="md" component="section" aria-label={g}>
            <Title order={5} mb="sm">{g}</Title>
            <Stack gap="md">
              {defs.map((d) => (
                <div key={d.key}>
                  {editable(d) ? (
                    <Stack gap={4}>
                      <Field def={d} value={drafts[d.key].follow ? (d.kind === "dayList" ? daysText(parent(d)) : parent(d)) : drafts[d.key].value}
                        disabled={drafts[d.key].follow} questionnaires={view.questionnaires} onChange={(value) => set(d.key, { value })} />
                      <Checkbox size="xs" label={followLabel} aria-label={`${followLabel}: ${d.label}`} checked={drafts[d.key].follow}
                        onChange={(e) => {
                          const follow = e.currentTarget.checked;
                          // Starting its own value from what applies now.
                          set(d.key, { follow, value: follow ? drafts[d.key].value : d.kind === "dayList" ? daysText(parent(d)) : parent(d) });
                        }} />
                    </Stack>
                  ) : (
                    <ReadOnly def={d} value={effective[d.key]} hospital={!!hospital} questionnaires={view.questionnaires} />
                  )}
                </div>
              ))}
            </Stack>
          </Paper>
        );
      })}
      {notice && (
        <Alert color={notice === NOTHING ? "blue" : "green"} withCloseButton onClose={() => setNotice("")} role="status">{notice}</Alert>
      )}
      {error && <Alert color="red">{error}</Alert>}
      <Group justify="flex-end">
        <Button onClick={save} loading={saving}>Save {hospital ? hospital.name : "the trust"}'s settings</Button>
      </Group>
    </Stack>
  );
}

/** A setting shown but not editable here: an opt-in (Preopsis sets it), or a trust-only one at a hospital. */
function ReadOnly({ def, value, hospital, questionnaires }: { def: SettingDef; value: unknown; hospital: boolean; questionnaires: View["questionnaires"] }) {
  const why = def.optIn ? "Set by Preopsis's platform admin" : hospital ? "Set for the whole trust" : "";
  return (
    <Group justify="space-between" wrap="nowrap" aria-label={def.label}>
      <div>
        <Text size="sm" fw={500}>{def.label}</Text>
        <Text size="xs" c="dimmed">{why}</Text>
      </div>
      <Badge variant="light" color={value === false ? "gray" : "blue"}>{show(def, value, questionnaires)}</Badge>
    </Group>
  );
}

/** A value in words, for read-only settings. */
function show(def: SettingDef, value: unknown, questionnaires: View["questionnaires"]): string {
  if (def.kind === "bool") return value ? "On" : "Off";
  if (def.kind === "dayList") return daysText(value) || "None";
  if (def.kind === "questionnaire") return questionnaires.find((q) => q.id === value)?.name ?? "None";
  return String(value ?? "") || "Not set";
}

/** The input for one setting, by its kind. */
function Field({ def, value, disabled, questionnaires, onChange }: {
  def: SettingDef; value: unknown; disabled: boolean; questionnaires: View["questionnaires"]; onChange: (v: unknown) => void;
}) {
  // A disabled field shows what it follows; "Not set" when that's empty.
  const common = { label: def.label, description: def.help, disabled, placeholder: disabled ? "Not set" : undefined };
  switch (def.kind) {
    case "bool":
      return <Switch {...common} checked={!!value} onChange={(e) => onChange(e.currentTarget.checked)} />;
    case "int":
      // clampBehavior "none": a number out of range stays as typed, and Save shows the API's refusal
      // ("…: enter a whole number from 1 to 365."), rather than quietly saving the nearest allowed
      // one (S22 test report 1, F-1). Every number setting uses this box. Decimals are allowed in the
      // box so "7.5" isn't read as 75 or 7; the API refuses anything but a whole number (test report 2).
      return <NumberInput {...common} w={200} min={def.min} max={def.max} clampBehavior="none" value={value as number}
        onChange={(v) => onChange(typeof v === "number" ? v : v === "" ? "" : Number(v))} />;
    case "dayList":
      return <TextInput {...common} w={300} placeholder={disabled ? "None" : "e.g. 2, 5"} value={String(value ?? "")}
        description={`Days after the invite, separated by commas (up to 5, each ${def.min} to ${def.max}).`}
        onChange={(e) => onChange(e.currentTarget.value)} />;
    case "text":
    case "template":
      return <Textarea {...common} autosize minRows={2} maxRows={8} value={String(value ?? "")} onChange={(e) => onChange(e.currentTarget.value)} />;
    case "questionnaire": {
      const id = String(value ?? "");
      const data = [{ value: "", label: "None" }, ...questionnaires.map((q) => ({ value: q.id, label: q.name }))];
      if (id && !questionnaires.some((q) => q.id === id)) data.push({ value: id, label: "A questionnaire that's no longer published" });
      return <Select {...common} w={360} searchable nothingFoundMessage="No published questionnaire has that name" allowDeselect={false}
        data={data} value={id} onChange={(v) => onChange(v ?? "")} />;
    }
    default: // url, email, phone
      return <TextInput {...common} value={String(value ?? "")} onChange={(e) => onChange(e.currentTarget.value)} />;
  }
}
