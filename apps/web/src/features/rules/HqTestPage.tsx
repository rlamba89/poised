// Testing the HQ's rules on the whole HQ (S40 answers 4): the author answers the patient's Question
// Sets as one made-up patient (set by set, as the patient sees them), and sees live which sets are
// shown, the suggested ASA with its reasons, and the rules that fire. "Save as test case" keeps the
// answers (by question key) with those lines; the Rules page and Publish run them again after every
// change. `?case=<id>` opens a saved case.
import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { Alert, Anchor, Badge, Box, Button, Card, Center, Grid, Group, Loader, NumberInput, Select, Stack, Tabs, Text, TextInput, Title } from "@mantine/core";
import { Model } from "survey-core";
import { Survey } from "survey-react-ui";
import "@/features/video/VideoQuestion"; // draws Video elements (S34)
import "@/features/information/InformationQuestion"; // draws Information, cleaned (S35)
import "survey-core/survey-core.css";
import {
  allElements, answersBySet, firedWords, earlierContexts, hqRuleResult, modelFor, newId, registerClinicalProperties, showBands, type Answers, type HqTestCase, type RuleSet,
} from "@poised/clinical";
import { showUnits } from "@/features/preview/units";
import { useHq } from "./useHq";

registerClinicalProperties();

const NO_RULES: RuleSet = { noGradeAutoTriage: true, rules: [], testCases: [] };
type Sample = { name: string; age: number | ""; sex: string | null };

export function HqTestPage({ base, questionnaireId }: { base: string; questionnaireId: string }) {
  const { hq, error: loadError, save, saving: busy } = useHq(base, questionnaireId);
  const [params, setParams] = useSearchParams();
  const caseId = params.get("case");
  const [patient, setPatient] = useState<Sample>({ name: "Sam Sample", age: 54, sex: "female" });
  const [procedureId, setProcedureId] = useState<string | null>(null);
  const [answers, setAnswers] = useState<Answers>({});
  const [selected, setSelected] = useState<string | null>(null);
  const [restart, setRestart] = useState(0);
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const loadedCase = useRef<string | null>(null);

  const sets = useMemo(() => (hq ? hq.sets.filter((s) => s.audience === "patient") : []), [hq]);
  const rules = hq?.rules ?? NO_RULES;
  const opened = caseId ? rules.testCases.find((t) => t.id === caseId) : undefined;
  // A saved case, once: its patient, procedure and answers.
  useEffect(() => {
    if (!opened || loadedCase.current === opened.id) return;
    loadedCase.current = opened.id;
    setPatient({ name: opened.patient.name, age: opened.patient.age ?? "", sex: opened.patient.sex ?? null });
    setProcedureId(opened.procedureId ?? null);
    setAnswers(opened.answers);
    setName(opened.name); // saving it again replaces it (test report 1, E-2)
    setRestart((n) => n + 1);
  }, [opened]);

  const sample = useMemo(() => ({ name: patient.name, age: patient.age === "" ? undefined : patient.age, sex: patient.sex ?? undefined }), [patient]);
  const result = useMemo(() => (hq ? hqRuleResult(hq.sets, rules, answers, sample, procedureId ?? undefined) : null), [hq, rules, answers, sample, procedureId]);
  const current = selected && sets.some((s) => s.id === selected) ? selected : sets[0]?.id;

  // The selected set's survey, with the sets before it (scores that use them, S38). Rebuilt when the
  // set, the sample patient or a loaded case changes; each answer goes back into the HQ's answers.
  const answersRef = useRef(answers);
  answersRef.current = answers;
  const model = useMemo(() => {
    const set = sets.find((s) => s.id === current);
    if (!set) return null;
    const bySet = answersBySet(sets, answersRef.current);
    const m = modelFor(set.content, "patient", sample, earlierContexts(sets, bySet, "patient", sample).get(set.id));
    showUnits(m);
    showBands(m);
    m.showPageTitles = false;
    m.data = bySet[set.id] ?? {};
    m.completedHtml = "<p>End of this Question Set. Choose the next one above.</p>";
    return m;
  }, [sets, current, sample, restart]);
  useEffect(() => {
    if (!model) return;
    const set = sets.find((s) => s.id === current)!;
    const own = new Set(allElements(set.content).map(({ el }) => el.name));
    const update = () => {
      const others = Object.fromEntries(Object.entries(answersRef.current).filter(([k]) => !own.has(k)));
      setAnswers({ ...others, ...model.data });
    };
    model.onValueChanged.add(update);
    return () => model.onValueChanged.remove(update);
  }, [model, sets, current]);
  // A replaced survey is disposed once the new one is showing (not in the effect's cleanup, which
  // React's development mode runs straight away), as the Preview does.
  const shownModel = useRef<Model | null>(null);
  useEffect(() => {
    const old = shownModel.current;
    shownModel.current = model;
    if (old && old !== model) old.dispose();
  }, [model]);

  if (loadError && !hq) return <Alert color="red" m="md">{loadError}</Alert>;
  if (!hq || !result) return <Center h="50vh"><Loader /></Center>;
  const q = hq.detail.questionnaire;
  const rulesPath = `${base}/questionnaires/${questionnaireId}/rules`;
  const canSave = !hq.readOnly && !!hq.rules;
  const replacing = !!hq.rules?.testCases.some((t) => t.name.trim().toLowerCase() === name.trim().toLowerCase() && name.trim() !== "");
  const verdict = opened && { pass: opened.expected.length === result.lines.length && opened.expected.every((x) => result.lines.includes(x)) };

  const saveCase = async () => {
    if (!hq.rules) return;
    setSaving(true);
    const given = Object.fromEntries(Object.entries(answers).filter(([, v]) => v !== undefined && v !== ""));
    const tc: HqTestCase = {
      id: newId("t", 6), name: name.trim(), patient: sample, answers: structuredClone(given), expected: result.lines,
      ...(procedureId ? { procedureId } : {}),
    };
    // A case of the same name is replaced where it is, keeping its id (names are unique, as Go checks).
    const same = hq.rules.testCases.find((t) => t.name.trim().toLowerCase() === tc.name.toLowerCase());
    if (same) tc.id = same.id;
    const testCases = same ? hq.rules.testCases.map((t) => (t.id === same.id ? tc : t)) : [...hq.rules.testCases, tc];
    const problem = await save({ ...hq.rules, testCases });
    setSaving(false);
    setError(problem);
    if (!problem) {
      setName(tc.name);
      loadedCase.current = tc.id;
      setParams({ case: tc.id }, { replace: true });
    }
  };

  return (
    <Box p="md">
      <Group justify="space-between" mb="md">
        <Group gap="xs">
          <Anchor component={Link} to={rulesPath} size="sm">← Rules</Anchor>
          <Text fw={600}>/ {q.name} · Test the whole HQ</Text>
          {opened && <Badge variant="light">{opened.name}</Badge>}
        </Group>
        <Button size="xs" variant="default" onClick={() => {
          setAnswers({});
          setParams({}, { replace: true });
          loadedCase.current = null;
          setRestart((n) => n + 1);
        }}>Clear answers</Button>
      </Group>
      <Grid>
        <Grid.Col span={{ base: 12, md: 8 }}>
          {sets.length === 0 ? (
            <Alert color="gray">This HQ has no Question Set for the patient.</Alert>
          ) : (
            <Tabs value={current ?? null} onChange={setSelected} mb="sm">
              <Tabs.List>
                {sets.map((s) => (
                  <Tabs.Tab key={s.id} value={s.id} rightSection={result.shown.includes(s.id) ? undefined : <Badge size="xs" color="gray" variant="light">not shown</Badge>}>
                    {s.name}
                  </Tabs.Tab>
                ))}
              </Tabs.List>
            </Tabs>
          )}
          {current && !result.shown.includes(current) && (
            <Alert color="orange" mb="sm" p="xs"><Text size="sm">A condition hides this Question Set for these answers: what's typed here doesn't count.</Text></Alert>
          )}
          {model && <Card withBorder padding={0}><Survey model={model} /></Card>}
        </Grid.Col>
        <Grid.Col span={{ base: 12, md: 4 }}>
          <Stack>
            <Card withBorder>
              <Title order={5} mb="xs">Sample patient</Title>
              <Text size="xs" c="dimmed" mb="xs">Made-up details only. Never enter a real patient.</Text>
              <Stack gap="xs">
                <TextInput label="Name" value={patient.name} onChange={(e) => setPatient({ ...patient, name: e.currentTarget.value })} />
                <Group grow>
                  <NumberInput label="Age" min={0} max={120} value={patient.age} onChange={(v) => setPatient({ ...patient, age: v === "" ? "" : Number(v) })} />
                  <Select label="Sex" data={[{ value: "female", label: "Female" }, { value: "male", label: "Male" }]} value={patient.sex}
                    onChange={(v) => setPatient({ ...patient, sex: v })} clearable />
                </Group>
                <Select label="Procedure" placeholder="None" clearable searchable data={hq.procedures.map((p) => ({ value: p.id, label: p.name }))}
                  value={procedureId} onChange={setProcedureId} />
              </Stack>
            </Card>
            <Card withBorder data-testid="hq-result">
              <Title order={5} mb="xs">What the rules see</Title>
              <Text size="sm" fw={600}>{result.asa ? `Suggested ASA ${result.asa.grade}${result.asa.emergency ? "E" : ""}` : "No suggested ASA"}</Text>
              {result.asa?.reasons.map((r) => <Text size="xs" c="dimmed" key={r.text}>{r.text}</Text>)}
              <Text size="sm" fw={600} mt="sm">Rules that fire</Text>
              <div data-testid="hq-fired">
                {result.fired.length === 0 && <Text size="sm" c="dimmed">None</Text>}
                {result.fired.map((f) => (
                  <Text size="sm" key={f.rule.id}>
                    <b>{f.rule.name}</b>: {firedWords(f)}
                  </Text>
                ))}
              </div>
              {!hq.rules && <Text size="sm" c="dimmed" mt="xs">This version has no rules.</Text>}
            </Card>
            <Card withBorder data-testid="hq-save-case">
              <Title order={5} mb="xs">Test case</Title>
              {opened && verdict && (
                <Alert color={verdict.pass ? "green" : "red"} p="xs" mb="xs">
                  <Text size="sm">{verdict.pass ? `“${opened.name}” passes` : `“${opened.name}” fails: it expected:`}</Text>
                  {!verdict.pass && opened.expected.map((x) => <Text key={x} size="xs">{x}</Text>)}
                </Alert>
              )}
              {canSave ? (
                // A form, so Enter saves (test report 1, E-4).
                <form onSubmit={(e) => { e.preventDefault(); if (name.trim() && !saving && !busy) saveCase(); }}>
                  <Group gap="xs" wrap="nowrap">
                    <TextInput size="xs" style={{ flex: 1 }} placeholder="Name, e.g. Fit and well" aria-label="Test case name" value={name}
                      onChange={(e) => setName(e.currentTarget.value)} />
                    <Button type="submit" size="xs" disabled={!name.trim() || saving || busy} loading={saving}>{replacing ? "Replace test case" : "Save as test case"}</Button>
                  </Group>
                  {replacing && <Text size="xs" c="dimmed" mt={4}>A test case has this name: saving replaces it with these answers and lines.</Text>}
                </form>
              ) : (
                <Text size="sm" c="dimmed">{hq.readOnly ?? "This version has no rules to test."}</Text>
              )}
              {error && <Alert color="red" p="xs" mt="xs"><Text size="sm">{error}</Text></Alert>}
              <Text size="xs" c="dimmed" mt={6}>Saves the answers with the lines above; it fails when a change gives other lines.</Text>
            </Card>
          </Stack>
        </Grid.Col>
      </Grid>
    </Box>
  );
}
