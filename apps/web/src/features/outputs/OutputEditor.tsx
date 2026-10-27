// Edit one clinical output: codes, a note with a category, ASA grade and review flag (CLN-06).
// Adding a code offers its examples from the ASA guide (S39): one fills in the grade, with the
// example as its reason; several (the grade depends on severity) must be chosen from before saving.
import { useEffect, useState } from "react";
import { Alert, Button, Checkbox, Group, Paper, Radio, SegmentedControl, Select, Stack, Text, Textarea } from "@mantine/core";
import {
  afterCodeAdded, afterCodeRemoved, ASA_GRADES, gradePicked, renderNote, validateOutput, type AsaChoice, type AsaGrade, type ClinicalOutput, type CodeRef,
  type ReviewFlag,
} from "@poised/clinical";
import { asaChoicesOf, CodePicker } from "./CodePicker";
import type { Category } from "./useCategories";

const codeKey = (c: CodeRef) => `${c.set}:${c.code}`;
const choiceKey = (c: AsaChoice) => `${c.grade}|${c.example}|${c.group}`;
const NO_GRADE = "none";
const CHOOSE = "Choose the ASA grade from the ASA guide, or No ASA grade.";

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
  const [asa, setAsa] = useState<ClinicalOutput["asa"]>(props.initial.asa);
  // The ASA guide's examples for each code added while this is open (S39), and whether the author
  // still has to choose one: a code was added with several and no grade has been picked since.
  const [guideByCode, setGuideByCode] = useState<Record<string, AsaChoice[]>>({});
  const [pending, setPending] = useState(false);
  // A deliberate "No ASA grade" is saved (asaNone, S44a), so it's remembered when the disclosure is opened again.
  const [noGrade, setNoGrade] = useState(props.initial.asaNone === true && !props.initial.asa);
  const [flag, setFlag] = useState<string>(props.initial.flag ?? "none");
  const [problems, setProblems] = useState<string[]>([]);

  // The codes already on the disclosure: their examples too (test report 1, F2), so the guide box,
  // a grade picked by hand and removing a code work for them as for codes added now.
  useEffect(() => {
    let current = true;
    for (const c of props.initial.codes ?? []) {
      asaChoicesOf(c)
        .then((choices) => current && setGuideByCode((was) => ({ [codeKey(c)]: choices, ...was })))
        .catch(() => undefined); // no guide for it: the author picks a grade by hand, as before S39
    }
    return () => {
      current = false;
    };
  }, [props.initial.codes]);

  const guide = codes.flatMap((c) => guideByCode[codeKey(c)] ?? []).filter((c, i, all) => all.findIndex((x) => choiceKey(x) === choiceKey(c)) === i);
  const mustChoose = pending && !asa && !noGrade && guide.length > 1;
  const chosen = asa ? guide.find((c) => c.grade === asa.grade && c.example === asa.reason) : undefined;

  const codeAdded = (code: CodeRef, choices: AsaChoice[]) => {
    setGuideByCode((was) => ({ ...was, [codeKey(code)]: choices }));
    const next = afterCodeAdded(asa, guide, choices, noGrade); // "No ASA grade" stands (verification 1, F1)
    setAsa(next.asa);
    if (next.mustChoose) {
      setPending(true);
      setNoGrade(false);
    }
  };
  // Removing a code takes away a grade that came from it (test report 1, F1).
  const changeCodes = (next: CodeRef[]) => {
    const removed = codes.filter((c) => !next.some((n) => codeKey(n) === codeKey(c)));
    setCodes(next);
    if (!removed.length) return;
    const left = next.flatMap((c) => guideByCode[codeKey(c)] ?? []);
    const after = afterCodeRemoved(asa, removed.flatMap((c) => guideByCode[codeKey(c)] ?? []), left);
    setAsa(after.asa);
    if (after.mustChoose) {
      setPending(true);
      setNoGrade(false);
    }
  };
  const chose = () => {
    setPending(false);
    setProblems((was) => was.filter((p) => p !== CHOOSE));
  };
  const pickGrade = (grade: string | null) => {
    setAsa(gradePicked(grade as AsaGrade | null, asa?.emergency ?? false, guide));
    if (grade) setNoGrade(false); // a grade picked replaces "No ASA grade"
    chose();
  };
  const pickChoice = (key: string) => {
    if (key === NO_GRADE) {
      setAsa(undefined);
      setNoGrade(true);
    } else {
      const c = guide.find((x) => choiceKey(x) === key);
      if (c) setAsa({ grade: c.grade, emergency: asa?.emergency ?? false, reason: c.example });
      setNoGrade(false);
    }
    chose();
  };

  const build = (): ClinicalOutput => ({
    id: props.initial.id,
    codes,
    ...(noteText.trim() ? { note: { text: noteText.trim(), category: category ?? "" } } : {}),
    ...(asa ? { asa: { grade: asa.grade, emergency: asa.emergency, ...(asa.reason ? { reason: asa.reason } : {}) } } : {}),
    ...(!asa && noGrade ? { asaNone: true as const } : {}),
    ...(flag !== "none" ? { flag: flag as ReviewFlag } : {}),
  });

  const save = () => {
    const output = build();
    const found = [...validateOutput(output), ...(mustChoose ? [CHOOSE] : [])];
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
        <CodePicker value={codes} onChange={changeCodes} onAdded={codeAdded} />
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
          value={asa?.grade ?? null}
          onChange={pickGrade}
          clearable
          w={140}
        />
        <Checkbox
          label="Emergency (E)"
          checked={asa?.emergency ?? false}
          disabled={!asa}
          onChange={(e) => {
            const on = e.currentTarget.checked;
            setAsa((was) => (was ? { ...was, emergency: on } : was));
          }}
          mb={8}
        />
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
      {guide.length > 0 && (
        <Paper withBorder p="xs" bg={mustChoose ? "yellow.0" : undefined} data-testid="asa-guide">
          <Radio.Group
            label="ASA from the ASA guide"
            description={
              mustChoose
                ? "The grade depends on how severe or well controlled it is: choose one. You can change it."
                : "The ASA's own examples for these codes. Choose one to use it; you can change it."
            }
            value={chosen ? choiceKey(chosen) : noGrade ? NO_GRADE : null}
            onChange={pickChoice}
          >
            <Stack gap={6} mt={6}>
              {guide.map((c) => (
                <Radio key={choiceKey(c)} value={choiceKey(c)} label={`ASA ${c.grade}: ${c.example} (${c.group})`} />
              ))}
              <Radio value={NO_GRADE} label="No ASA grade" />
            </Stack>
          </Radio.Group>
        </Paper>
      )}
      {asa?.reason && (
        <Text size="xs" c="dimmed" data-testid="asa-reason">
          Shown with the suggested ASA as: “{asa.reason}” (from the ASA guide). The anaesthetist gives the final grade.
        </Text>
      )}
      <Group justify="flex-end">
        <Button variant="default" onClick={props.onCancel}>Cancel</Button>
        <Button onClick={save}>Save output</Button>
      </Group>
    </Stack>
  );
}
