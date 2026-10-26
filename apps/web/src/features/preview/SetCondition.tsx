// The Question Set's own condition (LOG-02), in plain words above the preview. A one-set preview
// can't see earlier sets' answers, so it decides only conditions that test just the patient.
import { Alert, Text } from "@mantine/core";
import { IconGitBranch } from "@tabler/icons-react";
import { chapterConditionOf, describeLogic, isChapterShown, testsOnlyThePatient, type ChapterJson, type Viewer } from "@poised/clinical";
import { DisplaysWhen } from "@/features/editor/ElementCard";

/** `earlier`: the earlier Question Sets combined (useEarlierSets), once loaded. */
export function SetCondition(props: {
  earlier: ChapterJson | null;
  content: ChapterJson;
  variables: { patientAge?: number; patientSex?: string; viewer: Viewer };
}) {
  const { earlier, content, variables } = props;
  const expr = chapterConditionOf(content);
  if (!expr) return null;
  const described = describeLogic(earlier ?? {}, expr);
  const decided = testsOnlyThePatient(expr) ? isChapterShown(content, variables) : undefined;
  return (
    <Alert color={decided === false ? "red" : "orange"} icon={<IconGitBranch size={16} />} mb="md" p="sm">
      <Text size="sm" component="div">
        {described && <DisplaysWhen condition={described} label="This Question Set is shown only when" />}
      </Text>
      <Text size="xs" c="dimmed" mt={4}>
        {decided === undefined
          ? "It depends on answers in earlier Question Sets, which this preview doesn't have. The questions below are shown as if it were met."
          : decided
            ? "The sample patient meets it, so they would see this Question Set."
            : "The sample patient doesn't meet it, so they wouldn't see this Question Set. The questions below are shown anyway."}
      </Text>
    </Alert>
  );
}
