// The Settings of an Information block (S35, INF-03): the text with simple marks, a short "how to
// format" list, and a live preview of what the patient will see. The text saves when the box loses
// focus (one undo step), as the other text boxes do; the preview follows every key press. The rules
// (marks → HTML, the cleaning, an old statement moving to marks) are in @poised/clinical
// (information.ts).
import { useEffect, useState } from "react";
import { Alert, Box, Code, Text, Textarea } from "@mantine/core";
import { informationPatch, informationPreview, informationText, refusedLinks, type ElementJson } from "@poised/clinical";

const HINTS: [string, string][] = [
  ["# Heading", "a heading (## and ### for smaller ones)"],
  ["**bold**", "bold"],
  ["*italic*", "italic"],
  ["- item", "a bulleted list (1. for a numbered one)"],
  ["[words](https://…)", "a link (also mailto: and tel:)"],
  ["![what it shows](https://…)", "a picture from the web"],
];

export function InformationSettings({ el, readOnly, patch }: { el: ElementJson; readOnly: boolean; patch: (p: Partial<ElementJson>) => void }) {
  const value = informationText(el);
  const [draft, setDraft] = useState(value);
  useEffect(() => setDraft(value), [value]);
  const refused = refusedLinks(draft);
  // The preview shows the draft as it would be saved (an old statement's HTML is replaced by the text).
  const preview = informationPreview({ ...el, markdown: draft });

  return (
    <>
      <Textarea
        label="Text"
        autosize
        minRows={5}
        value={draft}
        disabled={readOnly}
        onChange={(e) => setDraft(e.currentTarget.value)}
        onBlur={() => draft !== value && patch(informationPatch(el, draft))}
      />
      <Box data-testid="information-hints">
        <Text size="xs" fw={600} c="dimmed">How to format</Text>
        {HINTS.map(([mark, what]) => (
          <Text key={mark} size="xs" c="dimmed">
            <Code>{mark}</Code> {what}
          </Text>
        ))}
        <Text size="xs" c="dimmed">Each new line starts a new paragraph.</Text>
      </Box>
      {refused.length > 0 && (
        <Alert color="orange" p="xs" data-testid="information-refused">
          <Text size="xs">
            Shown as text, not as a link or picture: {refused.join(", ")}. A link must start with https://, mailto: or tel:, and a picture with https://.
          </Text>
        </Alert>
      )}
      <Box>
        <Text size="sm" fw={500} mb={4}>Preview</Text>
        <div
          className="sj-info sj-info-preview"
          data-testid="information-preview"
          // Cleaned by informationPreview: only safe tags, links and https pictures.
          dangerouslySetInnerHTML={{ __html: preview || "<p>Nothing to show yet.</p>" }}
        />
      </Box>
    </>
  );
}
