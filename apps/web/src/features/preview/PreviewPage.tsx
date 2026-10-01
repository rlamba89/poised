"use client";
// Preview a chapter as a patient or a clinician (PRV-01), on a phone, tablet or desktop and in
// any translated language (PRV-02), with a sample patient (PRV-04), the outputs the answers
// produce, live (PRV-03), and saved test cases (PRV-05/06).
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
  computeOutputs, forPatient, showBands, localesIn, refreshClinicalSummaries, registerClinicalProperties, setViewer,
  LANGUAGES, type ChapterJson, type ComputedOutputs, type Viewer,
} from "@sj/clinical";
import { api } from "@/lib/api";
import { clinicianModel } from "./clinicianView";
import { SetCondition } from "./SetCondition";
import { TestCases } from "./TestCases";
import { showUnits } from "./units";

registerClinicalProperties();

type ChapterDetail = {
  id: string;
  name: string;
  content: object;
  revision: number;
  versionStatus: string;
  questionnaireId: string;
  questionnaireName: string;
};

export type SamplePatient = { name: string; age: number | ""; sex: string | null };

const DEVICES = { phone: 390, tablet: 820, desktop: undefined } as const;
type Device = keyof typeof DEVICES;

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
  const [device, setDevice] = useState<Device>("desktop");
  const [locale, setLocale] = useState("");
  // The JSON and revision change when test cases are saved from here.
  const [content, setContent] = useState(chapter.content as ChapterJson);
  const [revision, setRevision] = useState(chapter.revision);
  const answers = useRef<Record<string, unknown>>({});
  const [restored, setRestored] = useState(0);

  // Rebuilt when the viewer changes. Patients get the JSON without clinician-only content, with
  // each Section on its own screen and without page titles (VEW-01, STR-03). Clinicians see whole
  // pages, with each page's Clinical summary box (QT-09).
  const model = useMemo(() => {
    let m: Model;
    if (viewer === "patient") {
      m = new Model(forPatient(content));
      setViewer(m, viewer);
      showUnits(m);
      showBands(m);
      m.showPageTitles = false;
    } else m = clinicianModel(content);
    m.data = answers.current; // keep answers across the switch
    m.completedHtml = "<p>End of the chapter preview.</p>";
    return m;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [content, viewer, restored]);

  // A replaced survey is disposed once the new one is showing, so its resize observers stop
  // watching removed questions. (Not in the effect's cleanup: React's development mode runs
  // that straight away, which would dispose the survey on screen.)
  const shown = useRef<Model | null>(null);
  useEffect(() => {
    const old = shown.current;
    shown.current = model;
    if (old && old !== model) old.dispose();
  }, [model]);

  useEffect(() => {
    model.locale = locale;
  }, [model, locale]);

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
      refreshClinicalSummaries(model);
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
  const locales = localesIn(content);
  const width = DEVICES[device];
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
          {locales.length > 0 && (
            <Select
              size="xs"
              aria-label="Language"
              w={140}
              data={[{ value: "", label: "English" }, ...locales.map((l) => ({ value: l, label: LANGUAGES[l] ?? l.toUpperCase() }))]}
              value={locale}
              onChange={(v) => setLocale(v ?? "")}
              allowDeselect={false}
            />
          )}
          <SegmentedControl
            size="xs"
            aria-label="Screen size"
            value={device}
            onChange={(v) => setDevice(v as Device)}
            data={[
              { value: "phone", label: "Phone" },
              { value: "tablet", label: "Tablet" },
              { value: "desktop", label: "Desktop" },
            ]}
          />
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
          <Button size="xs" variant="light" component={Link} href={`${base}/questionnaires/${chapter.questionnaireId}?set=${chapter.id}`}>Edit</Button>
        </Group>
      </Group>
      <Grid>
        <Grid.Col span={{ base: 12, md: 8 }}>
          <SetCondition
            hospitalId={hospitalId}
            questionnaireId={chapter.questionnaireId}
            chapterId={chapter.id}
            content={content}
            variables={{ patientAge: patient.age === "" ? undefined : patient.age, patientSex: patient.sex ?? undefined, viewer }}
          />
          <Box maw={width} mx="auto" style={{ transition: "max-width 0.2s" }} className={width ? "sj-device" : undefined}>
            <Card withBorder padding={0} className={viewer === "clinician" && device === "desktop" ? "sj-clinician-layout" : undefined}>
              <Survey model={model} />
            </Card>
          </Box>
        </Grid.Col>
        <Grid.Col span={{ base: 12, md: 4 }}>
          <Stack>
            <SamplePatientCard value={patient} onChange={setPatient} />
            <OutputsPanel result={result} />
            <TestCases
              hospitalId={hospitalId}
              chapterId={chapter.id}
              content={content}
              revision={revision}
              viewer={viewer}
              patient={patient}
              model={model}
              onSaved={(c, r) => {
                setContent(c);
                setRevision(r);
              }}
              onLoad={(tc) => {
                answers.current = tc.answers;
                setPatient({ name: tc.patient.name, age: tc.patient.age ?? "", sex: tc.patient.sex ?? null });
                setViewerState(tc.viewer);
                setRestored((n) => n + 1);
              }}
            />
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
