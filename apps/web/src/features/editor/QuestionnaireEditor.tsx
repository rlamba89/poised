"use client";
// The questionnaire editor, laid out like the Lifebox Author tool (docs/lifebox-ui-notes.md):
// Structure tree on the left (replaced by the Settings panel while a card is selected), the
// selected Question Set or page on the right. Every change saves automatically.
import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Alert, Anchor, Avatar, Badge, Button, Center, Group, Loader, Menu, Modal, Popover, Stack, Text } from "@mantine/core";
import { IconAlertTriangle, IconArrowLeft, IconCopyPlus, IconEye, IconLanguage, IconLogout, IconSend } from "@tabler/icons-react";
import {
  addElement, addPage, chapterConditionProblems, combineChapters, copyPage, deleteElement, deletePage, dependentsOf, elementLabel,
  findElement, findPage, logicProblems, movePage, movePageTo, pageTitle, pagesOf, type ChapterJson, type Kind, type Target,
} from "@sj/clinical";
import { api } from "@/lib/api";
import { hasRole, useMe } from "@/lib/auth";
import type { Chapter } from "@/features/chapters/types";
import { PageCanvas, SetCanvas } from "./Canvas";
import { EditorContext, type EditorContextValue } from "./context";
import { DisclosureModal, type DisclosureTarget } from "./DisclosureModal";
import { PageLogic } from "./LogicEditor";
import { PublishModal } from "./PublishModal";
import { SettingsPanel, type PanelTab } from "./SettingsPanel";
import { StructurePanel, type Selection } from "./StructurePanel";
import { TranslateView } from "./TranslateView";
import { useChapters, type SaveStatus } from "./useChapters";
import css from "./editor.module.css";

type Detail = {
  questionnaire: { id: string; name: string; description: string; versionId: string; versionNo: number; status: string };
  chapters: Chapter[];
};

const STATUS: Record<SaveStatus, { text: string; color: string }> = {
  saved: { text: "All changes saved", color: "gray" },
  modified: { text: "Saving…", color: "blue" },
  saving: { text: "Saving…", color: "blue" },
  error: { text: "Not saved", color: "red" },
  conflict: { text: "Not saved", color: "red" },
};

type Confirm = { title: string; body: string; dependents?: string[]; onConfirm: () => void };

export function QuestionnaireEditor({ hospitalId, questionnaireId }: { hospitalId: string; questionnaireId: string }) {
  const me = useMe();
  const router = useRouter();
  const base = `/h/${hospitalId}`;
  const [data, setData] = useState<Detail | null>(null);
  const [error, setError] = useState("");
  const store = useChapters(hospitalId);
  const readOnly = !hasRole(me, hospitalId, "author") || (data ? data.questionnaire.status !== "draft" : true);

  const [selection, setSelection] = useState<Selection>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [tab, setTab] = useState<PanelTab>("settings");
  const [pageLogic, setPageLogic] = useState<{ chapterId: string; page: string } | null>(null);
  const [disclosure, setDisclosure] = useState<DisclosureTarget | null>(null);
  const [dragging, setDragging] = useState<string | null>(null);
  const [confirm, setConfirm] = useState<Confirm | null>(null);
  const [translating, setTranslating] = useState(false);
  const [publishing, setPublishing] = useState(false);

  const reload = useCallback(
    () =>
      api<Detail>(`${base}/questionnaires/${questionnaireId}`)
        .then(setData)
        .catch((e: Error) => setError(e.message)),
    [base, questionnaireId],
  );
  useEffect(() => {
    reload();
  }, [reload]);

  // Every Question Set is loaded: Question Set conditions test questions in earlier sets (LOG-02).
  useEffect(() => {
    data?.chapters.forEach((c) => store.load(c.id));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data]);

  // Keep the selection in the URL, so a reload comes back to the same place.
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const set = params.get("set");
    if (set) {
      setSelection({ chapterId: set, page: params.get("page") ?? undefined });
      store.load(set);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  useEffect(() => {
    const params = new URLSearchParams();
    if (selection) params.set("set", selection.chapterId);
    if (selection?.page) params.set("page", selection.page);
    const qs = params.toString();
    window.history.replaceState(null, "", qs ? `?${qs}` : window.location.pathname);
  }, [selection]);

  const run = async (fn: () => Promise<unknown>) => {
    setError("");
    try {
      await fn();
    } catch (e) {
      setError((e as Error).message);
    }
    await reload();
  };

  const chapterId = selection?.chapterId;
  const entry = chapterId ? store.get(chapterId) : undefined;
  const doc = entry?.doc;
  const page = doc && selection?.page ? findPage(doc, selection.page) : undefined;
  const chapter = data?.chapters.find((c) => c.id === chapterId);

  const change = useCallback(
    (edit: (d: ChapterJson) => ChapterJson) => {
      if (chapterId && !readOnly) store.apply(chapterId, edit);
    },
    [chapterId, readOnly, store],
  );

  // Ctrl/⌘+Z undoes and Ctrl/⌘+Shift+Z (or Ctrl+Y) redoes, except while typing in a box.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!chapterId || readOnly || !(e.metaKey || e.ctrlKey)) return;
      const t = e.target as HTMLElement;
      if (t.closest("input, textarea, [contenteditable=true]")) return;
      const key = e.key.toLowerCase();
      if (key === "z" && !e.shiftKey) store.undo(chapterId);
      else if ((key === "z" && e.shiftKey) || key === "y") store.redo(chapterId);
      else return;
      e.preventDefault();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [chapterId, readOnly, store]);

  const selectElement = (name: string | null) => {
    setSelected(name);
    setPageLogic(null);
    if (name) setTab("settings");
  };

  const ctx: EditorContextValue | null = useMemo(() => {
    if (!doc || !page) return null;
    return {
      doc,
      pageName: page.name,
      readOnly,
      selected,
      select: selectElement,
      change,
      add: (kind: Kind, target: Target) => {
        let added = "";
        change((d) => {
          const r = addElement(d, target, kind);
          added = r.name;
          return r.doc;
        });
        if (added && kind !== "bmi" && kind !== "profile") selectElement(added);
      },
      remove: (name: string) => {
        const el = findElement(doc, name)?.el;
        const label = el ? elementLabel(el) : name;
        setConfirm({
          title: "Delete this?",
          body: `“${label}” will be deleted, with its options and disclosures.`,
          dependents: dependentsOf(doc, name),
          onConfirm: () => {
            change((d) => deleteElement(d, name));
            if (selected === name) setSelected(null);
          },
        });
      },
      openDisclosures: (name: string, option?: string) => setDisclosure({ element: name, option }),
      openLogic: (name: string) => {
        setSelected(name);
        setPageLogic(null);
        setTab("logic");
      },
      dragging,
      setDragging,
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [doc, page, readOnly, selected, change, dragging]);

  const withChapter = (id: string, edit: (d: ChapterJson) => ChapterJson) => !readOnly && store.apply(id, edit);

  const actions = {
    selectSet: (id: string) => {
      setTranslating(false);
      setSelection({ chapterId: id });
      setSelected(null);
      setPageLogic(null);
    },
    selectPage: (id: string, p: string) => {
      setTranslating(false);
      setSelection({ chapterId: id, page: p });
      setSelected(null);
      setPageLogic(null);
    },
    moveSet: (id: string, delta: number) => {
      if (!data) return;
      const ids = data.chapters.map((c) => c.id);
      const i = ids.indexOf(id);
      const j = i + delta;
      if (j < 0 || j >= ids.length) return;
      [ids[i], ids[j]] = [ids[j], ids[i]];
      setData({ ...data, chapters: ids.map((x) => data.chapters.find((c) => c.id === x)!) });
      run(() => api(`${base}/questionnaires/${questionnaireId}/chapter-order`, { method: "PUT", body: JSON.stringify(ids) }));
    },
    deleteSet: (c: Chapter) =>
      setConfirm({
        title: "Delete Question Set?",
        body: `“${c.name}” and all its pages and questions will be deleted. This can't be undone.`,
        onConfirm: () =>
          run(async () => {
            await api(`${base}/chapters/${c.id}`, { method: "DELETE" });
            store.drop(c.id);
            if (selection?.chapterId === c.id) setSelection(null);
          }),
      }),
    addSet: () =>
      run(async () => {
        const n = (data?.chapters.length ?? 0) + 1;
        const res = await api<{ id: string }>(`${base}/questionnaires/${questionnaireId}/chapters`, {
          method: "POST",
          body: JSON.stringify({ name: `Question Set ${n}` }),
        });
        setSelection({ chapterId: res.id });
        store.load(res.id);
      }),
    addPage: (id: string) => {
      let name = "";
      withChapter(id, (d) => {
        const r = addPage(d);
        name = r.name;
        return r.doc;
      });
      if (name) actions.selectPage(id, name);
    },
    movePage: (id: string, p: string, delta: number) => withChapter(id, (d) => movePage(d, p, delta)),
    movePageTo: (id: string, p: string, before: string) => withChapter(id, (d) => movePageTo(d, p, before)),
    copyPage: (id: string, p: string) => withChapter(id, (d) => copyPage(d, p).doc),
    deletePage: (id: string, p: string) => {
      const d = store.get(id)?.doc;
      const target = d && findPage(d, p);
      if (!d || !target) return;
      const names = (target.elements ?? []).flatMap(function walk(el): string[] {
        return [el.name, ...(el.elements ?? []).flatMap(walk)];
      });
      const outside = names.flatMap((n) => dependentsOf(d, n)).filter((x, i, all) => all.indexOf(x) === i);
      setConfirm({
        title: "Delete page?",
        body: `“${pageTitle(target, pagesOf(d).indexOf(target))}” and everything on it will be deleted.`,
        dependents: outside,
        onConfirm: () => {
          withChapter(id, (x) => deletePage(x, p));
          if (selection?.page === p) setSelection({ chapterId: id });
        },
      });
    },
    pageLogic: (id: string, p: string) => {
      setSelection({ chapterId: id, page: p });
      setSelected(null);
      setPageLogic({ chapterId: id, page: p });
    },
  };

  // A new version has new Question Set ids, so start again from the Structure tree.
  const createVersion = () =>
    run(async () => {
      await api(`${base}/questionnaires/${questionnaireId}/versions`, { method: "POST" });
      setSelection(null);
      setSelected(null);
    });

  if (!data) {
    return error ? <Alert color="red" m="md">{error}</Alert> : <Center h="60vh"><Loader /></Center>;
  }
  const q = data.questionnaire;
  const status = STATUS[store.overall];
  const showSettings = !!(ctx && selected && findElement(ctx.doc, selected));
  const logicPage = pageLogic && doc && pageLogic.chapterId === chapterId ? findPage(doc, pageLogic.page) : undefined;
  const earlierSets = chapter && data ? data.chapters.slice(0, data.chapters.indexOf(chapter)) : [];
  const earlier = earlierSets.every((c) => store.get(c.id))
    ? combineChapters(earlierSets.map((c) => ({ name: c.name, doc: store.get(c.id)!.doc })))
    : undefined;
  const problems = [
    ...(doc && earlier ? chapterConditionProblems(doc, earlier).map((message) => ({ label: "This Question Set", message, owner: "" })) : []),
    ...(doc ? logicProblems(doc) : []),
  ];

  return (
    <div className={css.root}>
      <header className={css.header}>
        <Group gap="xs" wrap="nowrap">
          <IconArrowLeft size={16} />
          <Anchor component={Link} href={`${base}/questionnaires`} underline="always" c="dark">Questionnaires</Anchor>
          <Text>/ {q.name}</Text>
          <Badge variant="light" color={q.status === "published" ? "green" : "gray"} size="sm">
            v{q.versionNo} · {q.status}
          </Badge>
          {readOnly && <Badge variant="outline" color="gray" size="sm">Read only</Badge>}
          {!readOnly && <Text size="sm" c={status.color} data-testid="save-status">{status.text}</Text>}
        </Group>
        <Group gap="sm" wrap="nowrap">
          {problems.length > 0 && (
            <Popover position="bottom-end" width={380} withArrow>
              <Popover.Target>
                <Button size="xs" variant="light" color="orange" leftSection={<IconAlertTriangle size={14} />} data-testid="logic-problems">
                  {problems.length} logic problem{problems.length === 1 ? "" : "s"}
                </Button>
              </Popover.Target>
              <Popover.Dropdown>
                <Text size="sm" fw={600} mb={4}>These conditions won&apos;t work as intended</Text>
                <Stack gap={4}>
                  {problems.map((p, i) => (
                    <Text
                      key={i}
                      size="sm"
                      style={{ cursor: p.owner && findElement(doc!, p.owner) ? "pointer" : undefined }}
                      onClick={() => {
                        if (p.label === "This Question Set" && chapterId) {
                          setTranslating(false);
                          setSelection({ chapterId });
                          return;
                        }
                        const loc = p.owner ? findElement(doc!, p.owner) : undefined;
                        if (!loc || !chapterId) return;
                        setTranslating(false);
                        setSelection({ chapterId, page: loc.page.name });
                        setSelected(p.owner);
                        setTab("logic");
                      }}
                    >
                      · {p.label}: {p.message}
                    </Text>
                  ))}
                </Stack>
              </Popover.Dropdown>
            </Popover>
          )}
          {chapterId && doc && (
            <Button
              size="xs"
              variant={translating ? "filled" : "default"}
              leftSection={<IconLanguage size={14} />}
              onClick={() => {
                setTranslating(!translating);
                setSelected(null);
                setPageLogic(null);
              }}
            >
              Translate
            </Button>
          )}
          {q.status === "draft" && hasRole(me, hospitalId, "publisher") && (
            <Button size="xs" leftSection={<IconSend size={14} />} onClick={() => setPublishing(true)}>
              Publish
            </Button>
          )}
          {q.status !== "draft" && hasRole(me, hospitalId, "author") && (
            <Button size="xs" variant="default" leftSection={<IconCopyPlus size={14} />} onClick={createVersion}>
              Create new version
            </Button>
          )}
          {chapterId && (
            <Button size="xs" variant="default" leftSection={<IconEye size={14} />} component={Link} href={`${base}/chapters/${chapterId}/preview`}>
              Preview
            </Button>
          )}
          <Menu position="bottom-end">
            <Menu.Target>
              <Avatar radius="xl" color="blue" style={{ cursor: "pointer" }} aria-label="Account">
                {me.user.name.split(" ").map((w) => w[0]).join("").slice(0, 2)}
              </Avatar>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Label>{me.user.name}</Menu.Label>
              <Menu.Item
                leftSection={<IconLogout size={14} />}
                onClick={async () => {
                  await api("/logout", { method: "POST" });
                  router.replace("/login");
                }}
              >
                Sign out
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>
        </Group>
      </header>

      {(error || store.loadError) && <Alert color="red" radius={0}>{error || store.loadError}</Alert>}
      {store.firstError && (
        <Alert color="red" radius={0} title={store.firstError[1].status === "conflict" ? "Someone else changed this Question Set" : "Your latest changes are not saved"}>
          {store.firstError[1].error}{" "}
          {store.firstError[1].status === "conflict" ? (
            <Button size="xs" variant="white" color="red" onClick={() => window.location.reload()}>Reload</Button>
          ) : (
            <Button size="xs" variant="white" color="red" onClick={() => store.retry(store.firstError![0])}>Try again</Button>
          )}
        </Alert>
      )}

      <div className={css.body}>
        <aside className={css.side}>
          {showSettings && ctx && (
            <SettingsPanel
              doc={ctx.doc}
              name={selected!}
              tab={tab}
              readOnly={readOnly}
              onTab={setTab}
              onClose={() => setSelected(null)}
              change={change}
              openDisclosures={setDisclosure}
            />
          )}
          {!showSettings && logicPage && doc && (
            <>
              <div className={css.sideTitle}>
                <span>Page logic · {pageTitle(logicPage, pagesOf(doc).indexOf(logicPage))}</span>
                <Button size="xs" variant="subtle" onClick={() => setPageLogic(null)}>Close</Button>
              </div>
              <div style={{ padding: 20 }}>
                <PageLogic key={logicPage.name} doc={doc} page={logicPage.name} readOnly={readOnly} change={change} />
              </div>
            </>
          )}
          {/* Hidden, not removed, while a panel is open, so open Question Sets stay open (Lifebox). */}
          <div style={{ display: showSettings || logicPage ? "none" : undefined }}>
            <StructurePanel chapters={data.chapters} store={store} selection={selection} readOnly={readOnly} actions={actions} />
          </div>
        </aside>

        <main className={css.canvas}>
          {translating && doc && chapter && (
            <TranslateView doc={doc} setName={chapter.name} readOnly={readOnly} change={change} onClose={() => setTranslating(false)} />
          )}
          {!translating && !selection && <Center h="50%"><Text c="dimmed">Choose a Question Set or a page on the left.</Text></Center>}
          {!translating && selection && !selection.page && chapter && (
            <SetCanvas
              chapter={chapter}
              doc={doc}
              earlier={earlier}
              readOnly={readOnly}
              change={change}
              onPatch={(patch) => run(() => api(`${base}/chapters/${chapter.id}`, { method: "PATCH", body: JSON.stringify(patch) }))}
            />
          )}
          {!translating && selection?.page && !doc && <Center h="50%"><Loader /></Center>}
          {!translating && ctx && page && (
            <EditorContext.Provider value={ctx}>
              <PageCanvas page={page} onOpenPageLogic={() => actions.pageLogic(chapterId!, page.name)} />
            </EditorContext.Provider>
          )}
        </main>
      </div>

      {doc && (
        <DisclosureModal doc={doc} target={disclosure} readOnly={readOnly} onChange={change} onClose={() => setDisclosure(null)} />
      )}

      <PublishModal
        opened={publishing}
        onClose={() => setPublishing(false)}
        onPublished={() => {
          setSelected(null);
          reload();
        }}
        path={`${base}/questionnaires/${questionnaireId}`}
        versionId={q.versionId}
        versionNo={q.versionNo}
        sets={data.chapters.every((c) => store.get(c.id)) ? data.chapters.map((c) => ({ name: c.name, doc: store.get(c.id)!.doc })) : null}
        unsaved={store.overall !== "saved"}
      />

      <Modal opened={!!confirm} onClose={() => setConfirm(null)} title={confirm?.title}>
        <Stack>
          <Text>{confirm?.body}</Text>
          {!!confirm?.dependents?.length && (
            <Alert color="orange" title="Other items depend on this">
              Their conditions will no longer work:
              <ul style={{ margin: "6px 0 0", paddingLeft: 18 }}>
                {confirm.dependents.map((d) => (
                  <li key={d}>{d}</li>
                ))}
              </ul>
            </Alert>
          )}
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setConfirm(null)}>Cancel</Button>
            <Button
              color="red"
              onClick={() => {
                confirm?.onConfirm();
                setConfirm(null);
              }}
            >
              Delete
            </Button>
          </Group>
        </Stack>
      </Modal>
    </div>
  );
}
