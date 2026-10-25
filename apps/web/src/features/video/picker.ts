// The editor's "Choose from library" for a Video element (S34): the content library's search
// (GET /api/o/<trust>/content/search?q=&tag=, S33), which Go answers with the trust's and Preopsis's
// active items. These helpers only shape what it sends for the window.

/** One item as the search returns it (Go's searchContent). */
export type Found = {
  id: string;
  platform: boolean;
  kind: "video" | "leaflet";
  title: string;
  url: string;
  lengthSeconds: number | null;
  tags: string[];
};

/** The videos in a search's answer (it holds leaflets too). */
export const videosOf = (list: Found[]): Found[] => list.filter((i) => i.kind === "video");

/** The videos' tags, each once in any case, sorted: the picker's tag buttons. */
export function tagsOf(list: Found[]): string[] {
  const out: string[] = [];
  for (const t of videosOf(list).flatMap((i) => i.tags)) {
    if (!out.some((o) => o.toLowerCase() === t.toLowerCase())) out.push(t);
  }
  return out.sort((a, b) => a.localeCompare(b));
}

/** The search's path: words in the title or a tag (q), and a whole tag. */
export function searchPath(trustPath: string, q: string, tag: string | null): string {
  const params: string[] = [];
  if (q.trim()) params.push(`q=${encodeURIComponent(q.trim())}`);
  if (tag) params.push(`tag=${encodeURIComponent(tag)}`);
  return `${trustPath}/content/search${params.length ? `?${params.join("&")}` : ""}`;
}
