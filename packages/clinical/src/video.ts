// The Video element (plan C5, session S34; INF-01/02, A-6). A video lives on YouTube or Vimeo, never
// on our servers. The author pastes its link or picks it from the content library (S33), which
// copies the link, title and length into the questionnaire (S34 answers 1). The patient sees the
// player and, if the author wants, an "I have watched this video" tick, which can be required. The
// tick is the element's answer; it's never pre-filled from an earlier questionnaire (S34 answers 2).
//
// The link check is the same as Go's (apps/api/internal/content CheckVideo): both run the cases in
// apps/api/internal/content/testdata/video_links.json. Only a recognised video ID ever becomes a
// player's address: youtube-nocookie.com for YouTube, and Vimeo with dnt=1 (do not track).
import { ElementFactory, Question, Serializer, type SurveyError } from "survey-core";
import { allElements, edit, findElement, isVideo, textOf, VIDEO_TYPE, type ChapterJson, type ElementJson } from "./doc";

/** The tick's words until the author's translations say otherwise. */
export const DEFAULT_TICK_TEXT = "I have watched this video";

/** What a link plays: YouTube or Vimeo, the video's ID, and a private Vimeo link's hash. */
export type VideoRef = { provider: "youtube" | "vimeo"; id: string; hash?: string };

const MAX_URL = 2000;
const YOUTUBE_ID = /^[A-Za-z0-9_-]{11}$/;
const VIMEO_ID = /^[0-9]{1,12}$/;
const VIMEO_HASH = /^[0-9a-f]{6,20}$/;
const LINK = /^([A-Za-z][A-Za-z0-9+.-]*):\/\/([^/?#]*)([^?#]*)(?:\?([^#]*))?(?:#.*)?$/;

/** One query parameter's value ("" if absent or unreadable). */
function param(query: string | undefined, name: string): string {
  for (const pair of (query ?? "").split("&")) {
    const at = pair.indexOf("=");
    const key = at < 0 ? pair : pair.slice(0, at);
    if (key !== name) continue;
    try {
      return decodeURIComponent((at < 0 ? "" : pair.slice(at + 1)).replace(/\+/g, " "));
    } catch {
      return "";
    }
  }
  return "";
}

/**
 * Reads a YouTube or Vimeo link in one of the forms people copy (as Go's CheckVideo):
 * - YouTube: youtube.com/watch?v=<id> (www., m. or none), youtu.be/<id>, youtube.com/embed/<id>
 *   and youtube-nocookie.com/embed/<id>;
 * - Vimeo: vimeo.com/<id>, the private vimeo.com/<id>/<hash>, and player.vimeo.com/video/<id>
 *   (with ?h=<hash> when private).
 * Only https, with no user name or port. null for anything else.
 */
export function checkVideo(raw: string): VideoRef | null {
  const link = (raw ?? "").trim();
  if (!link || link.length > MAX_URL || /\s/.test(link)) return null;
  const m = LINK.exec(link);
  if (!m || m[1].toLowerCase() !== "https") return null;
  const authority = m[2];
  if (!authority || authority.includes("@") || authority.includes(":")) return null;
  const host = authority.toLowerCase().replace(/^www\./, "");
  const path = m[3];
  const query = m[4];
  const trimmed = path.replace(/^\/+|\/+$/g, "");
  const parts = path === "" || path === "/" ? [] : trimmed.split("/");
  let v: VideoRef | null = null;
  switch (host) {
    case "youtube.com":
    case "m.youtube.com":
      if (path === "/watch") v = { provider: "youtube", id: param(query, "v") };
      else if (parts.length === 2 && parts[0] === "embed") v = { provider: "youtube", id: parts[1] };
      break;
    case "youtube-nocookie.com":
      if (parts.length === 2 && parts[0] === "embed") v = { provider: "youtube", id: parts[1] };
      break;
    case "youtu.be":
      if (parts.length === 1) v = { provider: "youtube", id: parts[0] };
      break;
    case "vimeo.com":
      if (parts.length === 1) v = { provider: "vimeo", id: parts[0] };
      if (parts.length === 2) {
        if (!VIMEO_HASH.test(parts[1])) return null;
        v = { provider: "vimeo", id: parts[0], hash: parts[1] };
      }
      break;
    case "player.vimeo.com":
      if (parts.length === 2 && parts[0] === "video") {
        const hash = param(query, "h");
        if (hash && !VIMEO_HASH.test(hash)) return null;
        v = { provider: "vimeo", id: parts[1], ...(hash ? { hash } : {}) };
      }
      break;
  }
  if (v?.provider === "youtube") return YOUTUBE_ID.test(v.id) ? v : null;
  if (v?.provider === "vimeo") return VIMEO_ID.test(v.id) ? v : null;
  return null;
}

/**
 * The player's address for a video: YouTube through youtube-nocookie.com, Vimeo with dnt=1 (and its
 * private hash). Built only from a checked ID, never from the link itself.
 */
export function embedUrl(v: VideoRef): string {
  if (v.provider === "youtube") return `https://www.youtube-nocookie.com/embed/${v.id}`;
  return `https://player.vimeo.com/video/${v.id}?dnt=1${v.hash ? `&h=${v.hash}` : ""}`;
}

/** The player's address for a link, or null when the link isn't a YouTube or Vimeo video. */
export function videoEmbedFor(link: unknown): string | null {
  const v = typeof link === "string" ? checkVideo(link) : null;
  return v ? embedUrl(v) : null;
}

/** "YouTube" or "Vimeo", for the editor's card. */
export const providerName = (v: VideoRef) => (v.provider === "youtube" ? "YouTube" : "Vimeo");

const two = (n: number) => String(n).padStart(2, "0");

/** A length in seconds as "3:20", or "1:02:05" from an hour; "" for none. */
export function formatLength(seconds: number | null | undefined): string {
  if (seconds == null || !Number.isFinite(seconds) || seconds <= 0) return "";
  const s = Math.round(seconds);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  return h > 0 ? `${h}:${two(m)}:${two(s % 60)}` : `${m}:${two(s % 60)}`;
}

/** The longest length a video may have, in seconds (4 hours, as Go's content.MaxLength). */
export const MAX_VIDEO_LENGTH = 4 * 60 * 60;

/** A typed length, "m:ss", "h:mm:ss" or whole minutes, in seconds; null if it can't be read or is 0. */
export function parseLength(text: string): number | null {
  const t = text.trim();
  let seconds: number | null = null;
  let m: RegExpMatchArray | null;
  if ((m = t.match(/^(\d{1,3})$/))) seconds = Number(m[1]) * 60;
  else if ((m = t.match(/^(\d{1,3}):([0-5]\d)$/))) seconds = Number(m[1]) * 60 + Number(m[2]);
  else if ((m = t.match(/^(\d{1,2}):([0-5]\d):([0-5]\d)$/))) seconds = Number(m[1]) * 3600 + Number(m[2]) * 60 + Number(m[3]);
  return seconds ? seconds : null;
}

/** The settings a video's tick changes: on with its words, or off with Required too. */
export function tickPatch(on: boolean): Partial<ElementJson> {
  return on
    ? { watchTick: true, tickText: DEFAULT_TICK_TEXT }
    : { watchTick: undefined, tickText: undefined, isRequired: undefined, requiredErrorText: undefined };
}

/** A video in the content library (S33), as the picker's search returns it. */
export type LibraryVideo = { url: string; title: string; lengthSeconds: number | null };

/**
 * Puts a library video into a Video element: the questionnaire keeps its own copy of the link, title
 * and length (S34 answers 1), so a later change to the library item changes nothing here. The old
 * title goes with its translations: they were of another video. The tick settings stay.
 */
export function chooseLibraryVideo(doc: ChapterJson, name: string, item: LibraryVideo): ChapterJson {
  return edit(doc, (d) => {
    const el = findElement(d, name)?.el;
    if (!el) return;
    el.title = item.title;
    el.videoUrl = item.url;
    if (item.lengthSeconds) el.videoLength = item.lengthSeconds;
    else delete el.videoLength;
  });
}

/** What a video needs before the HQ can be published: a YouTube or Vimeo link, and its length. */
export function videoProblems(doc: ChapterJson): string[] {
  const out: string[] = [];
  for (const { el } of allElements(doc)) {
    if (!isVideo(el)) continue;
    const name = textOf(el.title) || el.name;
    const link = typeof el.videoUrl === "string" ? el.videoUrl.trim() : "";
    if (!link) out.push(`The video “${name}” has no link: paste a YouTube or Vimeo link, or choose one from the library`);
    else if (!checkVideo(link)) out.push(`The video “${name}” has a link that isn't a YouTube or Vimeo video`);
    const length = Number(el.videoLength);
    if (!(length > 0)) out.push(`The video “${name}” has no length`);
  }
  return out;
}

type Variables = { getVariable?: (name: string) => unknown };

/**
 * The Video question in a survey: the player (drawn by the web app) and the tick, whose answer is
 * `true` when ticked and nothing otherwise. For clinicians it's read-only and never required: they
 * see whether the patient ticked it (S34 brief).
 */
export class QuestionVideoModel extends Question {
  constructor(name: string) {
    super(name);
    this.createLocalizableString("tickText", this, false);
  }

  getType(): string {
    return VIDEO_TYPE;
  }

  get videoUrl(): string {
    return this.getPropertyValue("videoUrl", "");
  }
  set videoUrl(v: string) {
    this.setPropertyValue("videoUrl", v);
  }
  get videoLength(): number | undefined {
    return this.getPropertyValue("videoLength");
  }
  set videoLength(v: number | undefined) {
    this.setPropertyValue("videoLength", v);
  }
  get watchTick(): boolean {
    return this.getPropertyValue("watchTick", false);
  }
  set watchTick(v: boolean) {
    this.setPropertyValue("watchTick", v);
  }
  get tickText(): string {
    return this.getLocalizableStringText("tickText") || DEFAULT_TICK_TEXT;
  }
  set tickText(v: string) {
    this.setLocalizableStringText("tickText", v);
  }
  get locTickText() {
    return this.getLocalizableString("tickText");
  }
  /** The player's address, or null if the link isn't a YouTube or Vimeo video. */
  get embedUrl(): string | null {
    return videoEmbedFor(this.videoUrl);
  }
  /** "3:20", or "" with no length. */
  get lengthText(): string {
    return formatLength(this.videoLength);
  }

  /** Shown to a clinician (the survey's {viewer}): read-only, and never required of them. */
  get forClinician(): boolean {
    return (this.survey as unknown as Variables | undefined)?.getVariable?.("viewer") === "clinician";
  }
  public get isReadOnly(): boolean {
    return super.isReadOnly || this.forClinician;
  }
  /** The message when a required tick is missing: the author's own, else one that names the tick. */
  protected onCheckForErrors(errors: SurveyError[], isOnValueChanged: boolean, fireCallback: boolean): void {
    super.onCheckForErrors(errors, isOnValueChanged, fireCallback);
    for (const e of errors) {
      if (e.getErrorType() !== "required" || this.requiredErrorText) continue;
      const own = (err: SurveyError) => (err.text = `Please tick “${this.tickText}” to go on.`);
      own(e);
      e.onUpdateErrorTextCallback = own; // and again in another language

    }
  }
  /** Only a tick counts as an answer (unticking clears it). */
  public isEmpty(): boolean {
    return this.value !== true;
  }
  public hasRequiredError(): boolean {
    return this.isRequired && this.watchTick && !this.forClinician && this.isEmpty();
  }
}

/** Registers the Video type with SurveyJS. Safe to call twice. */
export function registerVideoType(): void {
  if (Serializer.findClass(VIDEO_TYPE)) return;
  Serializer.addClass(
    VIDEO_TYPE,
    [
      { name: "videoUrl", visible: false },
      { name: "videoLength:number", visible: false },
      { name: "watchTick:boolean", default: false, visible: false },
      { name: "tickText", serializationProperty: "locTickText", visible: false },
    ],
    () => new QuestionVideoModel(""),
    "question",
  );
  ElementFactory.Instance.registerElement(VIDEO_TYPE, (name) => new QuestionVideoModel(name));
}
