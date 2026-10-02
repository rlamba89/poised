"use client";
// Options of Yes / No, Select One and Select Many: labels, order, scores (OPT-03), the
// exclusive "None" option, one-click options (OPT-04/06), Select all, and options imported
// from the code library with their code as a disclosure (QT-07), and saved option lists
// (OPT-07). Also grid rows and columns.
import { useState } from "react";
import { ActionIcon, Button, Group, Modal, Stack, Switch, Text, Tooltip } from "@mantine/core";
import { IconArrowDown, IconArrowUp, IconBooks, IconDeviceFloppy, IconList, IconPlus, IconTrash } from "@tabler/icons-react";
import {
  addChoice, kindOf, newId, optionsOf, orderOptions, setNoneOption, setSpecialOption, textOf, updateElement, SPECIAL_OPTIONS,
  type ChapterJson, type ChoiceJson, type CodeRef, type ElementJson, type SpecialOption,
} from "@poised/clinical";
import { CodePicker } from "@/features/outputs/CodePicker";
import { BlurNumber, BlurText } from "./inputs";
import { SaveOptionList, UseOptionList } from "./OptionLists";

type Change = (e: (d: ChapterJson) => ChapterJson) => void;

export function OptionsEditor({ el, readOnly, change }: { el: ElementJson; readOnly: boolean; change: Change }) {
  const kind = kindOf(el);
  const all = optionsOf(el);
  const normal = all.filter((o) => !o.isExclusive && !o.special);
  const none = all.find((o) => o.isExclusive && !o.special);
  const min = kind === "selectmany" ? 1 : 2;
  const fixed = kind === "yesno"; // QT-01: two options, labels editable
  const [scores, setScores] = useState(() => all.some((o) => o.score !== undefined));
  const [importing, setImporting] = useState(false);
  const [listAction, setListAction] = useState<"use" | "save" | null>(null);
  const setAll = (choices: ChoiceJson[]) => change((d) => updateElement(d, el.name, { choices: orderOptions(choices) }));
  const update = (value: string, p: Partial<ChoiceJson>) => setAll(all.map((x) => (x.value === value ? clean({ ...x, ...p }) : x)));
  const move = (value: string, delta: number) => {
    const list = [...normal];
    const i = list.findIndex((o) => o.value === value);
    const j = i + delta;
    if (j < 0 || j >= list.length) return;
    [list[i], list[j]] = [list[j], list[i]];
    setAll([...list, ...all.filter((o) => o.isExclusive || o.special)]);
  };

  const row = (o: ChoiceJson, i: number, editable: boolean) => (
    <Group key={o.value} gap={4} wrap="nowrap">
      <BlurText
        style={{ flex: 1 }}
        size="xs"
        aria-label={`Option ${i + 1}`}
        value={o.text}
        disabled={readOnly}
        onSave={(text) => text.trim() && update(o.value, { text: text.trim() })}
      />
      {scores && (
        <BlurNumber w={64} size="xs" aria-label={`Score for ${o.text}`} placeholder="Score" value={o.score} disabled={readOnly} onSave={(score) => update(o.value, { score })} />
      )}
      {!readOnly && editable && !fixed && (
        <>
          <ActionIcon variant="subtle" color="gray" aria-label="Move option up" disabled={i === 0} onClick={() => move(o.value, -1)}>
            <IconArrowUp size={14} />
          </ActionIcon>
          <ActionIcon variant="subtle" color="gray" aria-label="Move option down" disabled={i === normal.length - 1} onClick={() => move(o.value, 1)}>
            <IconArrowDown size={14} />
          </ActionIcon>
          <Tooltip label={normal.length <= min ? `At least ${min} option${min > 1 ? "s" : ""}` : "Delete option"}>
            <ActionIcon variant="subtle" color="red" aria-label="Delete option" disabled={normal.length <= min} onClick={() => setAll(all.filter((x) => x.value !== o.value))}>
              <IconTrash size={14} />
            </ActionIcon>
          </Tooltip>
        </>
      )}
    </Group>
  );

  const special = (key: SpecialOption) => all.find((o) => o.special === key);
  return (
    <div>
      <Group justify="space-between" mb={6}>
        <Text size="sm" fw={500}>Options</Text>
        {!readOnly && !fixed && (
          <Button size="compact-xs" variant="subtle" leftSection={<IconPlus size={14} />} onClick={() => change((d) => addChoice(d, el.name, `Option ${normal.length + 1}`))}>
            Add
          </Button>
        )}
      </Group>
      <Stack gap={6}>{normal.map((o, i) => row(o, i, true))}</Stack>
      {!readOnly && !fixed && (
        <Group gap={4} mt={6}>
          <Button size="compact-xs" variant="subtle" leftSection={<IconList size={14} />} onClick={() => setListAction("use")}>
            Use a saved list
          </Button>
          <Button size="compact-xs" variant="subtle" leftSection={<IconDeviceFloppy size={14} />} onClick={() => setListAction("save")}>
            Save as a list
          </Button>
          <Button size="compact-xs" variant="subtle" leftSection={<IconBooks size={14} />} onClick={() => setImporting(true)}>
            From the code library
          </Button>
        </Group>
      )}
      <UseOptionList el={el} opened={listAction === "use"} onClose={() => setListAction(null)} change={change} />
      <SaveOptionList el={el} opened={listAction === "save"} onClose={() => setListAction(null)} />
      <Switch
        mt="sm"
        size="xs"
        label="Scores"
        description="Points for each option, totalled by a Calculation (for example STOP-BANG)."
        checked={scores}
        disabled={readOnly}
        onChange={(e) => {
          const on = e.currentTarget.checked;
          setScores(on);
          if (!on && all.some((o) => o.score !== undefined)) setAll(all.map(({ score, ...rest }) => (void score, rest)));
        }}
      />

      <Stack gap={8} mt="md">
        <Text size="sm" fw={500}>Special Options</Text>
        {kind === "selectmany" && (
          <>
            <Switch
              size="xs"
              label="No to all"
              description="An exclusive option shown after “or”. Choosing it clears the others."
              checked={!!none}
              disabled={readOnly}
              onChange={(e) => {
                const on = e.currentTarget.checked;
                change((d) => setNoneOption(d, el.name, on ? "None of the above" : undefined));
              }}
            />
            {none && row(none, 0, false)}
          </>
        )}
        {(Object.keys(SPECIAL_OPTIONS) as SpecialOption[]).map((key) => (
          <div key={key}>
            <Switch
              size="xs"
              label={SPECIAL_OPTIONS[key]}
              checked={!!special(key)}
              disabled={readOnly}
              onChange={(e) => {
                const on = e.currentTarget.checked;
                change((d) => setSpecialOption(d, el.name, key, on));
              }}
            />
            {special(key) && <div style={{ marginTop: 6 }}>{row(special(key)!, 0, false)}</div>}
          </div>
        ))}
        {kind !== "yesno" && (
          <Switch
            size="xs"
            label="Other (please specify)"
            description="Adds an Other option that opens a text box."
            checked={!!el.showOtherItem}
            disabled={readOnly}
            onChange={(e) => {
              const on = e.currentTarget.checked;
              change((d) => updateElement(d, el.name, { showOtherItem: on || undefined, otherText: on ? "Other (please specify)" : undefined }));
            }}
          />
        )}
        {kind === "selectmany" && (
          <Switch
            size="xs"
            label="Select all"
            checked={!!el.showSelectAllItem}
            disabled={readOnly}
            onChange={(e) => {
              const on = e.currentTarget.checked;
              change((d) => updateElement(d, el.name, { showSelectAllItem: on || undefined }));
            }}
          />
        )}
      </Stack>

      <CodeImport
        opened={importing}
        onClose={() => setImporting(false)}
        onAdd={(codes) => {
          const taken = new Set(all.map((o) => o.value));
          const added = codes.map((c) => {
            const value = newId("o", 4, (id) => taken.has(id));
            taken.add(value);
            return { value, text: c.display, clinicalOutputs: [{ id: newId("out", 6), codes: [c] }] };
          });
          setAll([...all, ...added]);
          setImporting(false);
        }}
      />
    </div>
  );
}

const clean = (c: ChoiceJson): ChoiceJson => {
  const out = { ...c };
  for (const k of Object.keys(out) as (keyof ChoiceJson)[]) if (out[k] === undefined) delete out[k];
  return out;
};

/** Pick codes; each becomes an option carrying that code as a disclosure (QT-07). */
function CodeImport({ opened, onClose, onAdd }: { opened: boolean; onClose: () => void; onAdd: (codes: CodeRef[]) => void }) {
  const [codes, setCodes] = useState<CodeRef[]>([]);
  return (
    <Modal opened={opened} onClose={onClose} title="Add options from the code library" size="lg">
      <Stack>
        <Text size="sm" c="dimmed">Each code becomes an option, with the code attached as its disclosure.</Text>
        <CodePicker value={codes} onChange={setCodes} />
        <Group justify="flex-end">
          <Button variant="default" onClick={onClose}>Cancel</Button>
          <Button
            disabled={!codes.length}
            onClick={() => {
              onAdd(codes);
              setCodes([]);
            }}
          >
            Add {codes.length || ""} option{codes.length === 1 ? "" : "s"}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}

/** Grid rows or columns: labels with stable IDs, reorder, add and delete (QT-10). */
export function ItemList({ el, list, label, prefix, readOnly, change }: { el: ElementJson; list: "rows" | "columns"; label: string; prefix: string; readOnly: boolean; change: Change }) {
  const items = (Array.isArray(el[list]) ? (el[list] as (ChoiceJson | string)[]) : []).map((c) =>
    typeof c === "string" ? { value: c, text: c } : { ...c, text: textOf(c.text) || String(c.value) },
  );
  const save = (next: ChoiceJson[]) => change((d) => updateElement(d, el.name, { [list]: next }));
  const swap = (i: number, j: number) => {
    if (j < 0 || j >= items.length) return;
    const next = [...items];
    [next[i], next[j]] = [next[j], next[i]];
    save(next);
  };
  return (
    <div>
      <Group justify="space-between" mb={6}>
        <Text size="sm" fw={500}>{label}</Text>
        {!readOnly && (
          <Button
            size="compact-xs"
            variant="subtle"
            leftSection={<IconPlus size={14} />}
            onClick={() => save([...items, { value: newId(prefix, 4, (id) => items.some((x) => x.value === id)), text: `${label.slice(0, -1)} ${items.length + 1}` }])}
          >
            Add
          </Button>
        )}
      </Group>
      <Stack gap={6}>
        {items.map((it, i) => (
          <Group key={it.value} gap={4} wrap="nowrap">
            <BlurText style={{ flex: 1 }} size="xs" aria-label={`${label} ${i + 1}`} value={it.text} disabled={readOnly} onSave={(text) => text.trim() && save(items.map((x) => (x.value === it.value ? { ...x, text: text.trim() } : x)))} />
            {!readOnly && (
              <>
                <ActionIcon variant="subtle" color="gray" aria-label="Move up" disabled={i === 0} onClick={() => swap(i, i - 1)}>
                  <IconArrowUp size={14} />
                </ActionIcon>
                <ActionIcon variant="subtle" color="gray" aria-label="Move down" disabled={i === items.length - 1} onClick={() => swap(i, i + 1)}>
                  <IconArrowDown size={14} />
                </ActionIcon>
                <ActionIcon variant="subtle" color="red" aria-label="Delete" disabled={items.length <= 1} onClick={() => save(items.filter((x) => x.value !== it.value))}>
                  <IconTrash size={14} />
                </ActionIcon>
              </>
            )}
          </Group>
        ))}
      </Stack>
    </div>
  );
}
