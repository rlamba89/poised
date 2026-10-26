// The suggested ASA (S39, ASA-01): the highest grade the answers' disclosures give, with its reasons.
// It's only a suggestion: the nurse and the anaesthetist give the ASA grade.
import type { ReactNode } from "react";
import { List, Stack, Text } from "@mantine/core";
import { asaHeading, type AsaReason, type SuggestedAsa as Suggested } from "@poised/clinical";

/** The POA Summary's block: "Suggested ASA III", each reason (a link, where `link` is given), and what it means. */
export function SuggestedAsaBlock({ asa, link }: { asa: Suggested | undefined; link?: (reason: AsaReason) => ReactNode }) {
  if (!asa) {
    return (
      <Text size="sm" data-testid="suggested-asa">No suggested ASA: no answer gives an ASA grade.</Text>
    );
  }
  return (
    <Stack gap={4} data-testid="suggested-asa">
      <Text fw={700} c="blue.9">{asaHeading(asa)}</Text>
      <List size="sm">
        {asa.reasons.map((r) => (
          <List.Item key={r.text}>{link ? link(r) : r.text}</List.Item>
        ))}
      </List>
      <Text size="xs" c="dimmed">Suggested from the answers. The nurse and the anaesthetist give the ASA grade.</Text>
    </Stack>
  );
}

/** One line, for the top of the episode page's Question Sets. */
export function SuggestedAsaLine({ asa }: { asa: Suggested | undefined }) {
  return (
    <Text size="sm" mb="xs" data-testid="suggested-asa-line">
      <b>{asa ? asaHeading(asa) : "No suggested ASA"}</b>
      {asa ? `: ${asa.reasons.map((r) => r.text).join("; ")}` : ": no answer gives an ASA grade"}
      {asa && <Text span size="xs" c="dimmed"> (the anaesthetist confirms)</Text>}
    </Text>
  );
}
