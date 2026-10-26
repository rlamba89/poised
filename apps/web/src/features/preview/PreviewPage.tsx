// Preview a chapter as a patient or a clinician (PRV-01), on a phone, tablet or desktop and in
// any translated language (PRV-02), with a sample patient (PRV-04), the outputs the answers
// produce, live (PRV-03), and saved test cases (PRV-05/06).
import { useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router";
import {
  Alert, Anchor, Badge, Box, Button, Card, Center, Grid, Group, Loader, NumberInput, SegmentedControl, Select, Stack, Text,
  TextInput, Title,
} from "@mantine/core";
import { Model } from "survey-core";
import { Survey } from "survey-react-ui";
import "@/features/video/VideoQuestion"; // draws Video elements (S34)
import "@/features/information/InformationQuestion"; // draws Information, cleaned (S35)
import "survey-core/survey-core.css";
import {
  actionWords, applyEarlier, chapterConditionOf, combineChapters, computeOutputs, earlierKeysUsed, forPatient, givenEarlier, showBands, localesIn, refreshClinicalSummaries, registerClinicalProperties,
  ruleSetOf, setFiredRules, setViewer, withoutSurveyHtml, LANGUAGES, type Answers, type ChapterJson, type ComputedOutputs, type FiredRule, type RuleSet, type Viewer,
} from "@poised/clinical";
import { api } from "@/lib/api";
import { clinicianModel } from "./clinicianView";
import { SetCondition } from "./SetCondition";
import { EarlierAnswers, useEarlierSets } from "./EarlierAnswers";
import { TestCases } from "./TestCases";
import { canChange, useHospital } from "@/lib/auth";
import { showUnits } from "./units";

registerClinicalProperties();

type ChapterDetail = {
  id: string;
  name: string;
  content: object;
  revision: number;
  versionStatus: string;
  ownerHospitalId: string | null; // null: the trust's questionnaire
  questionnaireId: string;
  questionnaireName: string;
};

export type SamplePatient = { name: string; age: number | ""; sex: string | null };

const DEVICES = { phone: 390, tablet: 820, desktop: undefined } as const;
type Device = keyof typeof DEVICES;

export default function PreviewPage({ base, chapterId }: { base: string; chapterId: string }) {
  const [chapter, setChapter] = useState<ChapterDetail | null>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    api<ChapterDetail>(`${base}/chapters/${chapterId}`)
      .then(setChapter)
      .catch((e: Error) => setError(e.message));
  }, [base, chapterId]);

  if (error) return <Alert color="red" m="md">{error}</Alert>;
  if (!chapter) {
    return (
      <Center h="50vh">
        <Loader />
      </Center>
    );
  }
  return <Preview base={base} chapter={chapter} />;
}

function Preview({ base, chapter }: { base: string; chapter: ChapterDetail }) {
  // Saving or deleting test cases changes the Question Set: only for a draft, and only for those
  // who may change this questionnaire (the trust's needs a whole-trust role; S08 answers, 5).
  const readOnly = !canChange(useHospital(), chapter.ownerHospitalId, "super_clinician")
    ? "Read only: only super clinicians and admins of the whole trust can change test cases."
    : chapter.versionStatus !== "draft"
      ? "Read only: test cases can be changed only in a draft version."
      : null;
  const canSave = !readOnly;
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
  // Scores here that use earlier sets' answers (S38): the keys, the earlier sets, and the answers the author gives.
  const earlierKeys = useMemo(() => earlierKeysUsed(content), [content]);
  const earlierSets = useEarlierSets(base, chapter.questionnaireId, chapter.id, earlierKeys.length > 0 || !!chapterConditionOf(content));
  const earlier = useMemo(() => (earlierSets ? combineChapters(earlierSets) : null), [earlierSets]);
  const [earlierValues, setEarlierValues] = useState<Answers>({});
  // Only the earlier sets this sample patient would see count (test report 2, F-1).
  const sampleAge = patient.age === "" ? undefined : patient.age;
  const given = useMemo(
    () => (earlierSets ? givenEarlier(earlierSets, earlierValues, { age: sampleAge, sex: patient.sex ?? undefined }, viewer) : null),
    [earlierSets, earlierValues, sampleAge, patient.sex, viewer],
  );

  // Rebuilt when the viewer changes. Patients get the JSON without clinician-only content, with
  // each Section on its own screen and without page titles (VEW-01, STR-03). Clinicians see whole
  // pages, with each page's Clinical summary box (QT-09).
  const model = useMemo(() => {
    let m: Model;
    if (viewer === "patient") {
      m = new Model(forPatient(withoutSurveyHtml(content)));
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

  // Sample patient details are survey variables: {patientName}, {patientAge}, {patientSex}. The
  // earlier sets' answers given here are too (S38).
  useEffect(() => {
    model.setVariable("patientName", patient.name);
    model.setVariable("patientAge", patient.age === "" ? undefined : patient.age);
    model.setVariable("patientSex", patient.sex ?? undefined);
    if (given) {
      // A key whose set is hidden now must lose the answer it had a moment ago.
      const cleared = Object.fromEntries(Object.keys(earlierValues).map((k) => [k, undefined]));
      applyEarlier(model, { ...given.earlier, values: { ...cleared, ...given.earlier.values } });
    }
    setResult(computeOutputs(model));
  }, [model, patient, given, earlierValues]);

  useEffect(() => {
    const update = () => {
      answers.current = model.data;
      refreshClinicalSummaries(model, content);
      setResult(computeOutputs(model));
    };
    model.onValueChanged.add(update);
    update();
    return () => model.onValueChanged.remove(update);
  }, [model]);

  // A fresh survey with no answers: SurveyJS's clear() also drops the survey variables (the sample
  // patient and the earlier sets' answers, S38 test report 1, F-1) and leaves the summary boxes as
  // they were, so the survey is built again and the effects above set them all.
  const restart = () => {
    answers.current = {};
    setRestored((n) => n + 1);
  };
  // The HQ's ASA rules (S40), for "Rules that fire" with this set's answers alone.
  const [rules, setRules] = useState<RuleSet | null>(null);
  useEffect(() => {
    api<{ questionnaire: { rules: unknown } }>(`${base}/questionnaires/${chapter.questionnaireId}`)
      .then((d) => setRules(ruleSetOf(d.questionnaire.rules)))
      .catch(() => setRules(null));
  }, [base, chapter.questionnaireId]);
  // Worked out as the rules see the answers on send, in the clinician's view, whichever view is shown
  // (test report 1, F-1: a clinician-only score counts). `result` changes with every answer.
  const fired = useMemo(
    () => (rules ? setFiredRules(rules, content, model.data, { name: patient.name, age: sampleAge, sex: patient.sex ?? undefined }, given?.earlier) : null),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rules, result, content, model, sampleAge, patient, given],
  );
  const locales = localesIn(content);
  const width = DEVICES[device];
  return (
    <Box p="md">
      <Group justify="space-between" mb="md">
        <Group gap="xs">
          <Anchor component={Link} to={`${base}/questionnaires/${chapter.questionnaireId}`} size="sm">
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
          <Button size="xs" variant="light" component={Link} to={`${base}/questionnaires/${chapter.questionnaireId}?set=${chapter.id}`}>{canSave ? "Edit" : "Open in editor"}</Button>
        </Group>
      </Group>
      <Grid>
        <Grid.Col span={{ base: 12, md: 8 }}>
          {earlierKeys.length > 0 && <EarlierAnswers earlier={earlier} keys={earlierKeys} values={earlierValues} hidden={given?.hidden ?? []} onChange={setEarlierValues} />}
          <SetCondition
            earlier={earlier}
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
            <OutputsPanel result={result} fired={fired} />
            <TestCases
              base={base}
              readOnly={readOnly}
              chapterId={chapter.id}
              content={content}
              revision={revision}
              viewer={viewer}
              patient={patient}
              model={model}
              earlier={earlierSets}
              earlierValues={earlierValues}
              onSaved={(c, r) => {
                setContent(c);
                setRevision(r);
              }}
              onLoad={(tc) => {
                answers.current = tc.answers;
                setEarlierValues(tc.earlier ?? {});
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

/** `fired`: the HQ's rules that fire for these answers alone (S40); null when the version has no rules. */
function OutputsPanel({ result, fired }: { result: ComputedOutputs; fired: FiredRule[] | null }) {
  const { outputs, suggestedAsa, scores = [] } = result;
  const byCategory = new Map<string, string[]>();
  for (const o of outputs) {
    if (!o.output.note || !o.noteText) continue;
    const category = o.output.note.category; // "" when the note was saved without one
    const list = byCategory.get(category) ?? [];
    list.push(o.noteText);
    byCategory.set(category, list);
  }
  // computeOutputs normalizes each output, so `codes` is always a list (S08 test reports 2 and 3).
  const codes = outputs.flatMap((o) => o.output.codes.map((c) => ({ ...c, from: o.questionTitle })));
  const flags = outputs.filter((o) => o.output.flag);

  return (
    <Card withBorder data-testid="outputs-panel">
      <Title order={5} mb="xs">Clinical outputs</Title>
      {outputs.length === 0 && <Text size="sm" c="dimmed">The current answers produce no outputs.</Text>}
      <Stack gap="sm">
        {scores.length > 0 && (
          <div data-testid="preview-scores">
            <Text size="sm" fw={600}>Scores</Text>
            {scores.map((s) => (
              <Text size="sm" key={s.questionId} component="div">
                {s.title}: {s.incomplete ? <Badge size="xs" color="gray" variant="light">Incomplete</Badge> : s.value === undefined ? "—" : `${s.value}${s.band ? ` (${s.band.label})` : ""}`}
              </Text>
            ))}
          </div>
        )}
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
          <div data-testid="preview-asa">
            <Text size="sm" fw={600}>Suggested ASA</Text>
            <Text size="sm">ASA {suggestedAsa.grade}{suggestedAsa.emergency ? "E" : ""}</Text>
            {/* Why (S39): each disclosure at that grade, and any emergency one. */}
            {suggestedAsa.reasons.map((r) => (
              <Text size="xs" c="dimmed" key={r.text}>{r.text}</Text>
            ))}
          </div>
        )}
        {fired && (
          <div data-testid="preview-rules">
            <Text size="sm" fw={600}>Rules that fire</Text>
            <Text size="xs" c="dimmed">For this Question Set&apos;s answers alone. Test the whole HQ from its Rules page.</Text>
            {fired.length === 0 && <Text size="sm" c="dimmed">None</Text>}
            {fired.map((f) => (
              <Text size="sm" key={f.rule.id}>
                <b>{f.rule.name}</b>: {[...f.actions.map(actionWords), ...(f.held ? ["no auto-triage: no ASA grade"] : [])].join("; ")}
              </Text>
            ))}
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
