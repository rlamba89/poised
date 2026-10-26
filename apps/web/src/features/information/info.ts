// The patient's Information for an episode (plan C5, session S36; INF-04): the types the Go API
// returns (apps/api/internal/httpapi/information.go) and the small rules that don't need React,
// unit-tested in info.test.ts. Go decides what's in each list and what the patient may see.

export type InfoKind = "video" | "leaflet";

/** An item in a procedure's list (GET /api/o/{trust}/procedures, infoItems): a retired one isn't shown to patients. */
export type ProcedureInfoItem = { id: string; kind: InfoKind; title: string; status: "active" | "retired" };

/** An item in an episode's list, as staff see it (GET …/episodes/{id}/information). */
export type EpisodeInfoItem = {
  id: string;
  kind: InfoKind;
  title: string;
  url: string;
  lengthSeconds: number | null;
  status: "active" | "retired";
  /** Copied from the procedure when the episode was made, or added for this episode. */
  fromProcedure: boolean;
  addedByName: string | null;
};

/** A library item that can be added (active, Preopsis's or the trust's own). */
export type LibraryItem = { id: string; kind: InfoKind; title: string; lengthSeconds?: number | null; tags?: string[] };

/** The staff route's answer: the episode's list and what can be added to it. */
export type EpisodeInfo = { items: EpisodeInfoItem[]; library: LibraryItem[] };

/** The patient's route's answer (GET /api/p/episodes/{id}/information): active items only. */
export type PatientInfo = {
  hospital: string;
  phone: string;
  email: string;
  items: { id: string; kind: InfoKind; title: string; url: string; lengthSeconds: number | null }[];
};

export const kindLabel = (kind: InfoKind) => (kind === "video" ? "Video" : "Leaflet");

const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? "" : "s"}`;

/** "2 videos, 1 leaflet", or "" for none. */
export function infoSummary(items: { kind: InfoKind }[]): string {
  const videos = items.filter((i) => i.kind === "video").length;
  const leaflets = items.length - videos;
  return [videos ? plural(videos, "video") : "", leaflets ? plural(leaflets, "leaflet") : ""].filter(Boolean).join(", ");
}

/** The library's items not already in the list (by id), by title. */
export function addable<T extends { id: string; title: string }>(library: T[], inList: string[]): T[] {
  return library.filter((i) => !inList.includes(i.id)).sort((a, b) => a.title.localeCompare(b.title));
}

/** A tel: link for the hospital's number (digits and a leading +), or "" if it has no digits. */
export function telHref(phone: string): string {
  const digits = phone.replace(/[^\d+]/g, "").replace(/(?!^)\+/g, "");
  return /\d/.test(digits) ? `tel:${digits}` : "";
}

/** A mailto: link for the hospital's address, or "" unless it's a plain address. */
export function mailtoHref(email: string): string {
  return /^[^\s@?&#/:]+@[^\s@?&#/:]+\.[^\s@?&#/:]+$/.test(email) ? `mailto:${email}` : "";
}
