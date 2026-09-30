"use client";
// Edit a chapter's name, description, icon (searchable) and audience.
import { useEffect, useState } from "react";
import {
  ActionIcon, Button, Group, Modal, ScrollArea, SegmentedControl, SimpleGrid, Stack, Text, Textarea, TextInput, Tooltip,
} from "@mantine/core";
import { CHAPTER_ICONS } from "./icons";

export type Audience = "patient" | "clinician" | "clinician_document";
export type Chapter = { id: string; position: number; name: string; description: string; icon: string; audience: Audience; revision: number };

export const AUDIENCE_LABELS: Record<Audience, string> = {
  patient: "Patient",
  clinician: "Clinician",
  clinician_document: "Clinician document",
};

type Patch = Pick<Chapter, "name" | "description" | "icon" | "audience">;

export function ChapterEditModal(props: { chapter: Chapter | null; onClose: () => void; onSave: (patch: Patch) => void }) {
  const [form, setForm] = useState<Patch>({ name: "", description: "", icon: "", audience: "patient" });
  const [iconSearch, setIconSearch] = useState("");
  useEffect(() => {
    if (props.chapter) {
      const { name, description, icon, audience } = props.chapter;
      setForm({ name, description, icon, audience });
      setIconSearch("");
    }
  }, [props.chapter]);

  const icons = Object.entries(CHAPTER_ICONS).filter(([name]) => name.includes(iconSearch.trim().toLowerCase()));

  return (
    <Modal opened={!!props.chapter} onClose={props.onClose} title="Edit chapter" size="lg">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          props.onSave(form);
        }}
      >
        <Stack>
          <TextInput label="Name" required value={form.name} onChange={(e) => setForm({ ...form, name: e.currentTarget.value })} />
          <Textarea label="Description" value={form.description} onChange={(e) => setForm({ ...form, description: e.currentTarget.value })} />
          <div>
            <Text size="sm" fw={500} mb={4}>Who is it for?</Text>
            <SegmentedControl
              value={form.audience}
              onChange={(v) => setForm({ ...form, audience: v as Audience })}
              data={Object.entries(AUDIENCE_LABELS).map(([value, label]) => ({ value, label }))}
            />
          </div>
          <div>
            <Text size="sm" fw={500} mb={4}>Icon</Text>
            <TextInput placeholder="Search icons" value={iconSearch} onChange={(e) => setIconSearch(e.currentTarget.value)} mb="xs" />
            <ScrollArea h={160}>
              <SimpleGrid cols={10} spacing={4}>
                {icons.map(([name, Icon]) => (
                  <Tooltip key={name} label={name}>
                    <ActionIcon
                      size="lg"
                      aria-label={name}
                      variant={form.icon === name ? "filled" : "subtle"}
                      onClick={() => setForm({ ...form, icon: name })}
                    >
                      <Icon size={20} />
                    </ActionIcon>
                  </Tooltip>
                ))}
              </SimpleGrid>
              {icons.length === 0 && <Text size="sm" c="dimmed">No icons match.</Text>}
            </ScrollArea>
          </div>
          <Group justify="flex-end">
            <Button variant="default" onClick={props.onClose}>Cancel</Button>
            <Button type="submit">Save</Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  );
}
