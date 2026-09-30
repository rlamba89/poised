"use client";
// Preview a chapter as a patient or a clinician (PRV-01), with a sample patient (PRV-04)
// and the outputs the answers produce, live (PRV-03).
import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import {
  Alert, Anchor, Badge, Box, Button, Card, Center, Grid, Group, Loader, NumberInput, SegmentedControl, Select, Stack, Text,
  TextInput, Title,
} from "@mantine/core";
import { Model } from "survey-core";
import { Survey } from "survey-react-ui";
import "survey-core/survey-core.css";
import {
  computeOutputs, registerClinicalProperties, setViewer, stripClinicianOnly, type ComputedOutputs, type Viewer,
} from "@sj/clinical";
import { api } from "@/lib/api";
import type { ChapterDetail } from "@/features/designer/DesignerPage";

registerClinicalProperties();

type SamplePatient = { name: string; age: number | ""; sex: string | null };

export default function PreviewPage({ hospitalId, chapterId }: { hospitalId: string; chapterId: string }) {
  const [chapter, setChapter] = useState<ChapterDetail | null>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    api<ChapterDetail>(`/h/${hospitalId}/chapters/${chapterId}`)
      .then(setChapter)
      .catch((e: Error) => setError(e.message));
  }, [hospitalId, chapterId]);

  if (error) return <Alert color="red" m="md">{error}</Alert>;
  if (!chapter) {
    return (
      <Center h="50vh">
        <Loader />
      </Center>
    );
  }
  return <Preview hospitalId={hospitalId} chapter={chapter} />;
}

function Preview({ hospitalId, chapter }: { hospitalId: string; chapter: ChapterDetail }) {
  const [viewer, setViewerState] = useState<Viewer>("patient");
  const [patient, setPatient] = useState<SamplePatient>({ name: "Sam Sample", age: 54, sex: "female" });
  const [result, setResult] = useState<ComputedOutputs>({ outputs: [] });
  const answers = useRef<Record<string, unknown>>({});

  // Rebuilt when the viewer changes. Patients get the JSON without clinician-only content.
  const model = useMemo(() => {
    const m = new Model(viewer === "patient" ? stripClinicianOnly(chapter.content) : chapter.content);
    setViewer(m, viewer);
    m.data = answers.current; // keep answers across the switch
    m.completedHtml = "<p>End of the chapter preview.</p>";
    return m;
  }, [chapter.content, viewer]);

  // Sample patient details are survey variables: {patientName}, {patientAge}, {patientSex}.
  useEffect(() => {
    model.setVariable("patientName", patient.name);
    model.setVariable("patientAge", patient.age === "" ? undefined : patient.age);
    model.setVariable("patientSex", patient.sex ?? undefined);
    setResult(computeOutputs(model));
  }, [model, patient]);

  useEffect(() => {
    const update = () => {
      answers.current = model.data;
      setResult(computeOutputs(model));
    };
    model.onValueChanged.add(update);
    update();
    return () => model.onValueChanged.remove(update);
  }, [model]);

  const restart = () => {
    answers.current = {};
    model.clear(true, true);
    setResult(computeOutputs(model));
  };

  const base = `/h/${hospitalId}`;
  return (
    <Box p="md">
      <Group justify="space-between" mb="md">
        <Group gap="xs">
          <Anchor component={Link} href={`${base}/questionnaires/${chapter.questionnaireId}`} size="sm">
            ← {chapter.questionnaireName}
          </Anchor>
          <Text fw={600}>/ {chapter.name} · Preview</Text>
        </Group>
        <Group gap="xs">
          <SegmentedControl
            data-testid="viewer-switch"
            value={viewer}
            onChange={(v) => setViewerState(v as Viewer)}
            data={[
              { value: "patient", label: "Patient" },
              { value: "clinician", label: "Clinician" },
            ]}
          />
          <Button size="xs" variant="default" onClick={restart}>Clear answers</Button>
          <Button size="xs" variant="light" component={Link} href={`${base}/chapters/${chapter.id}/design`}>Design</Button>
        </Group>
      </Group>
      <Grid>
        <Grid.Col span={{ base: 12, md: 8 }}>
          <Card withBorder padding={0}>
            <Survey model={model} />
          </Card>
        </Grid.Col>
        <Grid.Col span={{ base: 12, md: 4 }}>
          <Stack>
            <SamplePatientCard value={patient} onChange={setPatient} />
            <OutputsPanel result={result} />
          </Stack>
        </Grid.Col>
      </Grid>
    </Box>
  );
}

function SamplePatientCard({ value, onChange }: { value: SamplePatient; onChange: (p: SamplePatient) => void }) {
  return (
    <Card withBorder>
      <Title order={5} mb="xs">Sample patient</Title>
      <Text size="xs" c="dimmed" mb="xs">Made-up details only. Never enter a real patient.</Text>
      <Stack gap="xs">
        <TextInput label="Name" value={value.name} onChange={(e) => onChange({ ...value, name: e.currentTarget.value })} />
        <Group grow>
          <NumberInput label="Age" min={0} max={120} value={value.age} onChange={(v) => onChange({ ...value, age: v === "" ? "" : Number(v) })} />
          <Select
            label="Sex"
            data={[
              { value: "female", label: "Female" },
              { value: "male", label: "Male" },
            ]}
            value={value.sex}
            onChange={(v) => onChange({ ...value, sex: v })}
            clearable
          />
        </Group>
      </Stack>
    </Card>
  );
}

function OutputsPanel({ result }: { result: ComputedOutputs }) {
  const { outputs, suggestedAsa } = result;
  const byCategory = new Map<string, string[]>();
  for (const o of outputs) {
    if (!o.output.note || !o.noteText) continue;
    const list = byCategory.get(o.output.note.category) ?? [];
    list.push(o.noteText);
    byCategory.set(o.output.note.category, list);
  }
  const codes = outputs.flatMap((o) => o.output.codes.map((c) => ({ ...c, from: o.questionTitle })));
  const flags = outputs.filter((o) => o.output.flag);

  return (
    <Card withBorder data-testid="outputs-panel">
      <Title order={5} mb="xs">Clinical outputs</Title>
      {outputs.length === 0 && <Text size="sm" c="dimmed">The current answers produce no outputs.</Text>}
      <Stack gap="sm">
        {codes.length > 0 && (
          <div>
            <Text size="sm" fw={600}>Codes</Text>
            {codes.map((c, i) => (
              <Text size="sm" key={i} component="div">
                <Badge size="xs" variant="light" mr={4}>{c.set}</Badge>
                {c.code} {c.display}
              </Text>
            ))}
          </div>
        )}
        {byCategory.size > 0 && (
          <div>
            <Text size="sm" fw={600}>Notes</Text>
            {[...byCategory].map(([category, notes]) => (
              <div key={category}>
                <Text size="xs" c="dimmed" tt="uppercase" mt={4}>{category}</Text>
                {notes.map((n, i) => (
                  <Text size="sm" key={i}>{n}</Text>
                ))}
              </div>
            ))}
          </div>
        )}
        {suggestedAsa && (
          <div>
            <Text size="sm" fw={600}>Suggested ASA</Text>
            <Text size="sm">ASA {suggestedAsa.grade}{suggestedAsa.emergency ? "E" : ""}</Text>
          </div>
        )}
        {flags.length > 0 && (
          <div>
            <Text size="sm" fw={600}>Review flags</Text>
            {flags.map((o, i) => (
              <Text size="sm" key={i} component="div">
                <Badge size="xs" color={o.output.flag === "red" ? "red" : "orange"} mr={4}>{o.output.flag}</Badge>
                {o.questionTitle}: {o.answerLabel}
              </Text>
            ))}
          </div>
        )}
      </Stack>
    </Card>
  );
}
