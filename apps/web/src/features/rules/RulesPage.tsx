// The HQ's Rules page (S40; ASA-02, ASA-03, ASA-04): the rules in plain words, with add, edit,
// delete and reorder; the setting for patients whose answers give no ASA grade (S40 answers 2);
// and the whole-HQ test cases (S40 answers 4). Every change saves at once (PUT …/rules, with the
// revision). Read-only for those who can't change the HQ, and on a published version.
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { ActionIcon, Alert, Anchor, Badge, Box, Button, Card, Center, Checkbox, Group, Loader, Stack, Text, Title, Tooltip } from "@mantine/core";
import { IconArrowDown, IconArrowUp, IconPencil, IconTrash } from "@tabler/icons-react";
import {
  codesIn, newId, ruleProblems, ruleWords, runHqTestCase, scoresWithBands, type AsaRule, type RuleSet,
} from "@poised/clinical";
import { RuleForm } from "./RuleForm";
import { useHq } from "./useHq";

const blankRule = (): AsaRule => ({ id: newId("r", 6), name: "", when: { asa: { op: ">=", grade: 3 } }, then: [] });

export function RulesPage({ base, questionnaireId }: { base: string; questionnaireId: string }) {
  // `busy`: a change is saving; everything that saves waits (one save at a time, test report 3, F-1).
  const { hq, error: loadError, save, saving: busy } = useHq(base, questionnaireId);
  const [editing, setEditing] = useState<string | null>(null); // a rule's id, or "new"
  const [deleting, setDeleting] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState<string | null>(null);
  // Every whole-HQ test case is run against the rules as they are, after each change.
  const results = useMemo(
    () => (hq?.rules ? Object.fromEntries(hq.rules.testCases.map((tc) => [tc.id, runHqTestCase(hq.sets, hq.rules!, tc)])) : {}),
    [hq],
  );

  if (loadError && !hq) return <Alert color="red" m="md">{loadError}</Alert>;
  if (!hq) return <Center h="50vh"><Loader /></Center>;
  const q = hq.detail.questionnaire;
  const rules = hq.rules;
  const editorPath = `${base}/questionnaires/${questionnaireId}`;
  const testPath = `${editorPath}/rules/test`;
  const readOnly = hq.readOnly;
  const docs = hq.sets.map((s) => s.content);

  const change = async (next: RuleSet): Promise<string> => {
    const problem = await save(next);
    setError(problem);
    return problem;
  };
  const move = (i: number, delta: number) => {
    if (!rules) return;
    const list = [...rules.rules];
    [list[i], list[i + delta]] = [list[i + delta], list[i]];
    change({ ...rules, rules: list });
  };
  const saveRule = async (r: AsaRule): Promise<string> => {
    if (!rules) return "";
    const exists = rules.rules.some((x) => x.id === r.id);
    const problem = await save({ ...rules, rules: exists ? rules.rules.map((x) => (x.id === r.id ? r : x)) : [...rules.rules, r] });
    if (!problem) setEditing(null);
    return problem;
  };
  const problems = rules ? ruleProblems(rules, hq.sets) : [];
  const failed = rules ? rules.testCases.filter((tc) => results[tc.id] && !results[tc.id].pass).length : 0;

  return (
    <Box p="md" maw={980} mx="auto">
      <Group gap="xs" mb="md">
        <Anchor component={Link} to={editorPath} size="sm">← {q.name}</Anchor>
        <Text fw={600}>/ Rules</Text>
        <Badge variant="light" color={q.status === "published" ? "green" : "gray"} size="sm">v{q.versionNo} · {q.status}</Badge>
        {readOnly && <Badge variant="outline" color="gray" size="sm">Read only</Badge>}
      </Group>
      <Text size="sm" c="dimmed" mb="md">
        Rules run once, when the patient sends this HQ, on the suggested ASA from the patient&apos;s answers. Every rule that matches fires, in this order.
        A rule never auto-triages a patient whose answers have a red or amber flag or an Incomplete score, or when a rule requires the
        anaesthetist&apos;s ASA before Ready for admission: the patient goes to Ready for review instead.
      </Text>
      {readOnly && <Alert color="gray" mb="md" p="xs"><Text size="sm">{readOnly}</Text></Alert>}
      {error && <Alert color="red" mb="md" p="xs" data-testid="rules-error"><Text size="sm">{error}</Text></Alert>}
      {!rules && <Alert color="gray" mb="md">This version was published before HQs had rules, so it has none. A new version gets the default rules.</Alert>}

      {rules && (
        <Stack gap="lg">
          <Card withBorder>
            <Checkbox
              label="Patients whose answers give no ASA grade (usually fit and well) count as ASA I, so the rules can auto-triage them"
              description={rules.noGradeAutoTriage ? undefined : "Off: they're never auto-triaged; a clinician reviews them."}
              checked={rules.noGradeAutoTriage}
              disabled={!!readOnly || busy}
              onChange={(e) => change({ ...rules, noGradeAutoTriage: e.currentTarget.checked })}
            />
            {/* S42 answers 2: off by default, so the hospital's status message goes as soon as the rules move the patient on. */}
            <Checkbox
              mt="sm"
              label="When a rule auto-triages a patient, wait for a clinician to click Checked before the patient's status message goes"
              description={rules.messageAfterCheck
                ? "On: the hospital's message (for example Ready for admission) goes once a clinician has checked the episode on the Auto-triaged list."
                : "Off: the hospital's message goes straight away. The episode is still on the Auto-triaged list for a clinician to check."}
              checked={!!rules.messageAfterCheck}
              disabled={!!readOnly || busy}
              onChange={(e) => change({ ...rules, messageAfterCheck: e.currentTarget.checked })}
            />
          </Card>

          <div>
            <Group justify="space-between" mb="xs">
              <Title order={4}>Rules</Title>
              {!readOnly && editing !== "new" && <Button size="xs" disabled={busy} onClick={() => setEditing("new")}>Add rule</Button>}
            </Group>
            {problems.length > 0 && (
              <Alert color="orange" mb="sm" p="xs" title="These rules can't fire, and block Publish">
                {problems.map((p) => <Text key={p} size="sm">· {p}</Text>)}
              </Alert>
            )}
            <Stack gap="xs" data-testid="rules-list">
              {rules.rules.length === 0 && <Text size="sm" c="dimmed">No rules: nothing happens automatically when the patient sends this HQ.</Text>}
              {rules.rules.map((r, i) =>
                editing === r.id ? (
                  <RuleForm key={r.id} initial={r} scores={scoresWithBands(docs)} codes={codesIn(docs)} procedures={hq.procedures}
                    busy={busy} onSave={saveRule} onCancel={() => setEditing(null)} />
                ) : (
                  <Card key={r.id} withBorder padding="sm" data-testid="rule">
                    <Group justify="space-between" wrap="nowrap" align="start">
                      <div style={{ minWidth: 0 }}>
                        <Text fw={600} size="sm">{i + 1}. {r.name}</Text>
                        <Text size="sm">{ruleWords(r, hq.names)}</Text>
                      </div>
                      {!readOnly && (
                        <Group gap={2} wrap="nowrap">
                          <Tooltip label="Move up"><ActionIcon variant="subtle" aria-label={`Move ${r.name} up`} disabled={i === 0 || busy} onClick={() => move(i, -1)}><IconArrowUp size={14} /></ActionIcon></Tooltip>
                          <Tooltip label="Move down"><ActionIcon variant="subtle" aria-label={`Move ${r.name} down`} disabled={i === rules.rules.length - 1 || busy} onClick={() => move(i, 1)}><IconArrowDown size={14} /></ActionIcon></Tooltip>
                          <Tooltip label="Edit"><ActionIcon variant="subtle" aria-label={`Edit ${r.name}`} disabled={busy} onClick={() => setEditing(r.id)}><IconPencil size={14} /></ActionIcon></Tooltip>
                          <Tooltip label="Delete"><ActionIcon variant="subtle" color="red" aria-label={`Delete ${r.name}`} disabled={busy} onClick={() => setDeleting(r.id)}><IconTrash size={14} /></ActionIcon></Tooltip>
                        </Group>
                      )}
                    </Group>
                    {deleting === r.id && (
                      <Group mt="xs" gap="xs">
                        <Text size="sm">Delete “{r.name}”?</Text>
                        <Button size="compact-xs" color="red" disabled={busy} onClick={async () => {
                          if (!(await change({ ...rules, rules: rules.rules.filter((x) => x.id !== r.id) }))) setDeleting(null);
                        }}>Delete</Button>
                        <Button size="compact-xs" variant="default" disabled={busy} onClick={() => setDeleting(null)}>Keep it</Button>
                      </Group>
                    )}
                  </Card>
                ),
              )}
              {editing === "new" && (
                <RuleForm initial={blankRule()} scores={scoresWithBands(docs)} codes={codesIn(docs)} procedures={hq.procedures}
                  busy={busy} onSave={saveRule} onCancel={() => setEditing(null)} />
              )}
            </Stack>
          </div>

          <div data-testid="hq-test-cases">
            <Group justify="space-between" mb="xs">
              <Title order={4}>Test cases for the whole HQ</Title>
              <Button size="xs" variant="default" component={Link} to={testPath}>Test the whole HQ</Button>
            </Group>
            <Text size="sm" c="dimmed" mb="xs">
              Answer the patient&apos;s Question Sets as one made-up patient, then save the suggested ASA and the rules that fire. Each case runs again after every change here and in the editor; Publish is blocked while one fails.
            </Text>
            {rules.testCases.length > 0 && (
              <Alert color={failed ? "red" : "green"} p="xs" mb="xs">
                <Text size="sm">{failed ? `${failed} of ${rules.testCases.length} failed` : `All ${rules.testCases.length} passed`}</Text>
              </Alert>
            )}
            <Stack gap={6}>
              {rules.testCases.length === 0 && <Text size="sm" c="dimmed">No test cases yet.</Text>}
              {rules.testCases.map((tc) => {
                const r = results[tc.id];
                return (
                  <Card key={tc.id} withBorder padding="xs" data-testid="hq-test-case">
                    <Group justify="space-between" wrap="nowrap">
                      <Group gap={6} wrap="nowrap" style={{ minWidth: 0, cursor: r && !r.pass ? "pointer" : undefined }} onClick={() => setOpen(open === tc.id ? null : tc.id)}>
                        {r && <Badge size="xs" color={r.pass ? "green" : "red"}>{r.pass ? "pass" : "fail"}</Badge>}
                        <Text size="sm" truncate>{tc.name}</Text>
                      </Group>
                      <Group gap={2} wrap="nowrap">
                        <Button size="compact-xs" variant="subtle" component={Link} to={`${testPath}?case=${tc.id}`}>Open</Button>
                        {!readOnly && (
                          <ActionIcon size="sm" variant="subtle" color="red" aria-label={`Delete ${tc.name}`} disabled={busy}
                            onClick={() => setDeleting(tc.id)}>
                            <IconTrash size={13} />
                          </ActionIcon>
                        )}
                      </Group>
                    </Group>
                    {/* It asks first, as a rule's bin does (test report 1, E-3). */}
                    {deleting === tc.id && (
                      <Group mt="xs" gap="xs">
                        <Text size="sm">Delete “{tc.name}”?</Text>
                        <Button size="compact-xs" color="red" disabled={busy} onClick={async () => {
                          if (!(await change({ ...rules, testCases: rules.testCases.filter((x) => x.id !== tc.id) }))) setDeleting(null);
                        }}>Delete</Button>
                        <Button size="compact-xs" variant="default" disabled={busy} onClick={() => setDeleting(null)}>Keep it</Button>
                      </Group>
                    )}
                    <Stack gap={0} mt={4} pl="sm">
                      {tc.expected.map((x) => <Text key={x} size="xs" c="dimmed">{x}</Text>)}
                    </Stack>
                    {r && !r.pass && open === tc.id && (
                      <Stack gap={2} mt={4} pl="sm">
                        {r.missing.map((x) => <Text key={`m${x}`} size="xs" c="red">− expected, not produced: {x}</Text>)}
                        {r.unexpected.map((x) => <Text key={`u${x}`} size="xs" c="orange.8">+ produced, not expected: {x}</Text>)}
                      </Stack>
                    )}
                  </Card>
                );
              })}
            </Stack>
          </div>
        </Stack>
      )}
    </Box>
  );
}
