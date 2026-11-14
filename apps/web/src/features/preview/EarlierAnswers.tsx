// A one-set preview can't see the earlier Question Sets' answers (S38). When a score here uses them,
// the preview says so, and lets the author give them, as a test case can (TestCase.earlier).
import { useEffect, useState } from "react";
import { Alert, MultiSelect, Select, Stack, Text, TextInput } from "@mantine/core";
import { IconArrowBackUp } from "@tabler/icons-react";
import { findElement, kindOf, optionsOf, textOf, type Answers, type ChapterJson } from "@poised/clinical";
import { api } from "@/lib/api";
import { typedNumber } from "@/lib/numberBoxes";

type Detail = { chapters: { id: string; name: string }[] };

/** The earlier Question Sets, each with its JSON, once loaded; `[]` if they can't be. Only fetched when `wanted`. */
export function useEarlierSets(base: string, questionnaireId: string, chapterId: string, wanted: boolean): EarlierSet[] | null {
  const [earlier, setEarlier] = useState<EarlierSet[] | null>(null);
  useEffect(() => {
    if (!wanted) return;
    let current = true;
    (async () => {
      const d = await api<Detail>(`${base}/questionnaires/${questionnaireId}`);
      const before = d.chapters.slice(0, d.chapters.findIndex((c) => c.id === chapterId));
      const docs = await Promise.all(before.map((c) => api<{ content: ChapterJson }>(`${base}/chapters/${c.id}`)));
      if (current) setEarlier(before.map((c, i) => ({ name: c.name, doc: docs[i].content ?? {} })));
    })().catch(() => current && setEarlier([]));
    return () => {
      current = false;
    };
  }, [wanted, base, questionnaireId, chapterId]);
  return earlier;
}

export type EarlierSet = { name: string; doc: ChapterJson };

/**
 * The note, and an answer box for each earlier question the scores use. `keys` are the keys used
 * (earlierKeysUsed); `values` the answers given, by key (a BMI question's as `{ bmi }`). `hidden` names
 * the earlier sets the sample patient wouldn't see: their answers don't count (test report 2, F-1).
 */
export function EarlierAnswers(props: { earlier: ChapterJson | null; keys: string[]; values: Answers; hidden: string[]; onChange: (values: Answers) => void }) {
  const { earlier, keys, values, hidden, onChange } = props;
  const set = (key: string, value: unknown) => onChange({ ...values, [key]: value });
  return (
    <Alert color="blue" icon={<IconArrowBackUp size={16} />} mb="md" p="sm" data-testid="earlier-answers">
      <Text size="sm">Uses answers from earlier Question Sets, which count as unanswered here. To try them, give them below.</Text>
      {hidden.length > 0 && (
        <Text size="sm" c="orange.8" mt={4} data-testid="earlier-hidden">
          The sample patient wouldn&apos;t see {hidden.map((n) => `“${n}”`).join(", ")}, so {hidden.length === 1 ? "its" : "their"} answers don&apos;t count.
        </Text>
      )}
      {!earlier ? (
        <Text size="xs" c="dimmed" mt={4}>Loading the earlier Question Sets…</Text>
      ) : (
        <Stack gap={6} mt="xs">
          {keys.map((key) => {
            const found = findElement(earlier, key);
            if (!found) return <Text key={key} size="xs" c="red">{key}: not in an earlier Question Set</Text>;
            const { el, page } = found;
            const label = `${textOf(page.title)} › ${textOf(el.title) || el.name}`;
            const kind = kindOf(el);
            const common = { size: "xs" as const, label };
            const options = optionsOf(el).map((o) => ({ value: o.value, label: o.score !== undefined ? `${o.text} (${o.score} pt${o.score === 1 ? "" : "s"})` : o.text }));
            if (kind === "selectmany") {
              const v = values[key];
              return <MultiSelect key={key} {...common} data={options} value={Array.isArray(v) ? (v as string[]) : []} onChange={(x) => set(key, x.length ? x : undefined)} clearable />;
            }
            if (kind === "yesno" || kind === "selectone") {
              const v = values[key];
              return <Select key={key} {...common} data={options} value={typeof v === "string" ? v : null} onChange={(x) => set(key, x ?? undefined)} clearable />;
            }
            if (kind === "bmi") {
              const v = (values[key] as { bmi?: number } | undefined)?.bmi;
              return <PlainNumber key={key} {...common} label={`${label} (BMI)`} value={v} onChange={(n) => set(key, n === undefined ? undefined : { bmi: n })} />;
            }
            const v = values[key];
            return <PlainNumber key={key} {...common} value={typeof v === "number" ? v : undefined} onChange={(n) => set(key, n)} />;
          })}
        </Stack>
      )}
    </Alert>
  );
}

/**
 * A number answer as a plain text box (S49a, Rahul 14 Nov): no up/down arrows, and no wheel or
 * arrow key steps it (verification 1, finding 2). It shows what was typed while that is the value.
 */
function PlainNumber({ label, size, value, onChange }: { label: string; size: "xs"; value: number | undefined; onChange: (n: number | undefined) => void }) {
  const [typed, setTyped] = useState<string | null>(null);
  const text = typed !== null && typedNumber(typed) === value ? typed : value === undefined ? "" : String(value);
  return (
    <TextInput size={size} label={label} inputMode="decimal" value={text}
      onChange={(e) => {
        const t = e.currentTarget.value;
        setTyped(t);
        onChange(typedNumber(t));
      }}
      error={typed !== null && typed.trim() && typedNumber(typed) === undefined ? "A number" : undefined} />
  );
}
