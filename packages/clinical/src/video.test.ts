import { describe, expect, it } from "vitest";
import { Model } from "survey-core";
import cases from "../../../apps/api/internal/content/testdata/video_links.json";
import { addElement, groupsToMoveInto, kindOf, newElement, type ChapterJson } from "./doc";
import { canAskFresh } from "./keys";
import { registerClinicalProperties } from "./properties";
import { publishProblems } from "./publish";
import { isSetComplete } from "./respondent";
import { modelFor } from "./testcases";
import { setTranslation, textEntries } from "./translations";
import {
  checkVideo, chooseLibraryVideo, DEFAULT_TICK_TEXT, embedUrl, formatLength, parseLength, QuestionVideoModel, tickPatch, videoEmbedFor, videoProblems,
} from "./video";

registerClinicalProperties();

const YT = "https://www.youtube.com/watch?v=aqz-KE-bpKQ";

describe("the video link check (the same cases as Go's content.CheckVideo)", () => {
  it.each(cases.accepted)("accepts $url", (c) => {
    const v = checkVideo(c.url);
    expect(v).toEqual({ provider: c.provider, id: c.id, ...("hash" in c && c.hash ? { hash: c.hash } : {}) });
  });

  it.each(cases.refused)("refuses %j", (url) => {
    expect(checkVideo(url)).toBeNull();
  });
});

describe("the player's address", () => {
  it("plays YouTube through youtube-nocookie.com", () => {
    expect(embedUrl({ provider: "youtube", id: "aqz-KE-bpKQ" })).toBe("https://www.youtube-nocookie.com/embed/aqz-KE-bpKQ");
    expect(videoEmbedFor("https://youtu.be/aqz-KE-bpKQ?si=abc")).toBe("https://www.youtube-nocookie.com/embed/aqz-KE-bpKQ");
  });

  it("plays Vimeo with dnt=1, and a private link's hash", () => {
    expect(videoEmbedFor("https://vimeo.com/76979871")).toBe("https://player.vimeo.com/video/76979871?dnt=1");
    expect(videoEmbedFor("https://vimeo.com/76979871/8272103f6e")).toBe("https://player.vimeo.com/video/76979871?dnt=1&h=8272103f6e");
  });

  it("is never built from a link that isn't a recognised video", () => {
    for (const bad of ["https://evil.example/embed/aqz-KE-bpKQ", "javascript:alert(1)", "https://www.youtube.com/watch?v=\"><script>", undefined, 42]) {
      expect(videoEmbedFor(bad)).toBeNull();
    }
  });
});

describe("a video's length", () => {
  it("reads and shows m:ss and h:mm:ss", () => {
    expect(parseLength("10:35")).toBe(635);
    expect(parseLength("1:02:05")).toBe(3725);
    expect(parseLength("4")).toBe(240);
    expect(parseLength("1:2")).toBeNull();
    expect(formatLength(635)).toBe("10:35");
    expect(formatLength(3725)).toBe("1:02:05");
    expect(formatLength(undefined)).toBe("");
  });
});

describe("the Video element in the editor", () => {
  it("is a new kind with a placeholder title and no link yet", () => {
    const el = newElement("video", new Set());
    expect(el).toMatchObject({ type: "video", title: "Video" });
    expect(el.name).toMatch(/^q_/);
    expect(kindOf(el)).toBe("video");
  });

  it("never offers \"Always ask fresh\" or the library: its tick is never pre-filled", () => {
    expect(canAskFresh(newElement("video", new Set()))).toBe(false);
  });

  it("is never moved into a repeating group", () => {
    const doc: ChapterJson = {
      pages: [{ name: "p1", elements: [
        { type: "video", name: "q_vid" },
        { type: "paneldynamic", name: "g_rep", templateElements: [] },
        { type: "panel", name: "g_grp", elements: [] },
      ] }],
    };
    expect(groupsToMoveInto(doc, "q_vid").map((g) => g.name)).toEqual(["g_grp"]);
  });

  it("turning the tick off also turns Required off", () => {
    expect(tickPatch(true)).toEqual({ watchTick: true, tickText: DEFAULT_TICK_TEXT });
    expect(tickPatch(false)).toMatchObject({ watchTick: undefined, isRequired: undefined, tickText: undefined });
  });
});

describe("publishing", () => {
  const set = (el: Record<string, unknown>): ChapterJson => ({ pages: [{ name: "p1", elements: [{ type: "video", name: "q_vid", title: "Your anaesthetic", ...el }] }] });

  it("needs a YouTube or Vimeo link and a length", () => {
    expect(videoProblems(set({ videoLength: 60 }))).toEqual([
      "The video “Your anaesthetic” has no link: paste a YouTube or Vimeo link, or choose one from the library",
    ]);
    expect(videoProblems(set({ videoUrl: "https://www.dailymotion.com/video/x7tgad0", videoLength: 60 }))).toEqual([
      "The video “Your anaesthetic” has a link that isn't a YouTube or Vimeo video",
    ]);
    expect(videoProblems(set({ videoUrl: YT }))).toEqual(["The video “Your anaesthetic” has no length"]);
    expect(videoProblems(set({ videoUrl: YT, videoLength: 635 }))).toEqual([]);
  });

  it("blocks Publish, naming the Question Set", () => {
    expect(publishProblems([{ name: "About you", doc: set({ videoLength: 60 }) }])).toEqual([
      { set: "About you", message: "The video “Your anaesthetic” has no link: paste a YouTube or Vimeo link, or choose one from the library" },
    ]);
  });
});

describe("the Video question in a survey", () => {
  const doc = (el: Record<string, unknown>): ChapterJson => ({
    pages: [{ name: "p1", elements: [{ type: "video", name: "q_vid", title: "Your anaesthetic", videoUrl: YT, videoLength: 635, ...el }] }],
  });
  const patient = { name: "Sam Patient" };

  it("loads from the editor's JSON and keeps its settings through a save", () => {
    const m = new Model(doc({ watchTick: true, tickText: DEFAULT_TICK_TEXT, isRequired: true }));
    const q = m.getQuestionByName("q_vid") as QuestionVideoModel;
    expect(q).toBeInstanceOf(QuestionVideoModel);
    expect(q.embedUrl).toBe("https://www.youtube-nocookie.com/embed/aqz-KE-bpKQ");
    expect(q.lengthText).toBe("10:35");
    expect(q.watchTick).toBe(true);
    expect(m.toJSON().pages[0].elements[0]).toMatchObject({ type: "video", videoUrl: YT, videoLength: 635, watchTick: true, isRequired: true, tickText: DEFAULT_TICK_TEXT });
  });

  it("a required tick must be ticked before the set is complete", () => {
    const json = doc({ watchTick: true, isRequired: true });
    expect(isSetComplete(json, {}, "patient", patient)).toBe(false);
    expect(isSetComplete(json, { q_vid: false }, "patient", patient)).toBe(false);
    expect(isSetComplete(json, { q_vid: true }, "patient", patient)).toBe(true);
  });

  it("says what's missing in plain words, unless the author wrote their own message", () => {
    const m = new Model(doc({ watchTick: true, isRequired: true }));
    m.validate(true);
    expect(m.getQuestionByName("q_vid").errors.map((e) => e.text)).toEqual(["Please tick “I have watched this video” to go on."]);
    const own = new Model(doc({ watchTick: true, isRequired: true, requiredErrorText: "Watch it first." }));
    own.validate(true);
    expect(own.getQuestionByName("q_vid").errors.map((e) => e.text)).toEqual(["Watch it first."]);
  });

  it("an optional tick, or none, doesn't hold the set up", () => {
    expect(isSetComplete(doc({ watchTick: true }), {}, "patient", patient)).toBe(true);
    expect(isSetComplete(doc({ isRequired: true }), {}, "patient", patient)).toBe(true);
  });

  it("is read-only for clinicians, and never required of them", () => {
    const m = modelFor(doc({ watchTick: true, isRequired: true }), "clinician", patient);
    const q = m.getQuestionByName("q_vid") as QuestionVideoModel;
    expect(q.isReadOnly).toBe(true);
    expect(m.validate(false)).toBe(true);
    const p = modelFor(doc({ watchTick: true, isRequired: true }), "patient", patient);
    expect((p.getQuestionByName("q_vid") as QuestionVideoModel).isReadOnly).toBe(false);
  });

  it("only a tick is an answer", () => {
    const m = new Model(doc({ watchTick: true }));
    const q = m.getQuestionByName("q_vid") as QuestionVideoModel;
    expect(q.isEmpty()).toBe(true);
    q.value = true;
    expect(q.isEmpty()).toBe(false);
    expect(m.data).toEqual({ q_vid: true });
  });

  it("its title and tick can be translated with the rest of the HQ", () => {
    let json = doc({ watchTick: true, tickText: DEFAULT_TICK_TEXT });
    const entries = textEntries(json);
    expect(entries.map((e) => [e.field, e.english])).toEqual([
      ["Text", "Your anaesthetic"],
      ["Tick", DEFAULT_TICK_TEXT],
    ]);
    for (const e of entries) json = setTranslation(json, e.path, "de", e.field === "Tick" ? "Ich habe das Video gesehen" : "Ihre Narkose");
    const m = new Model(json);
    m.locale = "de";
    const q = m.getQuestionByName("q_vid") as QuestionVideoModel;
    expect(q.locTitle.renderedHtml).toBe("Ihre Narkose");
    expect(q.locTickText.renderedHtml).toBe("Ich habe das Video gesehen");
  });

  it("is added by the editor like any element", () => {
    const { doc: next, name } = addElement({ pages: [{ name: "p1", elements: [] }] }, { page: "p1" }, "video");
    expect(next.pages![0].elements![0]).toMatchObject({ type: "video", name });
  });
});

describe("choosing a video from the library", () => {
  it("copies its link, title and length into the questionnaire, replacing the old title and its translations", () => {
    const doc: ChapterJson = { pages: [{ name: "p1", elements: [
      { type: "video", name: "q_vid", title: { default: "Old", de: "Alt" } as unknown as string, videoUrl: "https://vimeo.com/1", videoLength: 5, watchTick: true, isRequired: true },
    ] }] };
    const next = chooseLibraryVideo(doc, "q_vid", { url: YT, title: "Example: Your anaesthetic", lengthSeconds: 635 });
    expect(next.pages![0].elements![0]).toEqual({
      type: "video", name: "q_vid", title: "Example: Your anaesthetic", videoUrl: YT, videoLength: 635, watchTick: true, isRequired: true,
    });
    expect(doc.pages![0].elements![0].videoUrl).toBe("https://vimeo.com/1"); // the original is unchanged (undo)
  });
});
