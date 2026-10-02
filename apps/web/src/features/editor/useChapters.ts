"use client";
// Each Question Set's (chapter's) SurveyJS JSON, loaded on demand, with autosave (LCY-03),
// undo and redo (LCY-05) and the stale-revision guard (LCY-04).
import { useCallback, useEffect, useRef, useState } from "react";
import type { ChapterJson } from "@poised/clinical";
import { api, ApiError } from "@/lib/api";

export type SaveStatus = "saved" | "modified" | "saving" | "error" | "conflict";

type Entry = {
  doc: ChapterJson;
  revision: number;
  undo: ChapterJson[];
  /** Versions undone, for redo; cleared by the next edit. */
  redo: ChapterJson[];
  status: SaveStatus;
  error?: string;
  /** Saves for one chapter run one at a time; each sends the latest JSON and revision. */
  queue: Promise<void>;
  timer?: ReturnType<typeof setTimeout>;
};

const SAVE_DELAY_MS = 600;
const UNDO_LIMIT = 100;

export function useChapters(hospitalId: string) {
  const entries = useRef(new Map<string, Entry>());
  const loading = useRef(new Set<string>());
  const [, setTick] = useState(0);
  const rerender = useCallback(() => setTick((n) => n + 1), []);
  const [loadError, setLoadError] = useState("");

  const load = useCallback(
    (id: string) => {
      if (entries.current.has(id) || loading.current.has(id)) return;
      loading.current.add(id);
      api<{ content: ChapterJson; revision: number }>(`/h/${hospitalId}/chapters/${id}`)
        .then((c) => {
          entries.current.set(id, { doc: c.content ?? {}, revision: c.revision, undo: [], redo: [], status: "saved", queue: Promise.resolve() });
          rerender();
        })
        .catch((e: Error) => setLoadError(e.message))
        .finally(() => loading.current.delete(id));
    },
    [hospitalId, rerender],
  );

  const save = useCallback(
    (id: string) => {
      const e = entries.current.get(id);
      if (!e) return;
      e.queue = e.queue.then(async () => {
        if (e.status === "conflict") return;
        const doc = e.doc;
        e.status = "saving";
        rerender();
        try {
          const res = await api<{ revision: number }>(`/h/${hospitalId}/chapters/${id}/content`, {
            method: "PUT",
            body: JSON.stringify({ content: doc, revision: e.revision }),
          });
          e.revision = res.revision;
          // Another edit may have arrived while saving; its own save is already queued.
          if (e.doc === doc) e.status = "saved";
        } catch (err) {
          e.error = (err as Error).message;
          e.status = err instanceof ApiError && err.status === 409 ? "conflict" : "error";
        }
        rerender();
      });
    },
    [hospitalId, rerender],
  );

  /** Replaces a chapter's JSON with an edited copy, records undo and schedules a save. */
  const apply = useCallback(
    (id: string, change: (doc: ChapterJson) => ChapterJson) => {
      const e = entries.current.get(id);
      if (!e || e.status === "conflict") return;
      const next = change(e.doc);
      if (next === e.doc) return;
      e.undo = [...e.undo.slice(-UNDO_LIMIT + 1), e.doc];
      e.redo = [];
      e.doc = next;
      e.status = "modified";
      clearTimeout(e.timer);
      e.timer = setTimeout(() => save(id), SAVE_DELAY_MS);
      rerender();
    },
    [rerender, save],
  );

  const undo = useCallback(
    (id: string) => {
      const e = entries.current.get(id);
      const previous = e?.undo.at(-1);
      if (!e || !previous || e.status === "conflict") return;
      e.undo = e.undo.slice(0, -1);
      e.redo = [...e.redo, e.doc];
      e.doc = previous;
      e.status = "modified";
      clearTimeout(e.timer);
      e.timer = setTimeout(() => save(id), SAVE_DELAY_MS);
      rerender();
    },
    [rerender, save],
  );

  const redo = useCallback(
    (id: string) => {
      const e = entries.current.get(id);
      const next = e?.redo.at(-1);
      if (!e || !next || e.status === "conflict") return;
      e.redo = e.redo.slice(0, -1);
      e.undo = [...e.undo, e.doc];
      e.doc = next;
      e.status = "modified";
      clearTimeout(e.timer);
      e.timer = setTimeout(() => save(id), SAVE_DELAY_MS);
      rerender();
    },
    [rerender, save],
  );

  const retry = useCallback((id: string) => save(id), [save]);

  /** Forgets a chapter (after it is deleted). */
  const drop = useCallback((id: string) => {
    clearTimeout(entries.current.get(id)?.timer);
    entries.current.delete(id);
  }, []);

  // Warn before leaving with unsaved changes.
  const unsaved = [...entries.current.values()].some((e) => e.status === "modified" || e.status === "saving" || e.status === "error");
  useEffect(() => {
    if (!unsaved) return;
    const warn = (ev: BeforeUnloadEvent) => ev.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [unsaved]);

  return {
    get: (id: string) => entries.current.get(id),
    load,
    apply,
    undo,
    redo,
    retry,
    drop,
    loadError,
    /** The worst status across chapters, for the header. */
    overall: worstStatus([...entries.current.values()].map((e) => e.status)),
    firstError: [...entries.current.entries()].find(([, e]) => e.status === "error" || e.status === "conflict"),
  };
}

export type Chapters = ReturnType<typeof useChapters>;

const ORDER: SaveStatus[] = ["saved", "modified", "saving", "error", "conflict"];
function worstStatus(list: SaveStatus[]): SaveStatus {
  return list.reduce<SaveStatus>((a, b) => (ORDER.indexOf(b) > ORDER.indexOf(a) ? b : a), "saved");
}
