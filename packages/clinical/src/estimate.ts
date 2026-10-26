// The estimated time for a Question Set (PX-10, plan C5, session S36): "About 4 minutes" on the
// patient's tile and in the editor. Rahul's rule (S36 answers 3): a fixed number of seconds for each
// question the patient sees, plus each video's length, rounded up to whole minutes, at least 1. The
// seconds per question are the deployment's setting (Go's SECONDS_PER_QUESTION, default 15, sent
// with GET /api/config), never a number written here.
//
// Only what the patient gets counts, as Go's chapter.ForPatient strips it: no clinician-only page,
// Group or question and no Clinical Summary. A question shown only under a condition counts (the
// patient might see it: conditions aren't evaluated for an estimate). Information, the profile,
// calculations, Groups and Sections aren't answered, so they don't count; nor does a question set
// hidden (visible: false). A video adds its length, and isn't a question itself, tick or not.
import { kindOf, type ChapterJson, type ElementJson } from "./doc";

/** What a Question Set asks of the patient: the questions they may see, and the videos' total length. */
export type Estimate = { questions: number; videoSeconds: number };

const NOT_QUESTIONS = new Set(["group", "section", "information", "profile", "summary", "calculation", "video"]);

/** The questions and video seconds in what the patient gets of a Question Set. */
export function estimate(doc: ChapterJson): Estimate {
  const out: Estimate = { questions: 0, videoSeconds: 0 };
  const walk = (elements: ElementJson[] | undefined) => {
    for (const el of elements ?? []) {
      if (el.clinicianOnly === true || kindOf(el) === "summary") continue;
      if (kindOf(el) === "video") {
        const length = Number(el.videoLength);
        if (Number.isFinite(length) && length > 0) out.videoSeconds += length;
        continue;
      }
      if (el.visible !== false && !NOT_QUESTIONS.has(kindOf(el))) out.questions++;
      walk(el.elements);
      walk(el.templateElements);
    }
  };
  for (const page of doc.pages ?? []) {
    if (page.clinicianOnly !== true) walk(page.elements);
  }
  return out;
}

/**
 * About how many minutes a Question Set takes: secondsPerQuestion for each question plus the videos,
 * rounded up, at least 1. undefined when there's no usable seconds per question (the config
 * couldn't be loaded), so nothing is shown rather than a guess.
 */
export function estimatedMinutes(doc: ChapterJson, secondsPerQuestion: number | undefined): number | undefined {
  if (secondsPerQuestion === undefined || !Number.isFinite(secondsPerQuestion) || secondsPerQuestion <= 0) return undefined;
  const e = estimate(doc);
  return Math.max(1, Math.ceil((e.questions * secondsPerQuestion + e.videoSeconds) / 60));
}

/** "About 4 minutes", "About 1 minute", or "" for none. */
export function estimateText(minutes: number | undefined): string {
  if (minutes === undefined) return "";
  return `About ${minutes} minute${minutes === 1 ? "" : "s"}`;
}
