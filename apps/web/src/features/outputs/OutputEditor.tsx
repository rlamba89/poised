"use client";
// Edit one clinical output: codes, a note with a category, ASA grade and review flag (CLN-06).
import { useState } from "react";
import { Alert, Button, Checkbox, Group, Paper, SegmentedControl, Select, Stack, Text, Textarea } from "@mantine/core";
import { renderNote, validateOutput, type AsaGrade, type ClinicalOutput, type ReviewFlag } from "@poised/clinical";
import { CodePicker } from "./CodePicker";
import type { Category } from "./useCategories";

const ASA_GRADES: AsaGrade[] = ["I", "II", "III", "IV", "V", "VI"];

export function OutputEditor(props: {
  initial: ClinicalOutput;
  /** How {answer} would read for this row, for the worked example. */
  sampleAnswer: string;
  categories: Category[];
  onSave: (o: ClinicalOutput) => void;
  onCancel: () => void;
}) {
  const [codes, setCodes] = useState(props.initial.codes);
  const [noteText, setNoteText] = useState(props.initial.note?.text ?? "");
  const [category, setCategory] = useState<string | null>(props.initial.note?.category ?? null);
  const [asa, setAsa] = useState<string | null>(props.initial.asa?.grade ?? null);
  const [emergency, setEmergency] = useState(props.initial.asa?.emergency ?? false);
  const [flag, setFlag] = useState<string>(props.initial.flag ?? "none");
  const [problems, setProblems] = useState<string[]>([]);

  const build = (): ClinicalOutput => ({
    id: props.initial.id,
    codes,
    ...(noteText.trim() ? { note: { text: noteText.trim(), category: category ?? "" } } : {}),
    ...(asa ? { asa: { grade: asa as AsaGrade, emergency } } : {}),
    ...(flag !== "none" ? { flag: flag as ReviewFlag } : {}),
  });

  const save = () => {
    const output = build();
    const found = validateOutput(output);
    setProblems(found);
    if (found.length === 0) props.onSave(output);
  };

  return (
    <Stack>
      {problems.length > 0 && (
        <Alert color="red">
          {problems.map((p) => (
            <div key={p}>{p}</div>
          ))}
        </Alert>
      )}
      <div>
        <Text fw={600} size="sm" mb={4}>Codes</Text>
        <CodePicker value={codes} onChange={setCodes} />
      </div>
      <div>
        <Text fw={600} size="sm" mb={4}>Clinical note</Text>
        <Group align="flex-start" grow>
          <Textarea
            label="Note"
            description="Use {answer} to insert the answer."
            placeholder="e.g. Smokes {answer} per day"
            autosize
            minRows={2}
            value={noteText}
            onChange={(e) => setNoteText(e.currentTarget.value)}
          />
          <Select
            label="Category"
            placeholder="Choose a category"
            data={props.categories.map((c) => c.name)}
            value={category}
            onChange={setCategory}
            searchable
            clearable
          />
        </Group>
        {noteText.includes("{answer}") && (
          <Paper withBorder p="xs" mt="xs" bg="gray.0" data-testid="note-example">
            <Text size="sm" c="dimmed">Example: when the answer is “{props.sampleAnswer}”, the note reads:</Text>
            <Text size="sm">{renderNote(noteText, props.sampleAnswer)}</Text>
          </Paper>
        )}
      </div>
      <Group align="flex-end">
        <Select
          label="ASA grade"
          placeholder="None"
          data={ASA_GRADES}
          value={asa}
          onChange={setAsa}
          clearable
          w={140}
        />
        <Checkbox label="Emergency (E)" checked={emergency} disabled={!asa} onChange={(e) => setEmergency(e.currentTarget.checked)} mb={8} />
        <div>
          <Text size="sm" fw={500} mb={4}>Review flag</Text>
          <SegmentedControl
            value={flag}
            onChange={setFlag}
            data={[
              { value: "none", label: "None" },
              { value: "amber", label: "Amber" },
              { value: "red", label: "Red" },
            ]}
          />
        </div>
      </Group>
      <Group justify="flex-end">
        <Button variant="default" onClick={props.onCancel}>Cancel</Button>
        <Button onClick={save}>Save output</Button>
      </Group>
    </Stack>
  );
}
