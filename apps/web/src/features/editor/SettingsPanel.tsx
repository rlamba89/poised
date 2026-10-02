"use client";
// The left column while a card is selected: Settings | Disclosures | Logic (Lifebox
// QuestionSettingsPanel). Text fields save when they lose focus, so typing is one undo step.
import { ActionIcon, Button, Group, Menu, Select, Stack, Switch, Tabs, Text } from "@mantine/core";
import { IconChevronDown, IconLink, IconPlus, IconX } from "@tabler/icons-react";
import {
  KIND_LABEL, bandsOf, convertKind, convertibleKinds, findElement, hasQuestionOutputs, isAnswerKind, isChoiceKind, kindOf,
  optionsOf, plainText, textOf, textToHtml, updateElement, type ChapterJson, type ElementJson,
} from "@poised/clinical";
import { KIND_ICON } from "./AddContentBar";
import { outputsAt, type DisclosureTarget } from "./DisclosureModal";
import { BlurText, BlurTextarea } from "./inputs";
import { ElementLogic } from "./LogicEditor";
import { TypeSettings } from "./TypeSettings";

export type PanelTab = "settings" | "disclosures" | "logic";

export function SettingsPanel(props: {
  doc: ChapterJson;
  name: string;
  tab: PanelTab;
  readOnly: boolean;
  onTab: (t: PanelTab) => void;
  onClose: () => void;
  change: (edit: (doc: ChapterJson) => ChapterJson) => void;
  openDisclosures: (t: DisclosureTarget) => void;
}) {
  const { doc, name, tab, readOnly, onTab, onClose, change } = props;
  const el = findElement(doc, name)?.el;
  if (!el) return null;
  const kind = kindOf(el);
  const Icon = KIND_ICON[kind];
  const canDisclose = isChoiceKind(kind) || hasQuestionOutputs(kind) || kind === "grid";
  const patch = (p: Partial<ElementJson>) => change((d) => updateElement(d, name, p));
  const targets = convertibleKinds(el);
  const isGroup = kind === "group" || kind === "section";

  return (
    <Tabs value={tab} onChange={(t) => t && onTab(t as PanelTab)} keepMounted={false}>
      <Group justify="space-between" wrap="nowrap" pr="sm" style={{ borderBottom: "1px solid var(--mantine-color-gray-3)" }}>
        <Tabs.List style={{ borderBottom: 0 }}>
          <Tabs.Tab value="settings">Settings</Tabs.Tab>
          {canDisclose && <Tabs.Tab value="disclosures">Disclosures</Tabs.Tab>}
          <Tabs.Tab value="logic">Logic</Tabs.Tab>
        </Tabs.List>
        <ActionIcon variant="subtle" color="gray" aria-label="Close settings" onClick={onClose}>
          <IconX size={18} />
        </ActionIcon>
      </Group>

      <div style={{ padding: 20 }}>
        <Group gap="xs" mb="md">
          <Icon size={18} />
          <Text fw={600}>{KIND_LABEL[kind]}{el.type === "paneldynamic" ? " (repeatable)" : ""}</Text>
          {!readOnly && targets.length > 0 && (
            <Menu position="bottom-start">
              <Menu.Target>
                <Button size="compact-xs" variant="subtle" rightSection={<IconChevronDown size={12} />}>Change type</Button>
              </Menu.Target>
              <Menu.Dropdown>
                {targets.map((k) => {
                  const I = KIND_ICON[k];
                  return (
                    <Menu.Item key={k} leftSection={<I size={14} />} onClick={() => change((d) => convertKind(d, name, k))}>
                      {KIND_LABEL[k]}
                    </Menu.Item>
                  );
                })}
              </Menu.Dropdown>
            </Menu>
          )}
          <Text size="xs" c="dimmed" ml="auto">ID {el.name}</Text>
        </Group>

        <Tabs.Panel value="settings">
          <Stack>
            {kind === "statement" ? (
              <>
                <BlurTextarea label="Text" value={plainText(textOf(el.html))} minRows={4} disabled={readOnly} onSave={(t) => patch({ html: textToHtml(t) })} />
                <ClinicalSwitch el={el} readOnly={readOnly} patch={patch} />
              </>
            ) : (
              <>
                <BlurTextarea
                  label={isGroup ? "Name" : "Question text"}
                  value={textOf(el.title)}
                  disabled={readOnly}
                  onSave={(title) => patch({ title })}
                />
                {kind !== "section" && (
                  <BlurTextarea label="Description" value={textOf(el.description)} disabled={readOnly} onSave={(v) => patch({ description: v || undefined })} />
                )}
                {!isGroup && textOf(el.description) && (
                  <Select
                    label="Show the description"
                    data={[
                      { value: "default", label: "Under the question" },
                      { value: "underInput", label: "Under the answer" },
                    ]}
                    value={el.descriptionLocation === "underInput" ? "underInput" : "default"}
                    allowDeselect={false}
                    disabled={readOnly}
                    onChange={(v) => patch({ descriptionLocation: v === "underInput" ? "underInput" : undefined })}
                  />
                )}
                {isAnswerKind(kind) && kind !== "calculation" && <RequiredFields el={el} readOnly={readOnly} patch={patch} />}
                {kind !== "section" && <ClinicalSwitch el={el} readOnly={readOnly} patch={patch} />}
                {kind === "section" && (
                  <Switch
                    label="Show as patient page heading"
                    description="Each Section is its own screen for patients. When switched off, the first question's text is the heading."
                    checked={el.showAsHeading !== false}
                    disabled={readOnly}
                    onChange={(e) => patch({ showAsHeading: e.currentTarget.checked ? undefined : false })}
                  />
                )}
                {kind === "medication" && (
                  <Select
                    label="Medication type"
                    data={[
                      { value: "prescribed", label: "Prescribed" },
                      { value: "non_prescribed", label: "Non-prescribed" },
                      { value: "recreational", label: "Recreational" },
                    ]}
                    value={String(el.medicationType ?? "prescribed")}
                    allowDeselect={false}
                    disabled={readOnly}
                    onChange={(v) => patch({ medicationType: v ?? "prescribed" })}
                  />
                )}
                <TypeSettings key={name} doc={doc} el={el} readOnly={readOnly} change={change} />
              </>
            )}
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="disclosures">
          <DisclosuresTab doc={doc} el={el} readOnly={readOnly} open={props.openDisclosures} />
        </Tabs.Panel>

        <Tabs.Panel value="logic">
          <ElementLogic key={name} doc={doc} name={name} readOnly={readOnly} change={change} />
        </Tabs.Panel>
      </div>
    </Tabs>
  );
}

function ClinicalSwitch({ el, readOnly, patch }: { el: ElementJson; readOnly: boolean; patch: (p: Partial<ElementJson>) => void }) {
  return (
    <Switch
      label="Clinical"
      description="Only clinicians see this. Patients never receive it."
      checked={!!el.clinicianOnly}
      disabled={readOnly}
      onChange={(e) => patch({ clinicianOnly: e.currentTarget.checked || undefined })}
    />
  );
}

/** Required, the message shown when it's missing (VAL-04), or the required-if note (LOG-10). */
function RequiredFields({ el, readOnly, patch }: { el: ElementJson; readOnly: boolean; patch: (p: Partial<ElementJson>) => void }) {
  if (el.requiredIf) {
    return (
      <Text size="sm" c="dimmed">
        <b>Required</b> only when a condition is met. Change it in the Logic tab.
      </Text>
    );
  }
  return (
    <>
      <Switch label="Required" checked={!!el.isRequired} disabled={readOnly} onChange={(e) => patch({ isRequired: e.currentTarget.checked || undefined })} />
      {el.isRequired && (
        <BlurText
          label="Message when it's missing"
          placeholder="This is a required field."
          value={textOf(el.requiredErrorText)}
          disabled={readOnly}
          onSave={(v) => patch({ requiredErrorText: v || undefined })}
        />
      )}
    </>
  );
}

function DisclosuresTab({ doc, el, readOnly, open }: { doc: ChapterJson; el: ElementJson; readOnly: boolean; open: (t: DisclosureTarget) => void }) {
  const kind = kindOf(el);
  const label = (list: unknown) =>
    (Array.isArray(list) ? list : []).map((c: { value: string; text: unknown } | string) =>
      typeof c === "string" ? { value: c, text: c } : { value: c.value, text: textOf(c.text) || c.value },
    );
  let rows: { key: string; label: string; heading?: string; target: DisclosureTarget }[];
  if (isChoiceKind(kind)) {
    rows = optionsOf(el).map((o) => ({
      key: o.value,
      label: o.text + (o.isExclusive && !o.special ? " (No to all)" : ""),
      target: { element: el.name, option: o.value },
    }));
  } else if (kind === "grid") {
    const columns = label(el.columns);
    rows = label(el.rows).flatMap((r) =>
      columns.map((c, i) => ({ key: `${r.value}.${c.value}`, heading: i === 0 ? r.text : undefined, label: c.text, target: { element: el.name, cell: { row: r.value, column: c.value } } })),
    );
  } else if (kind === "calculation" && bandsOf(el).length) {
    rows = bandsOf(el).map((b) => ({ key: b.id, label: `${b.label} band`, target: { element: el.name, band: b.id } }));
  } else {
    rows = [{ key: "q", label: textOf(el.title) || el.name, target: { element: el.name } }];
  }
  return (
    <Stack gap={0}>
      {rows.map((r) => {
        const n = outputsAt(doc, r.target).length;
        return (
          <div key={r.key}>
            {r.heading && <Text size="xs" fw={700} tt="uppercase" c="dimmed" mt="sm">{r.heading}</Text>}
            <Group justify="space-between" wrap="nowrap" py={8} pl={r.target.cell ? "sm" : 0} style={{ borderBottom: "1px solid var(--mantine-color-gray-2)" }}>
              <Text size="sm">{r.label}</Text>
              {n > 0 ? (
                <Button size="xs" variant="outline" color="green" leftSection={<IconLink size={12} />} onClick={() => open(r.target)}>
                  {n}
                </Button>
              ) : (
                !readOnly && (
                  <Button size="xs" variant="default" leftSection={<IconPlus size={12} />} onClick={() => open(r.target)}>
                    Add
                  </Button>
                )
              )}
            </Group>
          </div>
        );
      })}
    </Stack>
  );
}
