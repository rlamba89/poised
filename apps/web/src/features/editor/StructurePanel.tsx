"use client";
// The left column: Question Sets (chapters) as accordions, each with its pages (STR-01, FRM-07).
import { useEffect, useState } from "react";
import { ActionIcon, Button, Loader, Menu, Text, Tooltip } from "@mantine/core";
import {
  IconArrowBackUp, IconArrowForwardUp, IconArrowDown, IconArrowUp, IconChevronDown, IconChevronRight, IconCopy, IconDots, IconFile,
  IconGitBranch, IconPlus, IconTrash,
} from "@tabler/icons-react";
import { chapterConditionOf, pageTitle, pagesOf } from "@poised/clinical";
import type { Chapter } from "@/features/chapters/types";
import type { Chapters } from "./useChapters";
import css from "./editor.module.css";

export type Selection = { chapterId: string; page?: string } | null;

export type StructureActions = {
  selectSet: (chapterId: string) => void;
  selectPage: (chapterId: string, page: string) => void;
  moveSet: (chapterId: string, delta: number) => void;
  deleteSet: (chapter: Chapter) => void;
  addSet: () => void;
  addPage: (chapterId: string) => void;
  movePage: (chapterId: string, page: string, delta: number) => void;
  movePageTo: (chapterId: string, page: string, beforePage: string) => void;
  copyPage: (chapterId: string, page: string) => void;
  deletePage: (chapterId: string, page: string) => void;
  pageLogic: (chapterId: string, page: string) => void;
};

export function StructurePanel(props: {
  chapters: Chapter[];
  store: Chapters;
  selection: Selection;
  readOnly: boolean;
  actions: StructureActions;
}) {
  const { chapters, store, selection, readOnly, actions } = props;
  const [open, setOpen] = useState<Set<string>>(() => new Set(selection ? [selection.chapterId] : []));
  const [dragPage, setDragPage] = useState<{ chapterId: string; page: string } | null>(null);
  const [dropOn, setDropOn] = useState<string | null>(null);
  const current = selection ? store.get(selection.chapterId) : undefined;
  // Whatever is selected (a new Question Set, a page from the URL) is shown open.
  const selectedSet = selection?.chapterId;
  useEffect(() => {
    if (selectedSet) setOpen((prev) => (prev.has(selectedSet) ? prev : new Set(prev).add(selectedSet)));
  }, [selectedSet]);

  const toggle = (id: string) => {
    const next = new Set(open);
    if (next.has(id) && selection?.chapterId === id && !selection.page) next.delete(id);
    else next.add(id);
    setOpen(next);
    store.load(id);
    actions.selectSet(id);
  };

  return (
    <>
      <div className={css.sideTitle}>
        <span>Structure</span>
        {!readOnly && current && (current.undo.length > 0 || current.redo.length > 0) && (
          <span style={{ display: "flex", gap: 2 }}>
            <Tooltip label="Undo">
              <ActionIcon variant="subtle" color="gray" aria-label="Undo" disabled={!current.undo.length} onClick={() => store.undo(selection!.chapterId)}>
                <IconArrowBackUp size={18} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label="Redo">
              <ActionIcon variant="subtle" color="gray" aria-label="Redo" disabled={!current.redo.length} onClick={() => store.redo(selection!.chapterId)}>
                <IconArrowForwardUp size={18} />
              </ActionIcon>
            </Tooltip>
          </span>
        )}
      </div>
      <div style={{ padding: "8px 0" }}>
        {chapters.map((c, i) => {
          const isOpen = open.has(c.id);
          const entry = store.get(c.id);
          const pages = entry ? pagesOf(entry.doc) : [];
          return (
            <div key={c.id}>
              <div
                className={`${css.setRow} ${selection?.chapterId === c.id && !selection.page ? css.setRowActive : ""}`}
                onClick={() => toggle(c.id)}
                data-testid={`set-${c.name}`}
              >
                {isOpen ? <IconChevronDown size={16} /> : <IconChevronRight size={16} />}
                <span className={css.rowName}>{c.name}</span>
                {entry && chapterConditionOf(entry.doc) && (
                  <Tooltip label="Shown only when a condition is met">
                    <IconGitBranch size={14} style={{ color: "var(--mantine-color-orange-6)" }} />
                  </Tooltip>
                )}
                {!readOnly && (
                  <RowMenu
                    label={`${c.name} options`}
                    items={[
                      { label: "Move up", icon: IconArrowUp, disabled: i === 0, onClick: () => actions.moveSet(c.id, -1) },
                      { label: "Move down", icon: IconArrowDown, disabled: i === chapters.length - 1, onClick: () => actions.moveSet(c.id, 1) },
                      { label: "Delete", icon: IconTrash, danger: true, onClick: () => actions.deleteSet(c) },
                    ]}
                  />
                )}
              </div>
              {isOpen && !entry && <Loader size="xs" ml={52} my={6} />}
              {isOpen &&
                pages.map((p, pi) => {
                  const active = selection?.chapterId === c.id && selection.page === p.name;
                  return (
                    <div
                      key={p.name}
                      className={`${css.pageRow} ${active ? css.pageRowActive : ""} ${dropOn === p.name ? css.dropBefore : ""}`}
                      onClick={() => actions.selectPage(c.id, p.name)}
                      draggable={!readOnly}
                      onDragStart={() => setDragPage({ chapterId: c.id, page: p.name })}
                      onDragEnd={() => {
                        setDragPage(null);
                        setDropOn(null);
                      }}
                      onDragOver={(e) => {
                        if (dragPage?.chapterId === c.id && dragPage.page !== p.name) {
                          e.preventDefault();
                          setDropOn(p.name);
                        }
                      }}
                      onDragLeave={() => setDropOn(null)}
                      onDrop={() => {
                        if (dragPage) actions.movePageTo(c.id, dragPage.page, p.name);
                        setDragPage(null);
                        setDropOn(null);
                      }}
                      data-testid={`page-${pageTitle(p, pi)}`}
                    >
                      <IconFile size={18} style={{ flex: "none" }} />
                      <span className={css.rowName}>{pageTitle(p, pi)}</span>
                      {p.visibleIf && <IconGitBranch size={14} style={{ color: "var(--mantine-color-orange-6)" }} />}
                      {!readOnly && (
                        <RowMenu
                          label={`${pageTitle(p, pi)} options`}
                          items={[
                            { label: "Move up", icon: IconArrowUp, disabled: pi === 0, onClick: () => actions.movePage(c.id, p.name, -1) },
                            { label: "Move down", icon: IconArrowDown, disabled: pi === pages.length - 1, onClick: () => actions.movePage(c.id, p.name, 1) },
                            { label: "Logic", icon: IconGitBranch, onClick: () => actions.pageLogic(c.id, p.name) },
                            { label: "Copy", icon: IconCopy, onClick: () => actions.copyPage(c.id, p.name) },
                            { label: "Delete", icon: IconTrash, danger: true, onClick: () => actions.deletePage(c.id, p.name) },
                          ]}
                        />
                      )}
                    </div>
                  );
                })}
              {isOpen && entry && !readOnly && (
                <Button variant="default" size="xs" leftSection={<IconPlus size={14} />} ml={44} my={8} onClick={() => actions.addPage(c.id)}>
                  Add Page
                </Button>
              )}
            </div>
          );
        })}
        {chapters.length === 0 && <Text c="dimmed" size="sm" px="lg" py="sm">No Question Sets yet.</Text>}
        {!readOnly && (
          <Button variant="default" size="xs" leftSection={<IconPlus size={14} />} ml="lg" mt="sm" onClick={actions.addSet}>
            Add Question Set
          </Button>
        )}
      </div>
    </>
  );
}

type MenuItem = { label: string; icon: typeof IconTrash; onClick: () => void; disabled?: boolean; danger?: boolean };

function RowMenu({ label, items }: { label: string; items: MenuItem[] }) {
  return (
    <Menu position="bottom-end" withinPortal>
      <Menu.Target>
        <ActionIcon variant="subtle" color="gray" size="sm" aria-label={label} style={{ color: "inherit" }} onClick={(e) => e.stopPropagation()}>
          <IconDots size={16} />
        </ActionIcon>
      </Menu.Target>
      <Menu.Dropdown onClick={(e) => e.stopPropagation()}>
        {items.map((it) => (
          <Menu.Item key={it.label} leftSection={<it.icon size={14} />} disabled={it.disabled} color={it.danger ? "red" : undefined} onClick={it.onClick}>
            {it.label}
          </Menu.Item>
        ))}
      </Menu.Dropdown>
    </Menu>
  );
}
