// The settings that differ by type: text format, number range and unit, date format, list
// display, ticks, rating scale, grid rows and columns, calculations and bands, uploads and
// repeating groups (QT-02..12, VAL-02/03, CAL-01/02, STR-04).
import { useEffect, useMemo, useState } from "react";
import { ActionIcon, Button, Group, MultiSelect, NumberInput, Paper, Radio, SegmentedControl, Select, Stack, Switch, Text } from "@mantine/core";
import { IconPlus, IconTrash } from "@tabler/icons-react";
import {
  DATE_FORMATS, TEXT_FORMATS, allElements, bandsOf, calcFormOf, calcPatch, choiceDisplayOf, emptyCalc, dateSettingsOf, formatCalc, isValidExpression,
  kindOf, linesToSave, newId, numberSettingsOf, optionsOf, pointsSubjects, scoredQuestions, repeatSettingsOf, setChoiceDisplay, setDateSettings, setNumberSettings,
  setRepeating, setTextFormat, textFormatOf, textOf, unknownReferences, updateElement, type Band, type CalcForm, type ChapterJson,
  type DateFormat, type DateRange, type ElementJson, type PointsLine, type ScoreForm, type TextFormat,
} from "@poised/clinical";
import { RuleRow } from "./ConditionBuilder";
import { BlurNumber, BlurText, BlurTextarea } from "./inputs";
import { ItemList, OptionsEditor } from "./OptionsEditor";

type Change = (e: (d: ChapterJson) => ChapterJson) => void;
/** `earlier`: the earlier Question Sets combined, once loaded; a score may use their questions (S38). */
type Props = { doc: ChapterJson; el: ElementJson; readOnly: boolean; change: Change; earlier?: ChapterJson };

export function TypeSettings(props: Props) {
  switch (kindOf(props.el)) {
    case "text":
      return <TextSettings {...props} />;
    case "number":
      return <NumberSettingsForm {...props} />;
    case "date":
      return <DateSettingsForm {...props} />;
    case "yesno":
    case "selectone":
    case "selectmany":
      return <ChoiceSettings {...props} />;
    case "rating":
      return <RatingSettings {...props} />;
    case "grid":
      return (
        <>
          <ItemList el={props.el} list="rows" label="Rows" prefix="r" readOnly={props.readOnly} change={props.change} />
          <ItemList el={props.el} list="columns" label="Columns" prefix="o" readOnly={props.readOnly} change={props.change} />
        </>
      );
    case "calculation":
      return <CalculationSettings {...props} />;
    case "upload":
      return <UploadSettings {...props} />;
    case "group":
      return <RepeatSettingsForm {...props} />;
    default:
      return null;
  }
}

const usePatch = ({ el, change }: Props) => (p: Partial<ElementJson>) => change((d) => updateElement(d, el.name, p));

function Placeholder(props: Props) {
  const patch = usePatch(props);
  return (
    <BlurText
      label="Placeholder"
      description="Shown in the empty box"
      value={textOf(props.el.placeholder)}
      disabled={props.readOnly}
      onSave={(v) => patch({ placeholder: v || undefined })}
    />
  );
}

// ---------------------------------------------------------------- text (QT-04)

function TextSettings(props: Props) {
  const { el, readOnly } = props;
  const patch = usePatch(props);
  const f = textFormatOf(el);
  return (
    <>
      <Radio.Group label="Input length" value={el.type === "comment" ? "long" : "short"} onChange={(v) => patch({ type: v === "long" ? "comment" : "text" })}>
        <Group mt={6}>
          <Radio value="short" label="Short" disabled={readOnly} />
          <Radio value="long" label="Long" disabled={readOnly} />
        </Group>
      </Radio.Group>
      <Placeholder {...props} />
      <BlurNumber label="Maximum length" description="Characters; empty for no limit" min={1} value={el.maxLength as number | undefined} disabled={readOnly} onSave={(maxLength) => patch({ maxLength })} />
      {el.type === "text" && (
        <>
          <Select
            label="Format"
            data={Object.entries(TEXT_FORMATS).map(([value, x]) => ({ value, label: x.label }))}
            value={f.format}
            allowDeselect={false}
            disabled={readOnly}
            onChange={(v) => patch(setTextFormat(el, v as TextFormat, f.pattern, undefined))}
          />
          {f.format === "custom" && (
            <BlurText
              label="Pattern"
              description="A regular expression, for example ^[A-Z]{2}\d{6}$"
              value={f.pattern ?? ""}
              disabled={readOnly}
              styles={{ input: { fontFamily: "monospace" } }}
              onSave={(pattern) => patch(setTextFormat(el, "custom", pattern, f.message))}
            />
          )}
          {f.format !== "none" && (
            <BlurText label="Message when the format is wrong" value={f.message ?? ""} disabled={readOnly} onSave={(m) => patch(setTextFormat(el, f.format, f.pattern, m || undefined))} />
          )}
        </>
      )}
    </>
  );
}

// ---------------------------------------------------------------- number (QT-06, VAL-02/03)

function NumberSettingsForm(props: Props) {
  const { el, readOnly } = props;
  const patch = usePatch(props);
  const s = numberSettingsOf(el);
  const set = (p: Partial<typeof s>) => patch(setNumberSettings(el, { ...s, ...p }));
  return (
    <>
      <Placeholder {...props} />
      <Group grow>
        <BlurNumber label="Minimum" value={s.min} disabled={readOnly} onSave={(min) => set({ min })} />
        <BlurNumber label="Maximum" value={s.max} disabled={readOnly} onSave={(max) => set({ max })} />
      </Group>
      <Group grow>
        <Select
          label="Decimal places"
          data={[
            { value: "any", label: "Any" },
            { value: "0", label: "Whole numbers" },
            { value: "1", label: "1" },
            { value: "2", label: "2" },
            { value: "3", label: "3" },
          ]}
          value={s.decimals === undefined ? "any" : String(s.decimals)}
          allowDeselect={false}
          disabled={readOnly}
          onChange={(v) => set({ decimals: v === "any" ? undefined : Number(v) })}
        />
        <BlurText label="Unit" placeholder="e.g. kg" value={s.unit ?? ""} disabled={readOnly} onSave={(unit) => set({ unit: unit || undefined })} />
      </Group>
      <div>
        <Text size="sm" fw={500}>Soft warning</Text>
        <Text size="xs" c="dimmed" mb={6}>Outside this range the patient is asked to check, but can carry on.</Text>
        <Group grow>
          <BlurNumber size="xs" label="Below" value={s.warnMin} disabled={readOnly} onSave={(warnMin) => set({ warnMin })} />
          <BlurNumber size="xs" label="Above" value={s.warnMax} disabled={readOnly} onSave={(warnMax) => set({ warnMax })} />
        </Group>
        {(s.warnMin !== undefined || s.warnMax !== undefined) && (
          <BlurText mt={6} size="xs" label="Warning" value={s.warnText ?? ""} disabled={readOnly} onSave={(warnText) => set({ warnText: warnText || undefined })} />
        )}
      </div>
    </>
  );
}

// ---------------------------------------------------------------- date (QT-05)

function DateSettingsForm({ el, readOnly, change }: Props) {
  const s = dateSettingsOf(el);
  const set = (p: Partial<typeof s>) => change((d) => setDateSettings(d, el.name, { ...s, ...p }));
  return (
    <>
      <Select
        label="Format"
        data={Object.entries(DATE_FORMATS).map(([value, label]) => ({ value, label }))}
        value={s.format}
        allowDeselect={false}
        disabled={readOnly}
        onChange={(v) => set({ format: v as DateFormat })}
      />
      <div>
        <Text size="sm" fw={500} mb={4}>Allowed dates</Text>
        <SegmentedControl
          size="xs"
          value={s.range}
          disabled={readOnly}
          onChange={(v) => set({ range: v as DateRange })}
          data={[
            { value: "any", label: "Any (from 1900)" },
            { value: "past", label: "Past only" },
            { value: "future", label: "Future only" },
          ]}
        />
      </div>
      <Switch label="Allow several dates" description="The patient can add more than one date, as a list." checked={s.multiple} disabled={readOnly} onChange={(e) => set({ multiple: e.currentTarget.checked })} />
    </>
  );
}

// ---------------------------------------------------------------- choices (QT-02/03/07)

function ChoiceSettings(props: Props) {
  const { el, readOnly, change } = props;
  const patch = usePatch(props);
  const kind = kindOf(el);
  const display = choiceDisplayOf(el);
  return (
    <>
      {kind !== "yesno" && (
        <div>
          <Text size="sm" fw={500} mb={4}>Show options as</Text>
          <SegmentedControl
            size="xs"
            value={display}
            disabled={readOnly}
            onChange={(v) => patch(setChoiceDisplay(el, v as "buttons" | "list"))}
            data={[
              { value: "buttons", label: kind === "selectmany" ? "Checkboxes" : "Buttons" },
              { value: "list", label: kind === "selectmany" ? "Searchable list" : "Dropdown list" },
            ]}
          />
        </div>
      )}
      {display === "list" && <Placeholder {...props} />}
      <OptionsEditor el={el} readOnly={readOnly} change={change} />
      {kind === "selectmany" && (
        <Group grow>
          <BlurNumber label="At least" description="Ticks needed" min={1} value={el.minSelectedChoices as number | undefined} disabled={readOnly} onSave={(minSelectedChoices) => patch({ minSelectedChoices })} />
          <BlurNumber label="At most" description="Ticks allowed" min={1} value={el.maxSelectedChoices as number | undefined} disabled={readOnly} onSave={(maxSelectedChoices) => patch({ maxSelectedChoices })} />
        </Group>
      )}
    </>
  );
}

// ---------------------------------------------------------------- rating (QT-11)

function RatingSettings(props: Props) {
  const { el, readOnly } = props;
  const patch = usePatch(props);
  return (
    <>
      <Group grow>
        <BlurNumber label="From" value={(el.rateMin as number | undefined) ?? 1} disabled={readOnly} onSave={(rateMin) => patch({ rateMin: rateMin ?? 1 })} />
        <BlurNumber label="To" value={(el.rateMax as number | undefined) ?? 5} disabled={readOnly} onSave={(rateMax) => patch({ rateMax: rateMax ?? 5 })} />
      </Group>
      <Group grow>
        <BlurText label="Low label" value={textOf(el.minRateDescription)} disabled={readOnly} onSave={(v) => patch({ minRateDescription: v || undefined })} />
        <BlurText label="High label" value={textOf(el.maxRateDescription)} disabled={readOnly} onSave={(v) => patch({ maxRateDescription: v || undefined })} />
      </Group>
    </>
  );
}

// ---------------------------------------------------------------- calculation (CAL-01/02/03)

function CalculationSettings(props: Props) {
  const { doc, el, readOnly } = props;
  const patch = usePatch(props);
  const form = calcFormOf(el.expression as string | undefined, el.calcKind as string | undefined);
  const others = allElements(doc).filter(({ el: e }) => e.name !== el.name);
  const pick = (test: (e: ElementJson) => boolean) => others.filter(({ el: e }) => test(e)).map(({ el: e }) => ({ value: e.name, label: textOf(e.title) || e.name }));
  // Each earlier set's scored questions, grouped "Set › Page", then this set's, each key once (S38).
  const scored = scoredQuestions(doc, props.earlier, el.name);
  // A question picked whose set isn't loaded yet still shows (as its key).
  const known = new Set(scored.flatMap((g) => g.items.map((i) => i.value)));
  const missing = form.kind === "score" ? form.questions.filter((q) => !known.has(q)).map((q) => ({ value: q, label: q })) : [];
  const dates = pick((e) => e.type === "text" && e.inputType === "date");
  const numbers = pick((e) => kindOf(e) === "number");
  const set = (f: CalcForm) => patch(calcPatch(f));
  const custom = form.kind === "custom" ? form.expression : "";
  // A formula may use earlier sets' questions too (S38).
  const earlierNames = new Set(allElements(props.earlier ?? {}).map(({ el: e }) => e.name));
  const unknown = custom ? unknownReferences(doc, custom).filter((r) => !earlierNames.has(r)) : [];
  const customError = custom && !isValidExpression(custom) ? "This isn't a valid formula." : unknown.length ? `Unknown question: ${unknown.join(", ")}` : undefined;

  return (
    <>
      <Select
        label="Calculate"
        data={[
          { value: "score", label: "A score total" },
          { value: "years", label: "Years since a date (for example, age)" },
          { value: "bmi", label: "BMI from height and weight" },
          { value: "custom", label: "A custom formula" },
        ]}
        value={form.kind}
        allowDeselect={false}
        disabled={readOnly}
        onChange={(v) =>
          // A custom formula starts from the current one, so it can be adjusted rather than retyped.
          set(v === "custom" ? { kind: "custom", expression: formatCalc(form) } : emptyCalc(v ?? undefined))
        }
      />
      {form.kind === "score" && (
        <>
          <MultiSelect
            label="Add up the scores of"
            description={
              scored.length
                ? "Questions whose options have scores, in this Question Set or an earlier one (switch on Scores in their options)"
                : "No question has scored options yet. Switch on Scores in a question's options."
            }
            data={[...scored, ...(missing.length ? [{ group: "Not found", items: missing }] : [])]}
            searchable
            value={form.questions}
            disabled={readOnly}
            onChange={(questions) => set({ ...form, questions })}
          />
          <PointsLines {...props} form={form} onChange={(lines) => set({ ...form, lines })} />
          <Text size="xs" c="dimmed">Incomplete until every question it adds up and every line has an answer: an Incomplete score has no band.</Text>
        </>
      )}
      {form.kind === "years" && <Select label="Date" data={dates} value={form.question || null} disabled={readOnly} onChange={(q) => set({ kind: "years", question: q ?? "" })} />}
      {form.kind === "bmi" && (
        <Group grow>
          <Select label="Height (cm)" data={numbers} value={form.height || null} disabled={readOnly} onChange={(h) => set({ ...form, height: h ?? "" })} />
          <Select label="Weight (kg)" data={numbers} value={form.weight || null} disabled={readOnly} onChange={(w) => set({ ...form, weight: w ?? "" })} />
        </Group>
      )}
      {form.kind === "custom" && (
        <BlurTextarea
          label="Formula"
          description="SurveyJS expression, e.g. {q_abcdef} * 2. Functions: score(), age(), bmi(), round(), iif(), dateDiff()."
          value={custom}
          disabled={readOnly}
          styles={{ input: { fontFamily: "monospace" } }}
          error={customError}
          onSave={(expression) => set({ kind: "custom", expression })}
        />
      )}
      <BandsEditor {...props} />
    </>
  );
}

/**
 * "Add N points when…" lines (S38, SCR-01/02): each picks what it tests from menus (Patient,
 * Measurements, Other scores, then answers in this or an earlier Question Set), so authors never
 * type a formula. Unfinished lines stay here until complete; only complete ones are saved.
 */
function PointsLines(props: Props & { form: ScoreForm; onChange: (lines: PointsLine[]) => void }) {
  const { doc, el, readOnly, earlier, form, onChange } = props;
  const subjects = useMemo(() => pointsSubjects(doc, earlier, el.name), [doc, earlier, el.name]);
  const saved = form.lines ?? [];
  const [draft, setDraft] = useState<PointsLine[]>(saved);
  const key = JSON.stringify(saved);
  useEffect(() => {
    if (linesToSave(draft, saved)) setDraft(saved);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);
  // The complete lines are saved whenever they change, even while another line is unfinished (verification 1, F1).
  const update = (next: PointsLine[]) => {
    setDraft(next);
    const save = linesToSave(next, saved);
    if (save) onChange(save);
  };
  return (
    <div data-testid="points-lines">
      <Text size="sm" fw={500}>Add points when…</Text>
      <Text size="xs" c="dimmed" mb={6}>For example, 1 point when the patient&apos;s age is over 50, or when an answer in an earlier Question Set is Yes.</Text>
      <Stack gap={6}>
        {draft.map((line, i) => (
          <Paper key={i} withBorder p={6}>
            <Group gap={6} wrap="nowrap" align="flex-start" mb={6}>
              <Text size="xs" mt={6}>Add</Text>
              <NumberInput
                size="xs"
                w={70}
                aria-label="Points"
                value={Number.isFinite(line.points) ? line.points : ""}
                disabled={readOnly}
                onChange={(v) => update(draft.map((x, j) => (j === i ? { ...x, points: v === "" ? NaN : Number(v) } : x)))}
              />
              <Text size="xs" mt={6}>point{line.points === 1 ? "" : "s"} when</Text>
            </Group>
            <RuleRow
              subjects={subjects}
              rule={line.rule}
              readOnly={readOnly}
              answered={false}
              placeholder="Select…"
              removeLabel="Remove line"
              onChange={(rule) => update(rule ? draft.map((x, j) => (j === i ? { ...x, rule } : x)) : draft.filter((_, j) => j !== i))}
            />
          </Paper>
        ))}
      </Stack>
      {!readOnly && (
        <Button size="compact-xs" variant="subtle" mt={6} leftSection={<IconPlus size={14} />} onClick={() => setDraft([...draft, { points: 1, rule: { left: "", op: "eq" } }])}>
          Add points line
        </Button>
      )}
    </div>
  );
}

/** Score bands (CAL-02): each up to a maximum, the last "and above". Each can carry disclosures. */
function BandsEditor(props: Props) {
  const { el, readOnly } = props;
  const patch = usePatch(props);
  const bands = bandsOf(el);
  const save = (next: Band[]) => {
    const fixed = next.map((b, i) => {
      const { max, ...rest } = b;
      return i === next.length - 1 || max === undefined ? rest : { ...rest, max };
    });
    patch({ bands: fixed.length ? fixed : undefined });
  };
  return (
    <div>
      <Group justify="space-between" mb={4}>
        <Text size="sm" fw={500}>Bands</Text>
        {!readOnly && (
          <Button
            size="compact-xs"
            variant="subtle"
            leftSection={<IconPlus size={14} />}
            onClick={() => {
              const lastMax = bands.length ? (bands[bands.length - 1].max ?? (bands.length > 1 ? (bands[bands.length - 2].max ?? 0) + 2 : 2)) : undefined;
              const prev = bands.map((b, i) => (i === bands.length - 1 ? { ...b, max: lastMax } : b));
              save([...prev, { id: newId("b", 4, (id) => bands.some((b) => b.id === id)), label: bands.length ? "High" : "Low" }]);
            }}
          >
            Add band
          </Button>
        )}
      </Group>
      <Text size="xs" c="dimmed" mb={6}>For example STOP-BANG: Low up to 2, Intermediate up to 4, High above. Conditions and disclosures can use the band.</Text>
      <Stack gap={6}>
        {bands.map((b, i) => (
          <div key={b.id}>
          <Group gap={6} wrap="nowrap">
            <BlurText size="xs" style={{ flex: 1 }} aria-label={`Band ${i + 1}`} value={b.label} disabled={readOnly} onSave={(label) => label.trim() && save(bands.map((x) => (x.id === b.id ? { ...x, label: label.trim() } : x)))} />
            {i < bands.length - 1 ? (
              <BlurNumber size="xs" w={90} aria-label="Up to" leftSection={<Text size="xs">≤</Text>} value={b.max} disabled={readOnly} onSave={(max) => save(bands.map((x) => (x.id === b.id ? { ...x, max } : x)))} />
            ) : (
              <Text size="xs" c="dimmed" w={90}>{bands.length > 1 ? "above" : "any value"}</Text>
            )}
            {!readOnly && (
              <ActionIcon variant="subtle" color="red" aria-label="Delete band" onClick={() => save(bands.filter((x) => x.id !== b.id))}>
                <IconTrash size={14} />
              </ActionIcon>
            )}
          </Group>
          {/* A ready-made score's "also when" rule (S38). */}
          {b.when && <Text size="xs" c="dimmed" mt={2}>Also {b.label} when: {b.whenText ?? b.when}</Text>}
          </div>
        ))}
      </Stack>
    </div>
  );
}

// ---------------------------------------------------------------- upload (QT-12)

const FILE_TYPES = [
  { value: "image/*,application/pdf", label: "Images and PDFs" },
  { value: "image/*", label: "Images only" },
  { value: "application/pdf", label: "PDFs only" },
  { value: "", label: "Any file" },
];

function UploadSettings(props: Props) {
  const { el, readOnly } = props;
  const patch = usePatch(props);
  return (
    <>
      <Select
        label="Allowed files"
        data={FILE_TYPES}
        value={String(el.acceptedTypes ?? "")}
        allowDeselect={false}
        disabled={readOnly}
        onChange={(v) => patch({ acceptedTypes: v || undefined })}
      />
      <BlurNumber
        label="Largest file (MB)"
        min={1}
        max={20}
        value={el.maxSize ? Math.round(Number(el.maxSize) / 1048576) : undefined}
        disabled={readOnly}
        onSave={(mb) => patch({ maxSize: mb ? mb * 1048576 : undefined })}
      />
      <Switch label="Allow several files" checked={!!el.allowMultiple} disabled={readOnly} onChange={(e) => patch({ allowMultiple: e.currentTarget.checked || undefined })} />
    </>
  );
}

// ---------------------------------------------------------------- repeating group (STR-04)

function RepeatSettingsForm({ el, readOnly, change }: Props) {
  const on = el.type === "paneldynamic";
  const s = repeatSettingsOf(el);
  const set = (p: Partial<typeof s>) => change((d) => setRepeating(d, el.name, { ...s, ...p }));
  return (
    <>
      <Switch
        label="Repeatable"
        description="The patient can add another, for example one entry per operation."
        checked={on}
        disabled={readOnly}
        onChange={(e) => {
          const next = e.currentTarget.checked;
          change((d) => setRepeating(d, el.name, next ? { min: 1, addText: "Add another" } : undefined));
        }}
      />
      {on && (
        <>
          <Group grow>
            <BlurNumber label="At least" min={0} value={s.min} disabled={readOnly} onSave={(min) => set({ min: min ?? 0 })} />
            <BlurNumber label="At most" description="Empty for no limit" min={1} value={s.max} disabled={readOnly} onSave={(max) => set({ max })} />
          </Group>
          <BlurText label="Add button" value={s.addText} disabled={readOnly} onSave={(addText) => set({ addText: addText || "Add another" })} />
        </>
      )}
    </>
  );
}
