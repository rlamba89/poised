// Adding or editing one ASA rule (S40, ASA-03): its name, its "when" (the suggested ASA compared with a
// grade, and the optional extras) and its "then" (actions from a fixed list). Go checks the shape when
// the rules are saved, and its refusal is shown here in its own words, with the form left open.
import { useState } from "react";
import { ActionIcon, Alert, Button, Card, Checkbox, Group, MultiSelect, Select, Stack, Text, TextInput } from "@mantine/core";
import { IconTrash } from "@tabler/icons-react";
import {
  ASA_GRADES, notBandChoices, RULE_TEAMS, SUGGESTABLE, TRIAGE_TARGETS, type AsaOp, type AsaRule, type Band, type CodeRef, type RuleAction,
} from "@poised/clinical";
import type { OfferedProcedure } from "@/features/procedures/procedures";

const OPS: { value: AsaOp; label: string }[] = [
  { value: "<=", label: "is at most" },
  { value: ">=", label: "is at least" },
  { value: "=", label: "is exactly" },
];
const GRADES = ASA_GRADES.map((g, i) => ({ value: String(i + 1), label: g }));
const KINDS: { value: RuleAction["kind"]; label: string }[] = [
  { value: "autoTriage", label: "Auto-triage" },
  { value: "task", label: "Add a task" },
  { value: "flag", label: "Add a flag" },
  { value: "requireAnaesthetistAsa", label: "Require the anaesthetist's ASA" },
  { value: "suggestStatus", label: "Suggest a status" },
];

/** A new action of a kind, with its usual first values (auto-triage: Ready for admission, A-8). */
function newAction(kind: RuleAction["kind"]): RuleAction {
  switch (kind) {
    case "autoTriage":
      return { kind, target: "ready_for_admission" };
    case "task":
      return { kind, title: "", team: "nurse" };
    case "flag":
      return { kind, name: "" };
    case "requireAnaesthetistAsa":
      return { kind };
    case "suggestStatus":
      return { kind, status: "ready_for_poa" };
  }
}

const codeKey = (c: CodeRef) => `${c.set}|${c.code}`;

export function RuleForm(props: {
  initial: AsaRule;
  scores: { key: string; title: string; bands: Band[] }[];
  codes: CodeRef[];
  procedures: OfferedProcedure[];
  /** Saves the rule; returns Go's refusal, or "" once saved. */
  onSave: (rule: AsaRule) => Promise<string>;
  /** Another change is saving: Save rule waits for it (test report 3, F-1). */
  busy: boolean;
  onCancel: () => void;
}) {
  const [rule, setRuleState] = useState<AsaRule>(props.initial);
  const [error, setError] = useState("");
  // A refusal goes once the rule is changed (test report 1, E-5).
  const setRule = (r: AsaRule) => {
    setRuleState(r);
    setError("");
  };
  const [saving, setSaving] = useState(false);
  const w = rule.when;
  const setWhen = (patch: Partial<AsaRule["when"]>) => setRule({ ...rule, when: { ...rule.when, ...patch } });
  const setAction = (i: number, a: RuleAction) => setRule({ ...rule, then: rule.then.map((x, j) => (j === i ? a : x)) });
  const score = w.band ? props.scores.find((s) => s.key === w.band!.score) : undefined;
  // "Isn't in a band" (S44c) may name a score this HQ hasn't (the defaults' STOP-Bang): always offered.
  const notScores = notBandChoices(props.scores, w.notBand?.score);
  const notScore = w.notBand ? notScores.find((s) => s.key === w.notBand!.score) : undefined;

  const save = async () => {
    setSaving(true);
    const problem = await props.onSave(rule);
    setSaving(false);
    setError(problem);
  };

  return (
    <Card withBorder data-testid="rule-form">
      <Stack gap="sm">
        <TextInput label="Name" placeholder="e.g. Low risk" value={rule.name} onChange={(e) => setRule({ ...rule, name: e.currentTarget.value })} />
        <Text size="sm" fw={600}>When</Text>
        <Group gap="xs" align="end">
          <Text size="sm" pb={6}>the suggested ASA</Text>
          <Select aria-label="Compared" w={140} data={OPS} value={w.asa.op} allowDeselect={false}
            onChange={(v) => setWhen({ asa: { ...w.asa, op: (v ?? "<=") as AsaOp } })} />
          <Select aria-label="ASA grade" w={90} data={GRADES} value={String(w.asa.grade)} allowDeselect={false}
            onChange={(v) => setWhen({ asa: { ...w.asa, grade: Number(v ?? 1) } })} />
        </Group>
        <Checkbox label="and there are no red flags" checked={!!w.noRedFlags} onChange={(e) => setWhen({ noRedFlags: e.currentTarget.checked || undefined })} />
        <Checkbox
          label="and a score is in a band"
          checked={!!w.band}
          disabled={!props.scores.length && !w.band}
          description={!props.scores.length && !w.band ? "This HQ has no score with bands." : undefined}
          onChange={(e) => setWhen({ band: e.currentTarget.checked ? { score: props.scores[0]?.key ?? "", bands: [] } : undefined })}
        />
        {w.band && (
          <Group gap="xs" pl="xl" align="end">
            <Select aria-label="Score" w={220} data={props.scores.map((s) => ({ value: s.key, label: s.title }))} value={w.band.score || null}
              onChange={(v) => setWhen({ band: { score: v ?? "", bands: [] } })} allowDeselect={false} />
            <MultiSelect aria-label="Bands" w={260} placeholder="is one of…" data={(score?.bands ?? []).map((b) => ({ value: b.id, label: b.label }))}
              value={w.band.bands} onChange={(v) => setWhen({ band: { ...w.band!, bands: v } })} />
          </Group>
        )}
        <Checkbox
          label="and a score isn't in a band"
          checked={!!w.notBand}
          onChange={(e) => setWhen({ notBand: e.currentTarget.checked ? { score: notScores[0]?.key ?? "", bands: [] } : undefined })}
        />
        {w.notBand && (
          <Group gap="xs" pl="xl" align="end">
            <Select aria-label="Score not in a band" w={300} data={notScores.map((s) => ({ value: s.key, label: s.title }))} value={w.notBand.score || null}
              onChange={(v) => setWhen({ notBand: { score: v ?? "", bands: [] } })} allowDeselect={false} />
            <MultiSelect aria-label="Bands it isn't in" w={260} placeholder="isn't any of…" data={(notScore?.bands ?? []).map((b) => ({ value: b.id, label: b.label }))}
              value={w.notBand.bands} onChange={(v) => setWhen({ notBand: { ...w.notBand!, bands: v } })} />
          </Group>
        )}
        <Checkbox
          label="and a disclosure has the code"
          checked={!!w.code}
          disabled={!props.codes.length && !w.code}
          description={!props.codes.length && !w.code ? "No disclosure in this HQ has a code." : undefined}
          onChange={(e) => setWhen({ code: e.currentTarget.checked ? props.codes[0] : undefined })}
        />
        {w.code && (
          <Select pl="xl" aria-label="Code" w={420} searchable allowDeselect={false}
            data={[...props.codes, ...(props.codes.some((c) => codeKey(c) === codeKey(w.code!)) ? [] : [w.code])].map((c) => ({ value: codeKey(c), label: `${c.display} (${c.code})` }))}
            value={codeKey(w.code)}
            onChange={(v) => setWhen({ code: props.codes.find((c) => codeKey(c) === v) ?? w.code })} />
        )}
        <Checkbox
          label="and the procedure is one of"
          checked={!!w.procedures}
          disabled={!props.procedures.length && !w.procedures}
          description={!props.procedures.length && !w.procedures ? "This hospital has no procedures." : undefined}
          onChange={(e) => setWhen({ procedures: e.currentTarget.checked ? [] : undefined })}
        />
        {w.procedures && (
          <MultiSelect pl="xl" aria-label="Procedures" w={420} searchable placeholder="Choose procedures…"
            data={[...props.procedures.map((p) => ({ value: p.id, label: p.name })), ...w.procedures.filter((id) => !props.procedures.some((p) => p.id === id)).map((id) => ({ value: id, label: id }))]}
            value={w.procedures} onChange={(v) => setWhen({ procedures: v })} />
        )}

        <Text size="sm" fw={600}>Then</Text>
        {rule.then.length === 0 && <Text size="sm" c="dimmed">No actions yet.</Text>}
        {rule.then.map((a, i) => (
          <Group key={i} gap="xs" align="end" wrap="nowrap" data-testid="rule-action">
            <Select aria-label="Action" w={240} data={KINDS} value={a.kind} allowDeselect={false}
              onChange={(v) => setAction(i, newAction((v ?? a.kind) as RuleAction["kind"]))} />
            {a.kind === "autoTriage" && (
              <Select aria-label="Target" w={220} data={TRIAGE_TARGETS} value={a.target} allowDeselect={false}
                onChange={(v) => setAction(i, { ...a, target: (v ?? a.target) as typeof a.target })} />
            )}
            {a.kind === "task" && (
              <>
                <TextInput aria-label="Task" placeholder="The task, e.g. Book anaesthetic review" w={280} value={a.title}
                  onChange={(e) => setAction(i, { ...a, title: e.currentTarget.value })} />
                <Select aria-label="Team" w={160} data={RULE_TEAMS} value={a.team} allowDeselect={false}
                  onChange={(v) => setAction(i, { ...a, team: (v ?? a.team) as typeof a.team })} />
              </>
            )}
            {a.kind === "flag" && (
              <TextInput aria-label="Flag" placeholder="The flag, e.g. Anaesthetic review" w={280} value={a.name}
                onChange={(e) => setAction(i, { ...a, name: e.currentTarget.value })} />
            )}
            {a.kind === "suggestStatus" && (
              <Select aria-label="Status" w={240} data={SUGGESTABLE} value={a.status} allowDeselect={false}
                onChange={(v) => setAction(i, { ...a, status: (v ?? a.status) as typeof a.status })} />
            )}
            <ActionIcon variant="subtle" color="red" aria-label="Remove action" mb={4}
              onClick={() => setRule({ ...rule, then: rule.then.filter((_, j) => j !== i) })}>
              <IconTrash size={14} />
            </ActionIcon>
          </Group>
        ))}
        <Group>
          <Button size="xs" variant="light" onClick={() => setRule({ ...rule, then: [...rule.then, newAction("task")] })}>+ Add action</Button>
        </Group>
        {error && <Alert color="red" p="xs"><Text size="sm">{error}</Text></Alert>}
        <Group justify="flex-end">
          <Button variant="default" onClick={props.onCancel}>Cancel</Button>
          <Button onClick={save} loading={saving} disabled={props.busy && !saving}>Save rule</Button>
        </Group>
      </Stack>
    </Card>
  );
}
