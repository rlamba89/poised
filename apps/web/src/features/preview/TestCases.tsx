"use client";
// Test cases (PRV-05/06/07/08): save the preview's answers with the outputs they produce, then
// re-run them all after changes. Stored in the chapter JSON, so they carry over to new versions.
import { useState } from "react";
import { ActionIcon, Alert, Badge, Button, Card, Group, Stack, Text, TextInput, Title, Tooltip } from "@mantine/core";
import { IconPlayerPlay, IconTrash } from "@tabler/icons-react";
import type { SurveyModel } from "survey-core";
import {
  computeOutputs, newId, outputLines, runTestCase, setTestCases, testCasesOf, type ChapterJson, type TestCase, type TestResult,
  type Viewer,
} from "@poised/clinical";
import { api, ApiError } from "@/lib/api";
import type { SamplePatient } from "./PreviewPage";

export function TestCases(props: {
  base: string;
  chapterId: string;
  content: ChapterJson;
  revision: number;
  viewer: Viewer;
  patient: SamplePatient;
  model: SurveyModel;
  onSaved: (content: ChapterJson, revision: number) => void;
  onLoad: (tc: TestCase) => void;
}) {
  const { content, revision, viewer, patient, model } = props;
  const cases = testCasesOf(content);
  const [name, setName] = useState("");
  const [results, setResults] = useState<Record<string, TestResult>>({});
  const [open, setOpen] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const save = async (next: TestCase[]) => {
    setSaving(true);
    setError("");
    const doc = setTestCases(content, next);
    try {
      const res = await api<{ revision: number }>(`${props.base}/chapters/${props.chapterId}/content`, {
        method: "PUT",
        body: JSON.stringify({ content: doc, revision }),
      });
      props.onSaved(doc, res.revision);
    } catch (e) {
      setError(e instanceof ApiError && e.status === 409 ? "This Question Set was changed in the editor. Reload the preview, then save the test case again." : (e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const add = () => {
    const tc: TestCase = {
      id: newId("t", 6),
      name: name.trim(),
      viewer,
      patient: { name: patient.name, age: patient.age === "" ? undefined : patient.age, sex: patient.sex ?? undefined },
      answers: structuredClone(model.data),
      expected: outputLines(computeOutputs(model)),
    };
    setName("");
    save([...cases, tc]);
  };
  const runAll = () => setResults(Object.fromEntries(cases.map((tc) => [tc.id, runTestCase(content, tc)])));
  const ran = Object.keys(results).length > 0;
  const failed = cases.filter((tc) => results[tc.id] && !results[tc.id].pass).length;

  return (
    <Card withBorder data-testid="test-cases">
      <Group justify="space-between" mb="xs">
        <Title order={5}>Test cases</Title>
        {cases.length > 0 && (
          <Button size="compact-xs" variant="light" leftSection={<IconPlayerPlay size={12} />} onClick={runAll}>
            Run all
          </Button>
        )}
      </Group>
      {ran && (
        <Alert color={failed ? "red" : "green"} p="xs" mb="xs">
          <Text size="sm">{failed ? `${failed} of ${cases.length} failed` : `All ${cases.length} passed`}</Text>
        </Alert>
      )}
      {error && <Alert color="red" p="xs" mb="xs"><Text size="sm">{error}</Text></Alert>}
      <Stack gap={6}>
        {cases.length === 0 && <Text size="sm" c="dimmed">Answer the form, then save the answers and the outputs they produce as a test case.</Text>}
        {cases.map((tc) => {
          const r = results[tc.id];
          return (
            <div key={tc.id}>
              <Group justify="space-between" wrap="nowrap" gap={4}>
                <Group gap={6} wrap="nowrap" style={{ minWidth: 0, cursor: r && !r.pass ? "pointer" : undefined }} onClick={() => setOpen(open === tc.id ? null : tc.id)}>
                  {r && <Badge size="xs" color={r.pass ? "green" : "red"}>{r.pass ? "pass" : "fail"}</Badge>}
                  <Text size="sm" truncate>{tc.name}</Text>
                  <Badge size="xs" variant="light" color="gray">{tc.viewer}</Badge>
                </Group>
                <Group gap={2} wrap="nowrap">
                  <Button size="compact-xs" variant="subtle" onClick={() => props.onLoad(tc)}>Load</Button>
                  <Tooltip label="Delete test case">
                    <ActionIcon size="sm" variant="subtle" color="red" aria-label={`Delete ${tc.name}`} disabled={saving} onClick={() => save(cases.filter((x) => x.id !== tc.id))}>
                      <IconTrash size={13} />
                    </ActionIcon>
                  </Tooltip>
                </Group>
              </Group>
              {r && !r.pass && open === tc.id && (
                <Stack gap={2} mt={4} pl="sm">
                  {r.missing.map((x) => (
                    <Text key={`m${x}`} size="xs" c="red">− expected, not produced: {x}</Text>
                  ))}
                  {r.unexpected.map((x) => (
                    <Text key={`u${x}`} size="xs" c="orange.8">+ produced, not expected: {x}</Text>
                  ))}
                </Stack>
              )}
            </div>
          );
        })}
      </Stack>
      <Group gap="xs" mt="sm" wrap="nowrap">
        <TextInput size="xs" style={{ flex: 1 }} placeholder="Name, e.g. Smoker with COPD" value={name} onChange={(e) => setName(e.currentTarget.value)} aria-label="Test case name" />
        <Button size="xs" disabled={!name.trim() || saving} onClick={add}>Save answers</Button>
      </Group>
      <Text size="xs" c="dimmed" mt={6}>Made-up answers only. Never enter a real patient&apos;s details.</Text>
    </Card>
  );
}
