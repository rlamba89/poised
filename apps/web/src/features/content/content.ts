// The content library's small rules (plan C5, S33; INF-05, PX-13). Go checks every link, length and
// tag; these only shape the form and show a video's length.
import { formatLength, parseLength } from "@poised/clinical";

export type ContentKind = "video" | "leaflet";

/** One item as the library pages list it (Go's contentItem). platform: Preopsis's own. */
export type ContentItem = {
  id: string;
  platform: boolean;
  kind: ContentKind;
  title: string;
  url: string;
  lengthSeconds: number | null;
  tags: string[];
  status: "active" | "retired";
  updatedAt: string;
  updatedByName: string;
};

/** A PX-13 checklist topic, and whether an active item has its tag. */
export type Topic = { topic: string; covered: boolean };

export type ContentList = { items: ContentItem[]; checklist: Topic[] };

/** A platform starter question in the platform admin's list (S33 answers 4). */
export type StarterQuestion = { id: string; key: string; title: string; tags: string[]; retired: boolean };

// A video's length is read and shown by the same helpers as the Video element's (S34).
export { formatLength, parseLength };

/** tags with `tag` added, unless it's there already in any case. */
export function addTag(tags: string[], tag: string): string[] {
  return tags.some((t) => t.toLowerCase() === tag.toLowerCase()) ? tags : [...tags, tag];
}

/** The Tags box's tags, trimmed, without blank ones or repeats in any case (spaces left on blur made an empty one, test report 2). */
export function cleanTags(tags: string[]): string[] {
  return tags.map((t) => t.trim()).filter(Boolean).reduce<string[]>(addTag, []);
}

/** The add and edit form's fields. length is as typed. */
export type ContentForm = { kind: ContentKind; title: string; url: string; length: string; tags: string[] };

export const emptyForm = (kind: ContentKind): ContentForm => ({ kind, title: "", url: "", length: "", tags: [] });

export const formOf = (c: ContentItem): ContentForm => ({
  kind: c.kind, title: c.title, url: c.url, length: formatLength(c.lengthSeconds), tags: c.tags,
});

export type ContentBody = { kind: ContentKind; title: string; url: string; lengthSeconds?: number | null; tags: string[] };

/** The POST body for a form. A video's length goes in seconds; one that can't be read as null, for Go's message. */
export function bodyOf(f: ContentForm): ContentBody {
  const body: ContentBody = { kind: f.kind, title: f.title, url: f.url, tags: f.tags };
  if (f.kind === "video") body.lengthSeconds = parseLength(f.length);
  return body;
}

/** What an edit sends: the fields that changed, never the kind (it can't change). */
export function changedFields(was: ContentBody, next: ContentBody): Partial<ContentBody> {
  const out: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(next)) {
    if (k !== "kind" && JSON.stringify(v) !== JSON.stringify(was[k as keyof ContentBody])) out[k] = v;
  }
  return out as Partial<ContentBody>;
}
