"use client";
// The Logic tab (LOG-01/02/04/05/07/10/11/12/13, OPT-05). The Builder covers when an element is
// shown, required or read-only, and when each option is shown; for a page, when it is shown and
// its skip rules. The Code tab edits the raw SurveyJS expressions for anything more.
import { useEffect, useState } from "react";
import { Alert, Badge, Button, Divider, Group, Modal, Radio, SegmentedControl, Select, Stack, Tabs, Text, Textarea } from "@mantine/core";
import { IconGitBranch, IconPlus, IconTrash } from "@tabler/icons-react";
import {
  dependentsOf, describeLogic, enableIfToReadOnly, findElement, findPage, firstQuestionOn, isAnswerKind, isChoiceKind,
  isValidExpression, kindOf, optionsOf, pageOfTrigger, pageTitle, pagesOf, readOnlyToEnableIf, setTriggers,
  triggersOf, triggerTarget, unknownReferences, updateElement, updatePage, countDisclosures, type ChapterJson, type ElementJson,
  type TriggerJson,
} from "@poised/clinical";
import { ConditionBuilder, type Scope } from "./ConditionBuilder";
import { DisplaysWhen } from "./ElementCard";

type Change = (edit: (doc: ChapterJson) => ChapterJson) => void;

/** Logic for one element. */
export function ElementLogic({ doc, name, readOnly, change }: { doc: ChapterJson; name: string; readOnly: boolean; change: Change }) {
  const el = findElement(doc, name)?.el;
  const [tab, setTab] = useState<string | null>("builder");
  if (!el) return null;
  const kind = kindOf(el);
  const scope: Scope = { element: name };
  const patch = (p: Partial<ElementJson>) => change((d) => updateElement(d, name, p));
  const answer = isAnswerKind(kind) && kind !== "calculation";
  const dependents = dependentsOf(doc, name);

  return (
    <Tabs value={tab} onChange={setTab} keepMounted={false}>
      <Tabs.List mb="md">
        <Tabs.Tab value="builder">Builder</Tabs.Tab>
        <Tabs.Tab value="code">Code</Tabs.Tab>
      </Tabs.List>
      <Tabs.Panel value="builder">
        <Stack gap="lg">
          <Toggle
            label="Display"
            off="Always"
            on="Conditionally"
            doc={doc}
            scope={scope}
            value={el.visibleIf}
            readOnly={readOnly}
            onChange={(visibleIf) => patch({ visibleIf })}
          />
          {answer && (
            <Toggle
              label="Required"
              off={el.isRequired ? "Always (set in Settings)" : "Never (set in Settings)"}
              on="Only when"
              doc={doc}
              scope={scope}
              value={el.requiredIf as string | undefined}
              readOnly={readOnly}
              // A required-if condition decides on its own, so the Settings switch is turned off.
              onChange={(requiredIf) => patch({ requiredIf, isRequired: requiredIf ? undefined : el.isRequired })}
            />
          )}
          {answer && (
            <Toggle
              label="Read-only"
              off="Never"
              on="When"
              doc={doc}
              scope={scope}
              value={enableIfToReadOnly(el.enableIf as string | undefined)}
              readOnly={readOnly}
              onChange={(expr) => patch({ enableIf: readOnlyToEnableIf(expr) })}
            />
          )}
          {isChoiceKind(kind) && <OptionLogic doc={doc} el={el} readOnly={readOnly} change={change} />}
          <Divider />
          <div>
            <Text size="sm" fw={600} mb={4}>Used by</Text>
            {dependents.length === 0 && countDisclosures(el) === 0 ? (
              <Text size="sm" c="dimmed">Nothing depends on this yet.</Text>
            ) : (
              <Stack gap={2}>
                {dependents.map((d) => (
                  <Text size="sm" key={d}>· {d}</Text>
                ))}
                {countDisclosures(el) > 0 && <Text size="sm">· {countDisclosures(el)} disclosure{countDisclosures(el) > 1 ? "s" : ""}</Text>}
              </Stack>
            )}
          </div>
        </Stack>
      </Tabs.Panel>
      <Tabs.Panel value="code">
        <Code
          doc={doc}
          fields={[
            { key: "visibleIf", label: "Visible if", value: el.visibleIf },
            ...(answer
              ? [
                  { key: "requiredIf", label: "Required if", value: el.requiredIf as string | undefined },
                  { key: "enableIf", label: "Enable if", value: el.enableIf as string | undefined },
                ]
              : []),
          ]}
          readOnly={readOnly}
          onSave={(key, value) => patch({ [key]: value, ...(key === "requiredIf" && value ? { isRequired: undefined } : {}) })}
        />
      </Tabs.Panel>
    </Tabs>
  );
}

/** Always / Conditionally, and the builder when conditional. */
export function Toggle(props: {
  label: string;
  off: string;
  on: string;
  doc: ChapterJson;
  scope: Scope;
  value: string | undefined;
  readOnly: boolean;
  onChange: (e: string | undefined) => void;
  emptyHint?: string;
}) {
  const [conditional, setConditional] = useState(!!props.value);
  useEffect(() => setConditional(!!props.value), [props.value]);
  return (
    <div>
      <Radio.Group
        label={props.label}
        value={conditional ? "on" : "off"}
        onChange={(v) => {
          setConditional(v === "on");
          if (v === "off") props.onChange(undefined);
        }}
      >
        <Group mt={6} mb={conditional ? "sm" : 0}>
          <Radio value="off" label={props.off} disabled={props.readOnly} />
          <Radio value="on" label={props.on} disabled={props.readOnly} />
        </Group>
      </Radio.Group>
      {conditional && (
        <ConditionBuilder
          doc={props.doc}
          scope={props.scope}
          value={props.value}
          readOnly={props.readOnly}
          onChange={props.onChange}
          emptyHint={props.emptyHint ?? "Only questions that come earlier can be tested, plus the patient's age and sex and who is viewing."}
        />
      )}
    </div>
  );
}

/** When each option is shown (OPT-05). */
function OptionLogic({ doc, el, readOnly, change }: { doc: ChapterJson; el: ElementJson; readOnly: boolean; change: Change }) {
  const [editing, setEditing] = useState<string | null>(null);
  const options = optionsOf(el);
  const option = options.find((o) => o.value === editing);
  const setOptionIf = (value: string, visibleIf: string | undefined) =>
    change((d) => updateElement(d, el.name, { choices: options.map((o) => (o.value === value ? { ...o, visibleIf } : o)).map(cleanChoice) }));

  return (
    <div>
      <Text size="sm" fw={500} mb={6}>Options</Text>
      <Stack gap={4}>
        {options.map((o) => {
          const d = describeLogic(doc, o.visibleIf);
          return (
            <Group key={o.value} justify="space-between" wrap="nowrap" gap="xs" py={4} style={{ borderBottom: "1px solid var(--mantine-color-gray-2)" }}>
              <div style={{ minWidth: 0 }}>
                <Text size="sm">{o.text}</Text>
                {d ? (
                  <Text size="xs" c="orange.8" component="div" style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
                    <IconGitBranch size={12} /> <DisplaysWhen condition={d} compact />
                  </Text>
                ) : (
                  <Text size="xs" c="dimmed">Always shown</Text>
                )}
              </div>
              {!readOnly && (
                <Button size="compact-xs" variant="subtle" onClick={() => setEditing(o.value)}>{o.visibleIf ? "Edit" : "Add condition"}</Button>
              )}
            </Group>
          );
        })}
      </Stack>
      <Modal opened={!!option} onClose={() => setEditing(null)} title={`When is “${option?.text}” shown?`} size="lg">
        {option && (
          <Stack>
            <ConditionBuilder doc={doc} scope={{ element: el.name }} value={option.visibleIf} readOnly={readOnly} onChange={(v) => setOptionIf(option.value, v)} />
            <Group justify="space-between">
              <Button variant="subtle" color="red" disabled={!option.visibleIf} onClick={() => setOptionIf(option.value, undefined)}>Always show</Button>
              <Button onClick={() => setEditing(null)}>Done</Button>
            </Group>
          </Stack>
        )}
      </Modal>
    </div>
  );
}

const cleanChoice = <T extends { visibleIf?: string }>(c: T): T => {
  if (c.visibleIf) return c;
  const { visibleIf, ...rest } = c;
  void visibleIf;
  return rest as T;
};

/** Raw expressions, checked as they are typed (LOG-07). */
function Code(props: {
  doc: ChapterJson;
  fields: { key: string; label: string; value: string | undefined }[];
  readOnly: boolean;
  onSave: (key: string, value: string | undefined) => void;
}) {
  const [key, setKey] = useState(props.fields[0].key);
  const field = props.fields.find((f) => f.key === key) ?? props.fields[0];
  return (
    <Stack>
      {props.fields.length > 1 && (
        <SegmentedControl size="xs" value={field.key} onChange={setKey} data={props.fields.map((f) => ({ value: f.key, label: f.label }))} />
      )}
      <CodeField key={field.key} doc={props.doc} label={field.label} value={field.value} readOnly={props.readOnly} onSave={(v) => props.onSave(field.key, v)} />
    </Stack>
  );
}

function CodeField({ doc, label, value, readOnly, onSave }: { doc: ChapterJson; label: string; value: string | undefined; readOnly: boolean; onSave: (v: string | undefined) => void }) {
  const [draft, setDraft] = useState(value ?? "");
  useEffect(() => setDraft(value ?? ""), [value]);
  const trimmed = draft.trim();
  const invalid = trimmed !== "" && !isValidExpression(trimmed);
  const unknown = trimmed && !invalid ? unknownReferences(doc, trimmed) : [];
  return (
    <Stack>
      <Textarea
        label={label}
        description="A SurveyJS expression. Questions are referred to by ID, for example {q_abcdef} = 'o_abcd'. Also: {patientAge}, {patientSex}, {viewer}, age(), score(), band()."
        value={draft}
        onChange={(e) => setDraft(e.currentTarget.value)}
        autosize
        minRows={3}
        styles={{ input: { fontFamily: "monospace" } }}
        disabled={readOnly}
        error={invalid ? "This isn't a valid expression." : unknown.length ? `Unknown question: ${unknown.join(", ")}` : undefined}
      />
      {!readOnly && (
        <Group justify="flex-end">
          <Button variant="default" onClick={() => setDraft(value ?? "")} disabled={draft === (value ?? "")}>Reset</Button>
          <Button onClick={() => onSave(trimmed || undefined)} disabled={invalid || draft === (value ?? "")}>Save logic</Button>
        </Group>
      )}
    </Stack>
  );
}

// ---------------------------------------------------------------- pages

/** Logic for a page: when it is shown (LOG-02), and its skip rules (LOG-12). */
export function PageLogic({ doc, page: pageName, readOnly, change }: { doc: ChapterJson; page: string; readOnly: boolean; change: Change }) {
  const page = findPage(doc, pageName);
  const [tab, setTab] = useState<string | null>("builder");
  if (!page) return null;
  return (
    <Tabs value={tab} onChange={setTab} keepMounted={false}>
      <Tabs.List mb="md">
        <Tabs.Tab value="builder">Builder</Tabs.Tab>
        <Tabs.Tab value="code">Code</Tabs.Tab>
      </Tabs.List>
      <Tabs.Panel value="builder">
        <Stack gap="lg">
          <Toggle
            label="Display"
            off="Always"
            on="Conditionally"
            doc={doc}
            scope={{ page: pageName }}
            value={page.visibleIf}
            readOnly={readOnly}
            onChange={(visibleIf) => change((d) => updatePage(d, pageName, { visibleIf }))}
          />
          <Divider />
          <SkipRules doc={doc} page={pageName} readOnly={readOnly} change={change} />
        </Stack>
      </Tabs.Panel>
      <Tabs.Panel value="code">
        <Code
          doc={doc}
          fields={[{ key: "visibleIf", label: "Visible if", value: page.visibleIf }]}
          readOnly={readOnly}
          onSave={(_, visibleIf) => change((d) => updatePage(d, pageName, { visibleIf }))}
        />
      </Tabs.Panel>
    </Tabs>
  );
}

const END = "__end";

/** "When … skip to page X / end this Question Set", after this page's answers (LOG-12). */
function SkipRules({ doc, page: pageName, readOnly, change }: { doc: ChapterJson; page: string; readOnly: boolean; change: Change }) {
  const pages = pagesOf(doc);
  const index = pages.findIndex((p) => p.name === pageName);
  const all = triggersOf(doc);
  // A new rule has no condition yet, so it is kept here until it has one.
  const [pending, setPending] = useState<TriggerJson | null>(null);
  // Rules stay on the page they were added on (older ones: the latest page they test).
  const mine = all.filter((t) => (pageOfTrigger(doc, t) ?? pages[0])?.name === pageName);
  const later = pages.slice(index + 1).filter((p) => firstQuestionOn(p));
  const save = (before: TriggerJson | null, after: TriggerJson | null) => {
    const rest = all.filter((t) => t !== before);
    change((d) => setTriggers(d, after ? (before ? all.map((t) => (t === before ? after : t)) : [...rest, after]) : rest));
  };
  const targetValue = (t: TriggerJson) => (t.type === "complete" ? END : triggerTarget(doc, t)?.name ?? null);
  const withTarget = (t: TriggerJson, v: string | null): TriggerJson =>
    v === END
      ? { type: "complete", expression: t.expression, page: pageName }
      : { type: "skip", expression: t.expression, gotoName: firstQuestionOn(pages.find((p) => p.name === v)!)?.name, page: pageName };

  const row = (t: TriggerJson, i: number, isPending: boolean) => (
    <div key={isPending ? "pending" : i} style={{ border: "1px solid var(--mantine-color-gray-3)", borderRadius: 4, padding: 10 }}>
      <Group justify="space-between" mb={6}>
        <Badge variant="light" color="orange" size="sm">Skip rule</Badge>
        {!readOnly && (
          <Button size="compact-xs" variant="subtle" color="red" leftSection={<IconTrash size={12} />} onClick={() => (isPending ? setPending(null) : save(t, null))}>
            Remove
          </Button>
        )}
      </Group>
      <Text size="sm" fw={500} mb={4}>When</Text>
      <ConditionBuilder
        doc={doc}
        scope={{ upToPage: pageName }}
        value={t.expression || undefined}
        readOnly={readOnly}
        onChange={(expression) => {
          if (isPending) {
            if (expression) {
              setPending(null);
              save(null, { ...t, expression, page: pageName });
            } else setPending({ ...t, expression: "" });
          } else if (expression) save(t, { ...t, expression, page: pageName });
        }}
      />
      <Select
        mt="sm"
        size="xs"
        label="Then"
        description={t.type === "complete" ? "Ends when the respondent next clicks Next or Complete with the condition met." : "Jumps as soon as the answers meet the condition."}
        data={[
          ...later.map((p) => ({ value: p.name, label: `Skip to “${pageTitle(p, pages.indexOf(p))}”` })),
          { value: END, label: "End this Question Set" },
        ]}
        value={targetValue(t)}
        allowDeselect={false}
        disabled={readOnly}
        onChange={(v) => (isPending ? setPending(withTarget(t, v)) : save(t, withTarget(t, v)))}
      />
    </div>
  );

  return (
    <div>
      <Text size="sm" fw={600}>Skip rules</Text>
      <Text size="xs" c="dimmed" mb="xs">Skip ahead, or end the Question Set early, depending on the answers so far.</Text>
      <Stack gap="sm">
        {mine.map((t, i) => row(t, i, false))}
        {pending && row(pending, -1, true)}
        {mine.length === 0 && !pending && <Text size="sm" c="dimmed">No skip rules.</Text>}
        {!readOnly && !pending && (
          <Button size="xs" variant="light" w="fit-content" leftSection={<IconPlus size={14} />} onClick={() => setPending({ type: "complete", expression: "" })}>
            Add skip rule
          </Button>
        )}
      </Stack>
      {all.length > mine.length && (
        <Alert color="gray" mt="sm" p="xs">
          <Text size="xs">Other pages have {all.length - mine.length} more skip rule{all.length - mine.length > 1 ? "s" : ""}.</Text>
        </Alert>
      )}
    </div>
  );
}
