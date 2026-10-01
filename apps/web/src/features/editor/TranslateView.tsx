"use client";
// Translations for one Question Set (LNG-01/02/03): English beside the chosen language, gaps
// flagged, CSV out and back in for a translator. Clinical notes and codes are never listed.
import { useMemo, useRef, useState } from "react";
import { Alert, Badge, Button, Group, SegmentedControl, Select, Stack, Table, Text, Title } from "@mantine/core";
import { IconDownload, IconUpload, IconX } from "@tabler/icons-react";
import {
  LANGUAGES, fromCsv, localesIn, missingCount, plainText, setTranslation, textEntries, toCsv, translationOf, type ChapterJson,
} from "@sj/clinical";
import { BlurTextarea } from "./inputs";

export function TranslateView(props: {
  doc: ChapterJson;
  setName: string;
  readOnly: boolean;
  change: (edit: (doc: ChapterJson) => ChapterJson) => void;
  onClose: () => void;
}) {
  const { doc, setName, readOnly, change, onClose } = props;
  const existing = localesIn(doc);
  const [locale, setLocale] = useState<string>(existing[0] ?? "de");
  const [filter, setFilter] = useState<"all" | "missing">("all");
  const [message, setMessage] = useState("");
  const file = useRef<HTMLInputElement>(null);
  const entries = useMemo(() => textEntries(doc), [doc]);
  const missing = missingCount(doc, locale);
  const languages = [...new Set([...Object.keys(LANGUAGES), ...existing])].map((code) => ({
    value: code,
    label: `${LANGUAGES[code] ?? code.toUpperCase()}${existing.includes(code) ? ` · ${missingCount(doc, code) ? `${missingCount(doc, code)} missing` : "complete"}` : ""}`,
  }));
  const shown = entries.filter((e) => filter === "all" || !translationOf(e.value, locale));

  const download = () => {
    const blob = new Blob([toCsv(doc, locale)], { type: "text/csv;charset=utf-8" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `${setName} - ${LANGUAGES[locale] ?? locale}.csv`;
    a.click();
    URL.revokeObjectURL(a.href);
  };
  const upload = async (f: File) => {
    const text = await f.text();
    const r = fromCsv(doc, locale, text);
    change(() => r.doc);
    setMessage(`Imported ${r.applied} translation${r.applied === 1 ? "" : "s"}.${r.skipped ? ` ${r.skipped} row${r.skipped === 1 ? " was" : "s were"} skipped because the English text has changed since the export, or the language column is missing.` : ""}`);
  };

  return (
    <Stack>
      <Group justify="space-between">
        <div>
          <Title order={4}>Translate · {setName}</Title>
          <Text size="sm" c="dimmed">Missing translations fall back to English. Clinical notes and codes are not translated.</Text>
        </div>
        <Button variant="default" leftSection={<IconX size={14} />} onClick={onClose}>Close</Button>
      </Group>
      <Group>
        <Select label="Language" data={languages} value={locale} onChange={(v) => v && setLocale(v)} allowDeselect={false} searchable w={280} />
        <SegmentedControl
          mt={24}
          value={filter}
          onChange={(v) => setFilter(v as "all" | "missing")}
          data={[
            { value: "all", label: `All (${entries.length})` },
            { value: "missing", label: `Missing (${missing})` },
          ]}
        />
        <Group gap="xs" mt={24} ml="auto">
          <Button size="xs" variant="default" leftSection={<IconDownload size={14} />} onClick={download}>Export CSV</Button>
          {!readOnly && (
            <>
              <Button size="xs" variant="default" leftSection={<IconUpload size={14} />} onClick={() => file.current?.click()}>Import CSV</Button>
              <input
                ref={file}
                type="file"
                accept=".csv,text/csv"
                hidden
                onChange={(e) => {
                  const f = e.currentTarget.files?.[0];
                  if (f) upload(f);
                  e.currentTarget.value = "";
                }}
              />
            </>
          )}
        </Group>
      </Group>
      {message && <Alert color="blue" withCloseButton onClose={() => setMessage("")}>{message}</Alert>}
      <Table withTableBorder verticalSpacing="xs" layout="fixed">
        <Table.Thead>
          <Table.Tr>
            <Table.Th w="22%">Where</Table.Th>
            <Table.Th w="34%">English</Table.Th>
            <Table.Th>{LANGUAGES[locale] ?? locale}</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {shown.map((e) => {
            const value = translationOf(e.value, locale);
            return (
              <Table.Tr key={e.path.join("/")}>
                <Table.Td>
                  <Text size="xs" c="dimmed" lineClamp={2}>{e.where}</Text>
                  <Badge size="xs" variant="light" color="gray">{e.field}</Badge>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" style={{ whiteSpace: "pre-wrap" }}>{e.english}</Text>
                </Table.Td>
                <Table.Td>
                  <BlurTextarea
                    aria-label={`${LANGUAGES[locale] ?? locale} for ${e.english}`}
                    minRows={1}
                    size="xs"
                    value={e.isHtml ? plainText(value) : value}
                    placeholder="Not translated"
                    error={!value ? " " : undefined}
                    disabled={readOnly}
                    onSave={(text) => change((d) => setTranslation(d, e.path, locale, text, e.isHtml))}
                  />
                </Table.Td>
              </Table.Tr>
            );
          })}
        </Table.Tbody>
      </Table>
      {shown.length === 0 && <Text c="dimmed" ta="center">{filter === "missing" ? "Everything is translated." : "This Question Set has no text yet."}</Text>}
    </Stack>
  );
}
