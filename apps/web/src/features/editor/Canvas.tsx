"use client";
// The right-hand column: a Question Set's settings, or a page's cards.
import { Group, Paper, Select, Switch, Text, Textarea } from "@mantine/core";
import { chapterConditionOf, pagesOf as pagesIn, setChapterCondition, type ChapterJson } from "@sj/clinical";
import { Toggle } from "./LogicEditor";
import { IconFile, IconGitBranch } from "@tabler/icons-react";
import { describeLogic, pageTitle, pagesOf, updatePage, type PageJson } from "@sj/clinical";
import { CHAPTER_ICONS, chapterIcon } from "@/features/chapters/icons";
import { AUDIENCE_LABELS, type Audience, type Chapter } from "@/features/chapters/types";
import { AddContentBar } from "./AddContentBar";
import { useEditor } from "./context";
import { EditableText } from "./EditableText";
import { DisplaysWhen, ElementList } from "./ElementCard";
import css from "./editor.module.css";

type ChapterPatch = Partial<Pick<Chapter, "name" | "description" | "icon" | "audience">>;

export function SetCanvas(props: {
  chapter: Chapter;
  doc: ChapterJson | undefined;
  /** The earlier Question Sets combined (combineChapters), once they have loaded. */
  earlier: ChapterJson | undefined;
  readOnly: boolean;
  change: (edit: (doc: ChapterJson) => ChapterJson) => void;
  onPatch: (p: ChapterPatch) => void;
}) {
  const { chapter, doc, earlier, readOnly, change, onPatch } = props;
  const lastEarlierPage = earlier ? pagesIn(earlier).at(-1)?.name ?? "" : "";
  const Icon = chapterIcon(chapter.icon);
  return (
    <>
      <Group gap={6}>
        <Icon size={16} />
        <Text size="sm">Question Set</Text>
      </Group>
      <div style={{ margin: "8px 0 16px" }}>
        <EditableText label="Question Set name" value={chapter.name} readOnly={readOnly} onSave={(name) => onPatch({ name })} size="lg" />
      </div>
      <Paper withBorder p="lg" radius="sm">
        <Group align="flex-start" wrap="nowrap" gap="lg">
          <div style={{ width: 56, height: 56, borderRadius: "50%", background: "var(--mantine-color-gray-1)", display: "grid", placeItems: "center", flex: "none" }}>
            <Icon size={26} />
          </div>
          <div style={{ flex: 1, display: "flex", flexDirection: "column", gap: 14 }}>
            <Select
              label="Icon Name"
              description="Shown on the patient's chapter card"
              data={Object.keys(CHAPTER_ICONS)}
              value={chapter.icon || null}
              onChange={(v) => v && onPatch({ icon: v })}
              searchable
              disabled={readOnly}
              maw={260}
              renderOption={({ option }) => {
                const I = chapterIcon(option.value);
                return (
                  <Group gap="xs">
                    <I size={16} />
                    {option.label}
                  </Group>
                );
              }}
            />
            <Textarea
              label="Description"
              defaultValue={chapter.description}
              key={chapter.id + chapter.description}
              onBlur={(e) => e.currentTarget.value !== chapter.description && onPatch({ description: e.currentTarget.value })}
              disabled={readOnly}
              autosize
              minRows={2}
            />
            <Select
              label="Who is it for?"
              data={Object.entries(AUDIENCE_LABELS).map(([value, label]) => ({ value, label }))}
              value={chapter.audience}
              onChange={(v) => v && onPatch({ audience: v as Audience })}
              allowDeselect={false}
              disabled={readOnly}
              maw={260}
            />
            {doc && (
              <Select
                label="Patients see"
                description="Sections are always their own screens. One question per screen suits phones and longer answers (STR-05)."
                data={[
                  { value: "standard", label: "A page at a time" },
                  { value: "questionPerPage", label: "One question per screen" },
                ]}
                value={doc.questionsOnPageMode === "questionPerPage" ? "questionPerPage" : "standard"}
                onChange={(v) => change((d) => ({ ...d, questionsOnPageMode: v === "questionPerPage" ? "questionPerPage" : undefined }))}
                allowDeselect={false}
                disabled={readOnly}
                maw={260}
              />
            )}
            {doc && (
              <Switch
                label="Show a progress bar to patients"
                checked={doc.showProgressBar === true || doc.showProgressBar === "top"}
                disabled={readOnly}
                onChange={(e) => {
                  const on = e.currentTarget.checked;
                  change((d) => ({ ...d, showProgressBar: on ? true : undefined }));
                }}
              />
            )}
          </div>
        </Group>
      </Paper>
      {doc && (
        <Paper withBorder p="lg" radius="sm" mt="md">
          {earlier ? (
            <Toggle
              label="Display this Question Set"
              off="Always"
              on="Conditionally"
              doc={earlier}
              scope={{ upToPage: lastEarlierPage }}
              value={chapterConditionOf(doc)}
              readOnly={readOnly}
              onChange={(expr) => change((d) => setChapterCondition(d, expr))}
              emptyHint="This is the first Question Set, so only the patient's age and sex and who is viewing can be tested."
            />
          ) : (
            <Text size="sm" c="dimmed">Loading the earlier Question Sets…</Text>
          )}
          <Text size="xs" c="dimmed" mt="sm">
            A hidden Question Set isn&apos;t shown to the patient, and its answers produce no clinical outputs (LOG-09).
          </Text>
        </Paper>
      )}
    </>
  );
}

export function PageCanvas({ page, onOpenPageLogic }: { page: PageJson; onOpenPageLogic: () => void }) {
  const ed = useEditor();
  const index = pagesOf(ed.doc).indexOf(page);
  const elements = page.elements ?? [];
  const condition = describeLogic(ed.doc, page.visibleIf);
  return (
    <div onClick={() => ed.select(null)} style={{ minHeight: "100%" }}>
      <div className={css.canvasHeader}>
        <Group gap={6}>
          <IconFile size={16} />
          <Text size="sm">Clinical Page</Text>
        </Group>
        <Switch
          label="Clinical summary"
          labelPosition="left"
          checked={!!page.clinicalSummary}
          disabled={ed.readOnly}
          onClick={(e) => e.stopPropagation()}
          onChange={(e) => {
            const on = e.currentTarget.checked;
            ed.change((d) => updatePage(d, page.name, { clinicalSummary: on || undefined }));
          }}
        />
      </div>
      <div onClick={(e) => e.stopPropagation()}>
        <EditableText
          label="Page name"
          value={pageTitle(page, index)}
          readOnly={ed.readOnly}
          onSave={(title) => ed.change((d) => updatePage(d, page.name, { title }))}
        />
      </div>
      {condition && (
        <div
          className={`${css.displaysWhen} ${css.pageLogic}`}
          onClick={(e) => {
            e.stopPropagation();
            onOpenPageLogic();
          }}
        >
          <IconGitBranch size={15} className={css.branch} />
          <DisplaysWhen condition={condition} />
        </div>
      )}
      <div className={css.cards}>
        <ElementList elements={elements} target={{ page: page.name }} pageElements={elements} />
        {elements.length === 0 && <Text c="dimmed" ta="center">This page is empty. Add content below.</Text>}
      </div>
      <AddContentBar target={{ page: page.name }} pageElements={elements} />
    </div>
  );
}
