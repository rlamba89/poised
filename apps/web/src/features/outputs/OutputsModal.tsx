"use client";
// The outputs of one question: its own (text, number, date) or each option's (choice
// questions, including an exclusive "None of these"). Opened from the "Outputs (n)" badge.
import { useState } from "react";
import { Badge, Button, Divider, Group, Modal, Stack, Text } from "@mantine/core";
import type { Base, Question } from "survey-core";
import {
  choicesOf, formatAnswer, isChoiceQuestion, newId, outputsOf, setOutputs, type ClinicalOutput,
} from "@sj/clinical";
import { OutputEditor } from "./OutputEditor";
import { useCategories } from "./useCategories";

type Row = { key: string; label: string; target: Base; sampleAnswer: string };

function rowsOf(q: Question): Row[] {
  if (isChoiceQuestion(q)) {
    return choicesOf(q).map((item) => ({
      key: String(item.value),
      label: `${item.text}${(item as unknown as { isExclusive?: boolean }).isExclusive ? " (exclusive)" : ""}`,
      target: item,
      sampleAnswer: formatAnswer(q, item.value),
    }));
  }
  const inputType = (q as unknown as { inputType?: string }).inputType;
  const sample = inputType === "number" ? 20 : inputType === "date" ? "2025-03-14" : "their answer";
  return [{ key: q.name, label: "When answered", target: q, sampleAnswer: formatAnswer(q, sample) }];
}

export function OutputsModal({ question, onClose }: { question: Question | null; onClose: () => void }) {
  const categories = useCategories();
  const [editing, setEditing] = useState<{ row: Row; output: ClinicalOutput; isNew: boolean } | null>(null);
  const [, rerender] = useState(0);

  const close = () => {
    setEditing(null);
    onClose();
  };
  // Each change replaces the whole list: one undo step, then autosave.
  const save = (row: Row, output: ClinicalOutput) => {
    const current = outputsOf(row.target);
    const exists = current.some((o) => o.id === output.id);
    setOutputs(row.target, exists ? current.map((o) => (o.id === output.id ? output : o)) : [...current, output]);
    setEditing(null);
  };
  const remove = (row: Row, id: string) => {
    setOutputs(row.target, outputsOf(row.target).filter((o) => o.id !== id));
    rerender((n) => n + 1);
  };

  return (
    <Modal opened={!!question} onClose={close} size="xl" title={question ? `Clinical outputs · ${question.title}` : ""}>
      {question && editing && (
        <Stack>
          <Text size="sm" c="dimmed">
            {editing.isNew ? "New output" : "Edit output"} for <b>{editing.row.label}</b>
          </Text>
          <OutputEditor
            initial={editing.output}
            sampleAnswer={editing.row.sampleAnswer}
            categories={categories}
            onSave={(o) => save(editing.row, o)}
            onCancel={() => setEditing(null)}
          />
        </Stack>
      )}
      {question && !editing && (
        <Stack>
          {rowsOf(question).map((row, i) => (
            <div key={row.key} data-testid={`outputs-row-${i}`}>
              {i > 0 && <Divider mb="sm" />}
              <Group justify="space-between" mb={4}>
                <Text fw={600}>{row.label}</Text>
                <Button
                  size="xs"
                  variant="light"
                  onClick={() => setEditing({ row, output: { id: newId("out", 6), codes: [] }, isNew: true })}
                >
                  + Add output
                </Button>
              </Group>
              {outputsOf(row.target).length === 0 && <Text size="sm" c="dimmed">No outputs.</Text>}
              {outputsOf(row.target).map((o) => (
                <Group key={o.id} justify="space-between" wrap="nowrap" py={4}>
                  <OutputSummary output={o} />
                  <Group gap={4} wrap="nowrap">
                    <Button size="xs" variant="subtle" onClick={() => setEditing({ row, output: o, isNew: false })}>Edit</Button>
                    <Button size="xs" variant="subtle" color="red" onClick={() => remove(row, o.id)}>Delete</Button>
                  </Group>
                </Group>
              ))}
            </div>
          ))}
          <Group justify="flex-end">
            <Button onClick={close}>Done</Button>
          </Group>
        </Stack>
      )}
    </Modal>
  );
}

export function OutputSummary({ output }: { output: ClinicalOutput }) {
  return (
    <Stack gap={2}>
      {output.note && (
        <Text size="sm" component="div">
          {output.note.text} <Badge size="xs" variant="outline">{output.note.category}</Badge>
        </Text>
      )}
      <Group gap={4}>
        {output.codes.map((c) => (
          <Badge key={`${c.set}:${c.code}`} size="sm" variant="light">
            {c.set} {c.code} · {c.display}
          </Badge>
        ))}
        {output.asa && (
          <Badge size="sm" color="gray">ASA {output.asa.grade}{output.asa.emergency ? "E" : ""}</Badge>
        )}
        {output.flag && (
          <Badge size="sm" color={output.flag === "red" ? "red" : "orange"}>{output.flag} flag</Badge>
        )}
      </Group>
    </Stack>
  );
}
