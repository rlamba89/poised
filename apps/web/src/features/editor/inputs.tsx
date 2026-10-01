"use client";
// Inputs that save when they lose focus, so each edit is one undo step (not one per letter).
import { useEffect, useState } from "react";
import { NumberInput, Textarea, TextInput, type NumberInputProps, type TextareaProps, type TextInputProps } from "@mantine/core";

export function BlurText({ value, onSave, ...rest }: { value: string; onSave: (v: string) => void } & Omit<TextInputProps, "value" | "onChange" | "onBlur">) {
  const [draft, setDraft] = useState(value);
  useEffect(() => setDraft(value), [value]);
  return <TextInput {...rest} value={draft} onChange={(e) => setDraft(e.currentTarget.value)} onBlur={() => draft !== value && onSave(draft)} />;
}

export function BlurTextarea({ value, onSave, ...rest }: { value: string; onSave: (v: string) => void } & Omit<TextareaProps, "value" | "onChange" | "onBlur">) {
  const [draft, setDraft] = useState(value);
  useEffect(() => setDraft(value), [value]);
  return <Textarea autosize minRows={2} {...rest} value={draft} onChange={(e) => setDraft(e.currentTarget.value)} onBlur={() => draft !== value && onSave(draft)} />;
}

/** A number, or undefined when the box is empty. */
export function BlurNumber({ value, onSave, ...rest }: { value: number | undefined; onSave: (v: number | undefined) => void } & Omit<NumberInputProps, "value" | "onChange" | "onBlur">) {
  const [draft, setDraft] = useState<number | string>(value ?? "");
  useEffect(() => setDraft(value ?? ""), [value]);
  return (
    <NumberInput
      {...rest}
      value={draft}
      onChange={setDraft}
      onBlur={() => {
        const next = draft === "" ? undefined : Number(draft);
        if (next !== value) onSave(next);
      }}
    />
  );
}
