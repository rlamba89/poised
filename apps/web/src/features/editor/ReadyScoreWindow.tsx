// Insert a ready-made score (S38): STOP-Bang, with its published wording, points and bands, locked.
// Its BMI comes from a BMI question or BMI Calculation in this Question Set or an earlier one.
import { useEffect, useState } from "react";
import { Alert, Button, Group, Modal, Select, Stack, Text } from "@mantine/core";
import { POINTS_MENUS, pointsSubjects, READY_SCORES, type ChapterJson } from "@poised/clinical";

export function ReadyScoreWindow(props: {
  opened: boolean;
  doc: ChapterJson | undefined;
  /** The earlier Question Sets combined, once loaded. */
  earlier: ChapterJson | undefined;
  onClose: () => void;
  /** Inserts it; returns why it can't be, or "". */
  onInsert: (id: string, bmi: string) => string;
}) {
  const { opened, doc, earlier, onClose, onInsert } = props;
  const [score, setScore] = useState(READY_SCORES[0].id);
  const [bmi, setBmi] = useState<string | null>(null);
  const [error, setError] = useState("");
  const bmis = doc ? pointsSubjects(doc, earlier, "").filter((s) => s.group === POINTS_MENUS.measurements) : [];
  useEffect(() => {
    if (!opened) return;
    setError("");
    setBmi(bmis.length === 1 ? bmis[0].left : null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [opened]);
  const chosen = READY_SCORES.find((r) => r.id === score)!;
  const insert = () => {
    const refused = onInsert(score, bmi ?? "");
    if (refused) setError(refused);
    else onClose();
  };
  return (
    <Modal opened={opened} onClose={onClose} title="Insert a ready-made score" size="lg">
      <Stack>
        <Select
          label="Score"
          data={READY_SCORES.map((r) => ({ value: r.id, label: r.name }))}
          value={score}
          allowDeselect={false}
          onChange={(v) => v && setScore(v)}
        />
        <div>
          <Text size="sm">Inserted as a Group with its published questions, points and bands. It can&apos;t be changed: you can give the Group a condition and add disclosures.</Text>
          <Text size="xs" c="dimmed" mt={6}>{chosen.source}</Text>
          <Text size="xs" c="dimmed">{chosen.licence}</Text>
        </div>
        {chosen.needsBmi &&
          (!earlier ? (
            <Text size="sm" c="dimmed">Loading the earlier Question Sets…</Text>
          ) : bmis.length ? (
            <Select
              label="BMI from"
              description="The patient's BMI, from this Question Set or an earlier one"
              data={bmis.map((s) => ({ value: s.left, label: s.label }))}
              value={bmi}
              onChange={setBmi}
            />
          ) : (
            <Alert color="orange">It uses the patient&apos;s BMI. Add a BMI question (or a BMI Calculation) to this Question Set or an earlier one first.</Alert>
          ))}
        {error && <Alert color="red">{error}</Alert>}
        <Group justify="flex-end">
          <Button variant="default" onClick={onClose}>Cancel</Button>
          <Button onClick={insert} disabled={chosen.needsBmi && !bmi}>Insert</Button>
        </Group>
      </Stack>
    </Modal>
  );
}
