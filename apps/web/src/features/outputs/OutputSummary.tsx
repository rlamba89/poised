"use client";
// One clinical output in a line: its note with category, codes, ASA grade and flag.
import { Badge, Group, Stack, Text } from "@mantine/core";
import type { ClinicalOutput } from "@poised/clinical";

export function OutputSummary({ output }: { output: ClinicalOutput }) {
  return (
    <Stack gap={2}>
      {output.note && (
        <Text size="sm" component="div">
          {output.note.text} <Badge size="xs" variant="outline">{output.note.category}</Badge>
        </Text>
      )}
      <Group gap={4}>
        {output.codes.map((c) => (
          <Badge key={`${c.set}:${c.code}`} size="sm" variant="light">
            {c.set} {c.code} · {c.display}
          </Badge>
        ))}
        {output.asa && (
          <Badge size="sm" color="gray">ASA {output.asa.grade}{output.asa.emergency ? "E" : ""}</Badge>
        )}
        {output.flag && (
          <Badge size="sm" color={output.flag === "red" ? "red" : "orange"}>{output.flag} flag</Badge>
        )}
      </Group>
    </Stack>
  );
}
