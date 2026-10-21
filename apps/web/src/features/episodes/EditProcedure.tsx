// The episode's procedure and its dates (S19, EPI-01), edited on the episode page until the HQ
// review is complete; the API refuses a change after that (409). The procedure is one the trust's
// list offers at this hospital, or the one the episode already has, even if it's since retired.
import { useEffect, useRef, useState } from "react";
import { Alert, Button, Card, Group, Select, SimpleGrid, TextInput, Title } from "@mantine/core";
import { api } from "@/lib/api";
import { useHospital } from "@/lib/auth";
import { procedureChoices, type OfferedProcedure } from "@/features/procedures/procedures";
import { partlyTypedDate } from "./dates";
import type { Episode } from "./types";

export function EditProcedure({ path, episode, onSaved, onCancel }: {
  path: string;
  episode: Episode;
  onSaved: () => void;
  onCancel: () => void;
}) {
  const { trust, hospital } = useHospital();
  const [offered, setOffered] = useState<OfferedProcedure[]>([]);
  const [form, setForm] = useState({
    procedureId: episode.procedureId ?? "", plannedDate: episode.plannedDate ?? "", poaDate: episode.poaDate ?? "",
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const plannedRef = useRef<HTMLInputElement>(null);
  const poaRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    api<OfferedProcedure[]>(`/o/${trust.id}/procedures?hospital=${hospital.id}`).then(setOffered).catch((e: Error) => setError(e.message));
  }, [trust.id, hospital.id]);

  const save = async (e?: React.FormEvent) => {
    e?.preventDefault(); // Enter in a field sends the form
    if (saving) return;
    const partly = partlyTypedDate([
      { label: "planned procedure date", input: plannedRef.current },
      { label: "POA appointment date", input: poaRef.current },
    ]);
    if (partly) {
      setError(partly); // a half-typed date reads as none: never drop it silently (S19 F1)
      return;
    }
    setSaving(true);
    try {
      await api(path, { method: "PATCH", body: JSON.stringify({ ...form, revision: episode.revision }) }); // S20: a stale page gets 409
      onSaved();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card withBorder data-testid="edit-procedure" component="form" onSubmit={save} noValidate>
      <Title order={5} mb="xs">Procedure and dates</Title>
      <SimpleGrid cols={{ base: 1, sm: 3 }}>
        <Select
          label="Procedure"
          data={procedureChoices(offered, { id: episode.procedureId, name: episode.procedure })}
          value={form.procedureId || null}
          onChange={(v) => setForm({ ...form, procedureId: v ?? "" })}
          searchable
          clearable
        />
        <TextInput label="Planned procedure date" type="date" ref={plannedRef} value={form.plannedDate}
          onChange={(e) => setForm({ ...form, plannedDate: e.currentTarget.value })} />
        <TextInput label="POA appointment date" type="date" ref={poaRef} value={form.poaDate}
          onChange={(e) => setForm({ ...form, poaDate: e.currentTarget.value })} />
      </SimpleGrid>
      {error && <Alert color="red" mt="sm">{error}</Alert>}
      <Group justify="flex-end" mt="sm">
        <Button variant="default" onClick={onCancel}>Cancel</Button>
        <Button type="submit" loading={saving}>Save</Button>
      </Group>
    </Card>
  );
}
