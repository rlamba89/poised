"use client";
// Text with a pencil button; editing shows an input with cancel and confirm (Lifebox EditableText).
import { useEffect, useState } from "react";
import { ActionIcon, Group, Text, TextInput, type TextProps } from "@mantine/core";
import { IconCheck, IconPencil, IconX } from "@tabler/icons-react";

export function EditableText(props: { value: string; onSave: (v: string) => void; readOnly?: boolean; label: string } & Pick<TextProps, "fw" | "size">) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(props.value);
  useEffect(() => setDraft(props.value), [props.value]);

  if (!editing) {
    return (
      <Group gap={6} wrap="nowrap">
        <Text fw={props.fw ?? 700} size={props.size}>{props.value}</Text>
        {!props.readOnly && (
          <ActionIcon variant="subtle" color="gray" size="sm" aria-label={`Edit ${props.label}`} onClick={() => setEditing(true)}>
            <IconPencil size={15} />
          </ActionIcon>
        )}
      </Group>
    );
  }
  const save = () => {
    const v = draft.trim();
    if (v && v !== props.value) props.onSave(v);
    setEditing(false);
  };
  return (
    <Group gap={6} wrap="nowrap">
      <TextInput
        aria-label={props.label}
        value={draft}
        onChange={(e) => setDraft(e.currentTarget.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") save();
          if (e.key === "Escape") setEditing(false);
        }}
        autoFocus
        w={360}
      />
      <ActionIcon variant="subtle" color="gray" aria-label="Cancel" onClick={() => { setDraft(props.value); setEditing(false); }}>
        <IconX size={16} />
      </ActionIcon>
      <ActionIcon variant="subtle" aria-label="Save" onClick={save}>
        <IconCheck size={16} />
      </ActionIcon>
    </Group>
  );
}
