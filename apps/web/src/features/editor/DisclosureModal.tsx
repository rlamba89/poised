"use client";
// The disclosures (clinical outputs) of one option, or of a Text / Date / Number question.
// Opened from a `🔗 n` chip or the Disclosures tab. Edits the chapter JSON.
import { useState } from "react";
import { Button, Group, Modal, Stack, Text } from "@mantine/core";
import {
  bandsOf, findElement, kindOf, newId, optionsOf, renderNote, textOf, updateElement, type ChapterJson, type ClinicalOutput, type ElementJson,
} from "@poised/clinical";
import { OutputEditor } from "@/features/outputs/OutputEditor";
import { OutputSummary } from "@/features/outputs/OutputSummary";
import { useCategories } from "@/features/outputs/useCategories";

/** Where disclosures sit: on the question, one option, one grid cell (QT-10) or one score band (CAL-02). */
export type DisclosureTarget = { element: string; option?: string; cell?: { row: string; column: string }; band?: string };

type Cells = Record<string, Record<string, ClinicalOutput[]>>;

/** The outputs on a target, read from the JSON. */
export function outputsAt(doc: ChapterJson, t: DisclosureTarget): ClinicalOutput[] {
  const el = findElement(doc, t.element)?.el;
  if (!el) return [];
  if (t.cell) return ((el.cellOutputs ?? {}) as Cells)[t.cell.row]?.[t.cell.column] ?? [];
  if (t.band) return bandsOf(el).find((b) => b.id === t.band)?.clinicalOutputs ?? [];
  if (t.option === undefined) return el.clinicalOutputs ?? [];
  return optionsOf(el).find((o) => o.value === t.option)?.clinicalOutputs ?? [];
}

/** A change that replaces the outputs on a target. */
export function setOutputsAt(t: DisclosureTarget, items: ClinicalOutput[]) {
  return (doc: ChapterJson): ChapterJson => {
    const el = findElement(doc, t.element)?.el;
    if (!el) return doc;
    const value = items.length ? items : undefined;
    if (t.cell) {
      const cells = structuredClone((el.cellOutputs ?? {}) as Cells);
      const row = (cells[t.cell.row] ??= {});
      if (value) row[t.cell.column] = value;
      else delete row[t.cell.column];
      if (!Object.keys(row).length) delete cells[t.cell.row];
      return updateElement(doc, t.element, { cellOutputs: Object.keys(cells).length ? cells : undefined });
    }
    if (t.band) {
      const bands = bandsOf(el).map((b) => (b.id === t.band ? { ...b, clinicalOutputs: value } : b));
      return updateElement(doc, t.element, { bands: bands.map(({ clinicalOutputs, ...rest }) => (clinicalOutputs ? { ...rest, clinicalOutputs } : rest)) });
    }
    if (t.option === undefined) return updateElement(doc, t.element, { clinicalOutputs: value });
    const choices = optionsOf(el).map((o) => (o.value === t.option ? { ...o, clinicalOutputs: value } : o));
    return updateElement(doc, t.element, { choices: choices.map(({ clinicalOutputs, ...rest }) => (clinicalOutputs ? { ...rest, clinicalOutputs } : rest)) });
  };
}

export function DisclosureModal(props: {
  doc: ChapterJson;
  target: DisclosureTarget | null;
  readOnly: boolean;
  onChange: (change: (doc: ChapterJson) => ChapterJson) => void;
  onClose: () => void;
}) {
  const { doc, target, readOnly, onChange, onClose } = props;
  const categories = useCategories();
  const [editing, setEditing] = useState<{ output: ClinicalOutput; isNew: boolean } | null>(null);
  const el = target ? findElement(doc, target.element)?.el : undefined;
  const option = el && target?.option !== undefined ? optionsOf(el).find((o) => o.value === target.option) : undefined;
  const band = el && target?.band ? bandsOf(el).find((b) => b.id === target.band) : undefined;
  const cell = el && target?.cell ? cellLabel(el, target.cell) : undefined;
  const outputs = target ? outputsAt(doc, target) : [];
  const sample = option ? option.text : cell ? cell : band ? "4" : el ? sampleAnswer(el) : "their answer";

  const close = () => {
    setEditing(null);
    onClose();
  };
  const save = (o: ClinicalOutput) => {
    const exists = outputs.some((x) => x.id === o.id);
    onChange(setOutputsAt(target!, exists ? outputs.map((x) => (x.id === o.id ? o : x)) : [...outputs, o]));
    setEditing(null);
  };

  const heading = option ? `${textOf(el?.title)} → ${option.text}` : cell ? `${textOf(el?.title)} → ${cell}` : band ? `${textOf(el?.title)} → ${band.label} band` : textOf(el?.title);
  return (
    <Modal opened={!!target && !!el} onClose={close} size="xl" title={editing ? (editing.isNew ? "New Disclosure" : "Edit Disclosure") : "Disclosures"}>
      <Text size="sm" c="dimmed" mb="md">{heading}</Text>
      {editing ? (
        <OutputEditor initial={editing.output} sampleAnswer={sample} categories={categories} onSave={save} onCancel={() => setEditing(null)} />
      ) : (
        <Stack>
          {outputs.length === 0 && <Text size="sm" c="dimmed">No disclosures yet.</Text>}
          {outputs.map((o) => (
            <Group key={o.id} justify="space-between" wrap="nowrap" py={4} style={{ borderBottom: "1px solid var(--mantine-color-gray-2)" }}>
              <div>
                <OutputSummary output={o} />
                {o.note?.text.includes("{answer}") && (
                  <Text size="xs" c="dimmed">e.g. {renderNote(o.note.text, sample)}</Text>
                )}
              </div>
              {!readOnly && (
                <Group gap={4} wrap="nowrap">
                  <Button size="xs" variant="subtle" onClick={() => setEditing({ output: o, isNew: false })}>Edit</Button>
                  <Button size="xs" variant="subtle" color="red" onClick={() => onChange(setOutputsAt(target!, outputs.filter((x) => x.id !== o.id)))}>
                    Remove
                  </Button>
                </Group>
              )}
            </Group>
          ))}
          <Group justify="space-between">
            {!readOnly ? (
              <Button variant="light" onClick={() => setEditing({ output: { id: newId("out", 6), codes: [] }, isNew: true })}>+ Add disclosure</Button>
            ) : (
              <span />
            )}
            <Button onClick={close}>Done</Button>
          </Group>
        </Stack>
      )}
    </Modal>
  );
}

const label = (list: unknown, value: string) =>
  textOf((Array.isArray(list) ? list : []).find((x: { value?: string }) => x && x.value === value)?.text) || value;

function cellLabel(el: Record<string, unknown>, cell: { row: string; column: string }): string {
  return `${label(el.rows, cell.row)}: ${label(el.columns, cell.column)}`;
}

/** An example answer for a note's {answer}, by type (a Number with decimal places has no inputType). */
function sampleAnswer(el: ElementJson): string {
  switch (kindOf(el)) {
    case "number":
    case "rating":
    case "calculation":
      return "20";
    case "date":
      return el.dateFormat === "y" ? "2019" : el.inputType === "month" ? "03/2025" : "14/03/2025";
    default:
      return "their answer";
  }
}
