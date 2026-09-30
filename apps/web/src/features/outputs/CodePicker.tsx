"use client";
// Find codes by part of the code or description, within one code set (CLN-09).
import { useEffect, useState } from "react";
import { useDebouncedValue } from "@mantine/hooks";
import { Badge, CloseButton, Group, Paper, SegmentedControl, Stack, Text, TextInput, UnstyledButton } from "@mantine/core";
import type { CodeRef, CodeSet } from "@sj/clinical";
import { api } from "@/lib/api";

type CodeRow = { id: string; codeSet: CodeSet; code: string; description: string; categoryName: string | null };

export function CodePicker({ value, onChange }: { value: CodeRef[]; onChange: (codes: CodeRef[]) => void }) {
  const [set, setSet] = useState<CodeSet>("SNOMED");
  const [query, setQuery] = useState("");
  const [debounced] = useDebouncedValue(query.trim(), 200);
  const [results, setResults] = useState<CodeRow[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!debounced) {
      setResults([]);
      return;
    }
    let current = true;
    api<CodeRow[]>(`/codes?${new URLSearchParams({ set, q: debounced })}`)
      .then((rows) => current && setResults(rows))
      .catch((e: Error) => current && setError(e.message));
    return () => {
      current = false;
    };
  }, [set, debounced]);

  const add = (row: CodeRow) => {
    if (!value.some((c) => c.set === row.codeSet && c.code === row.code)) {
      onChange([...value, { set: row.codeSet, code: row.code, display: row.description }]);
    }
    setQuery("");
  };

  return (
    <Stack gap="xs">
      <Group gap="xs">
        {value.map((c) => (
          <Badge
            key={`${c.set}:${c.code}`}
            variant="light"
            size="lg"
            rightSection={
              <CloseButton size="xs" aria-label={`Remove ${c.code}`} onClick={() => onChange(value.filter((x) => x !== c))} />
            }
          >
            {c.set} {c.code} · {c.display}
          </Badge>
        ))}
        {value.length === 0 && <Text size="sm" c="dimmed">No codes.</Text>}
      </Group>
      <Group gap="xs" align="flex-start">
        <SegmentedControl value={set} onChange={(v) => setSet(v as CodeSet)} data={["SNOMED", "ICD10"]} />
        <TextInput
          style={{ flex: 1 }}
          placeholder={`Search ${set} by code or description`}
          value={query}
          onChange={(e) => setQuery(e.currentTarget.value)}
          error={error || undefined}
        />
      </Group>
      {results.length > 0 && (
        <Paper withBorder mah={220} style={{ overflowY: "auto" }}>
          {results.map((r) => (
            <UnstyledButton key={r.id} onClick={() => add(r)} w="100%" px="sm" py={6} className="sj-code-row">
              <Text size="sm">
                <b>{r.code}</b> {r.description}
                {r.categoryName && <Text span c="dimmed"> · {r.categoryName}</Text>}
              </Text>
            </UnstyledButton>
          ))}
        </Paper>
      )}
      {debounced && results.length === 0 && !error && <Text size="sm" c="dimmed">No active {set} codes match.</Text>}
    </Stack>
  );
}
