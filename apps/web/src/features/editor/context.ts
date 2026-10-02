"use client";
// What every part of the page canvas needs: the chapter being edited and the editor's actions.
import { createContext, useContext } from "react";
import type { ChapterJson, Kind, Target } from "@poised/clinical";

export type EditorContextValue = {
  doc: ChapterJson;
  pageName: string;
  readOnly: boolean;
  selected: string | null;
  /** The group or section new elements go into (Lifebox: a selected Section receives Add content). */
  select: (name: string | null) => void;
  change: (edit: (doc: ChapterJson) => ChapterJson) => void;
  add: (kind: Kind, target: Target) => void;
  remove: (name: string) => void;
  openDisclosures: (name: string, option?: string) => void;
  openLogic: (name: string) => void;
  dragging: string | null;
  setDragging: (name: string | null) => void;
};

export const EditorContext = createContext<EditorContextValue | null>(null);

export function useEditor(): EditorContextValue {
  const ctx = useContext(EditorContext);
  if (!ctx) throw new Error("useEditor outside the editor");
  return ctx;
}
