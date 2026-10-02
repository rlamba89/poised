"use client";
// Pick a question, a comparison and a value from lists (LOG-05), as many rules as needed,
// joined by All / Any / None / Not all, with groups nested to any depth (LOG-04).
// Writes a SurveyJS expression.
import { useEffect, useMemo, useState } from "react";
import { ActionIcon, Alert, Button, Group, MultiSelect, NumberInput, Paper, SegmentedControl, Select, Stack, Text, TextInput } from "@mantine/core";
import { IconPlus, IconTrash } from "@tabler/icons-react";
import {
  OPS, builderCanShow, formatLogic, isAllComplete, isGroup, opChoiceOf, parseLogic, subjectsFor, type ChapterJson, type Group as LogicGroup,
  type OpChoice, type Rule, type Subject,
} from "@poised/clinical";

export type Scope = { element: string } | { page: string } | { upToPage: string };

export function ConditionBuilder(props: {
  doc: ChapterJson;
  scope: Scope;
  value: string | undefined;
  readOnly: boolean;
  onChange: (expr: string | undefined) => void;
  /** Shown when there is nothing to test yet. */
  emptyHint?: string;
}) {
  const { doc, scope, value, readOnly, onChange } = props;
  const subjects = useMemo(() => subjectsFor(doc, scope), [doc, scope]);
  // A local draft keeps rows the author hasn't finished; only complete rules are saved.
  const [draft, setDraft] = useState<LogicGroup | undefined>(() => start(value));
  useEffect(() => {
    if (formatLogic(draft ?? { join: "and", items: [] }) !== (value || undefined)) setDraft(start(value));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value]);

  if (!builderCanShow(draft)) {
    return <Alert color="yellow">This condition uses arithmetic or functions the Builder doesn&apos;t show. Use the Code tab.</Alert>;
  }
  const update = (next: LogicGroup) => {
    setDraft(next);
    // Half-finished rules are never saved: the saved condition stays until every rule is complete.
    if (!isAllComplete(next)) return;
    const expr = formatLogic(next);
    if (expr !== (value || undefined)) onChange(expr);
  };
  const ownSubjects = subjects.filter((s) => s.group !== "Patient and viewer");

  return (
    <Stack gap="xs">
      {ownSubjects.length === 0 && props.emptyHint && <Text size="xs" c="dimmed">{props.emptyHint}</Text>}
      <GroupEditor group={draft} subjects={subjects} readOnly={readOnly} depth={0} onChange={(g) => g && update(g)} />
    </Stack>
  );
}

/** How a group's items combine: All, Any, None of them, or Not all of them. */
const JOINS = [
  { value: "and", label: "All" },
  { value: "or", label: "Any" },
  { value: "none", label: "None" },
  { value: "notall", label: "Not all" },
];
const joinOf = (g: LogicGroup) => (g.not ? (g.join === "or" ? "none" : "notall") : g.join);
const withJoin = (g: LogicGroup, v: string): LogicGroup => ({
  ...g,
  join: v === "or" || v === "none" ? "or" : "and",
  not: v === "none" || v === "notall" ? true : undefined,
});
const joinWord = (g: LogicGroup) => (g.join === "or" ? "or" : "and");

/** A group of rules and groups, nested to any depth (LOG-04). The top level can't be removed. */
function GroupEditor(props: { group: LogicGroup; subjects: Subject[]; readOnly: boolean; depth: number; onChange: (g: LogicGroup | null) => void }) {
  const { group: g, subjects, readOnly, depth, onChange } = props;
  const nested = depth > 0;
  const set = (i: number, next: Rule | LogicGroup | null) => {
    const items = next ? g.items.map((x, j) => (j === i ? next : x)) : g.items.filter((_, j) => j !== i);
    onChange(items.length || !nested ? { ...g, items } : null);
  };
  const showJoin = g.items.length > 1 || g.not || nested;
  const body = (
    <Stack gap="xs">
      {showJoin && (
        <Group justify="space-between" wrap="nowrap">
          <Group gap="xs" wrap="nowrap">
            <Text size={nested ? "xs" : "sm"}>Match</Text>
            <SegmentedControl size="xs" disabled={readOnly} value={joinOf(g)} onChange={(v) => onChange(withJoin(g, v))} data={JOINS} />
            <Text size={nested ? "xs" : "sm"} c={nested ? "dimmed" : undefined}>{nested ? "of this group" : "of these"}</Text>
          </Group>
          {nested && !readOnly && (
            <ActionIcon variant="subtle" color="red" aria-label="Remove group" onClick={() => onChange(null)}>
              <IconTrash size={14} />
            </ActionIcon>
          )}
        </Group>
      )}
      {g.items.map((item, i) => (
        <div key={i}>
          {i > 0 && <Text size="xs" fw={700} c="dimmed" tt="uppercase" mb={4}>{joinWord(g)}</Text>}
          {isGroup(item) ? (
            <GroupEditor group={item} subjects={subjects} readOnly={readOnly} depth={depth + 1} onChange={(next) => set(i, next)} />
          ) : (
            <RuleRow subjects={subjects} rule={item} readOnly={readOnly} onChange={(next) => set(i, next)} />
          )}
        </div>
      ))}
      {!readOnly && (
        <Group gap="xs">
          <Button
            size={nested ? "compact-xs" : "xs"}
            variant={nested ? "subtle" : "light"}
            leftSection={<IconPlus size={nested ? 12 : 14} />}
            onClick={() => onChange({ ...g, items: [...g.items, emptyRule()] })}
          >
            {nested ? "Add condition to group" : "Add condition"}
          </Button>
          {g.items.length > 0 && (
            <Button
              size={nested ? "compact-xs" : "xs"}
              variant="subtle"
              leftSection={<IconPlus size={nested ? 12 : 14} />}
              onClick={() => onChange({ ...g, items: [...g.items, { join: g.join === "and" ? "or" : "and", items: [emptyRule(), emptyRule()] }] })}
            >
              {nested ? "Add group inside" : "Add group"}
            </Button>
          )}
        </Group>
      )}
    </Stack>
  );
  return nested ? (
    <Paper withBorder p="xs" bg={depth % 2 ? "gray.0" : "white"}>
      {body}
    </Paper>
  ) : (
    body
  );
}

const emptyRule = (): Rule => ({ left: "", op: "eq" });

/** A new draft: the parsed expression, or one empty rule to fill in. */
function start(value: string | undefined): LogicGroup | undefined {
  const g = parseLogic(value);
  if (g && g.items.length === 0) return { join: "and", items: [emptyRule()] };
  return g;
}

/** One rule: question → comparison → value. */
function RuleRow({ subjects, rule, readOnly, onChange }: { subjects: Subject[]; rule: Rule; readOnly: boolean; onChange: (r: Rule | null) => void }) {
  const subject = subjects.find((s) => (rule.fn === "band" ? s.id === `band:${rule.left}` : s.left === rule.left && !s.fn));
  const groups = [...new Set(subjects.map((s) => s.group))].map((group) => ({
    group,
    items: subjects.filter((s) => s.group === group).map((s) => ({ value: s.id, label: s.label })),
  }));
  const ops = subject ? OPS[subject.type] : [];
  const choice = subject ? opChoiceOf(subject, rule) : undefined;

  const pickSubject = (id: string | null) => {
    const s = subjects.find((x) => x.id === id);
    if (!s) return onChange({ left: "", op: "eq" });
    const first = OPS[s.type][0];
    onChange({ left: s.left, fn: s.fn ?? first.fn, op: first.op });
  };
  const pickOp = (key: string | null) => {
    const o = ops.find((x) => x.key === key);
    if (!o || !subject) return;
    const keepValue = choice && sameInput(choice, o) ? rule.value : undefined;
    onChange({ left: rule.left, fn: subject.fn ?? o.fn, op: o.op, value: keepValue });
  };

  return (
    <Stack gap={6}>
      <Group gap={6} wrap="nowrap" align="flex-start">
        <Select
          style={{ flex: 1 }}
          size="xs"
          aria-label="Question"
          placeholder="Select a question…"
          data={groups}
          value={subject?.id ?? null}
          searchable
          disabled={readOnly}
          onChange={pickSubject}
          comboboxProps={{ withinPortal: true }}
          nothingFoundMessage="No earlier question matches"
        />
        {!readOnly && (
          <ActionIcon variant="subtle" color="red" aria-label="Remove condition" onClick={() => onChange(null)} mt={2}>
            <IconTrash size={14} />
          </ActionIcon>
        )}
      </Group>
      {subject && (
        <Group gap={6} wrap="nowrap" align="flex-start" pr={34}>
          <Select
            size="xs"
            w={170}
            aria-label="Comparison"
            data={ops.map((o) => ({ value: o.key, label: o.label }))}
            value={choice?.key ?? null}
            allowDeselect={false}
            disabled={readOnly}
            onChange={pickOp}
          />
          {choice && <ValueInput subject={subject} choice={choice} rule={rule} readOnly={readOnly} onChange={(value) => onChange({ ...rule, value })} />}
        </Group>
      )}
    </Stack>
  );
}

const sameInput = (a: OpChoice, b: OpChoice) => a.input === b.input && a.fn === b.fn;

function ValueInput({ subject, choice, rule, readOnly, onChange }: { subject: Subject; choice: OpChoice; rule: Rule; readOnly: boolean; onChange: (v: Rule["value"]) => void }) {
  const options = subject.options ?? [];
  const common = { size: "xs" as const, style: { flex: 1 }, disabled: readOnly, "aria-label": "Value" };
  switch (choice.input) {
    case "option":
      return <Select {...common} placeholder="Select an option" data={options} value={typeof rule.value === "string" ? rule.value : null} onChange={(v) => onChange(v ?? undefined)} />;
    case "options":
      return (
        <MultiSelect
          {...common}
          placeholder="Select options"
          searchable
          data={options}
          value={Array.isArray(rule.value) ? rule.value : []}
          onChange={(v) => onChange(v)}
        />
      );
    case "number":
      return (
        <NumberInput
          {...common}
          placeholder={choice.fn === "age" ? "Years" : "Number"}
          value={typeof rule.value === "number" ? rule.value : ""}
          onChange={(v) => onChange(v === "" ? undefined : Number(v))}
        />
      );
    case "date":
      return <TextInput {...common} type="date" value={typeof rule.value === "string" ? rule.value : ""} onChange={(e) => onChange(e.currentTarget.value || undefined)} />;
    case "text":
      return <BlurText {...common} value={typeof rule.value === "string" ? rule.value : ""} onSave={(v) => onChange(v || undefined)} />;
    default:
      return null;
  }
}

/** Text saved when it loses focus, so typing doesn't save every letter. */
function BlurText({ value, onSave, ...rest }: { value: string; onSave: (v: string) => void } & Omit<React.ComponentProps<typeof TextInput>, "value" | "onChange">) {
  const [draft, setDraft] = useState(value);
  useEffect(() => setDraft(value), [value]);
  return <TextInput {...rest} value={draft} onChange={(e) => setDraft(e.currentTarget.value)} onBlur={() => draft !== value && onSave(draft)} placeholder="Text" />;
}
