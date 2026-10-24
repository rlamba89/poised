// Clickable answers on the POA Summary (S27; Rahul, 24 Oct). Each answer and disclosure on either
// tab opens Validate for its Question Set (…/episodes/:eid/sets/:cid?q=<question>), which shows
// that question, scrolled into view and highlighted briefly (highlightQuestion). Back (the
// browser's, or Validate's "POA Summary" link) returns to the same place: the tab is in the POA
// Summary's URL, and the link clicked, with how far down the screen it was, is kept in the
// browser's session storage until the POA Summary takes it back (saveSpot, takeSpot).
import type { Question, SurveyModel } from "survey-core";

/** Validate for one Question Set at one question. `episodePath` is `<base>/episodes/<id>`. */
export function validateAt(episodePath: string, setId: string, question: string): string {
  return `${episodePath}/sets/${setId}?q=${encodeURIComponent(question)}`;
}

/** The question a Validate address names (its `q`), or "". */
export function questionOf(search: string): string {
  return new URLSearchParams(search).get("q") ?? "";
}

/** Where the clinician was on the POA Summary: the link they clicked, and its distance from the top of the screen. */
export type Spot = { anchor: string; top: number };

type Store = Pick<Storage, "getItem" | "setItem" | "removeItem">;
const key = (episodeId: string) => `poa-spot:${episodeId}`;

/** Keeps the spot for the way back. Without storage (a private window, blocked) Back still works, only at the top. */
export function saveSpot(store: Store | null, episodeId: string, spot: Spot): void {
  try {
    store?.setItem(key(episodeId), JSON.stringify(spot));
  } catch {
    // nothing kept: Back shows the summary from the top
  }
}

/** The spot kept for this episode, once: it's removed as it's read. */
export function takeSpot(store: Store | null, episodeId: string): Spot | null {
  try {
    const raw = store?.getItem(key(episodeId));
    store?.removeItem(key(episodeId));
    const spot = raw ? (JSON.parse(raw) as Partial<Spot>) : null;
    return spot && typeof spot.anchor === "string" && typeof spot.top === "number" ? { anchor: spot.anchor, top: spot.top } : null;
  } catch {
    return null;
  }
}

/**
 * Puts the element back where it was on the screen (`top` px from the top), once it's drawn;
 * the POA Summary's sets load after the page, so it tries for a second at most.
 */
export function restoreSpot(spot: Spot): void {
  let frames = 0;
  const tryIt = () => {
    const el = document.querySelector<HTMLElement>(`[data-poa-anchor="${CSS.escape(spot.anchor)}"]`);
    if (!el) {
      if (frames++ < 60) requestAnimationFrame(tryIt);
      return;
    }
    window.scrollBy(0, el.getBoundingClientRect().top - spot.top);
  };
  requestAnimationFrame(tryIt);
}

/**
 * Shows one question of the Validate page: its page of the set first, then the question scrolled
 * to the middle of the screen and marked (data-jumped, which globals.css highlights) for 3
 * seconds. SurveyJS draws the question after this is called and then settles its layout, so it
 * waits for the question (a second at most) and keeps it in place for a second, or until the
 * clinician scrolls or types. A question the answers hide isn't drawn: then the page alone shows.
 * Returns the clean-up.
 */
export function highlightQuestion(model: SurveyModel, name: string): () => void {
  const q = model.getQuestionByName(name) as Question | null;
  if (!q) return () => {};
  if (q.page && q.page.isVisible && model.currentPage !== q.page) model.currentPage = q.page;
  let stopped = false;
  let frames = 0;
  const cleanups: (() => void)[] = [];
  const stop = () => {
    stopped = true;
    for (const c of cleanups.splice(0)) c();
  };
  const tryIt = () => {
    if (stopped) return;
    const el = document.getElementById(q.id) ?? document.querySelector<HTMLElement>(`[data-name="${CSS.escape(name)}"]`);
    if (!el) {
      if (frames++ < 60) requestAnimationFrame(tryIt);
      return;
    }
    const pin = () => el.scrollIntoView({ block: "center" });
    pin();
    el.dataset.jumped = "true";
    const unmark = setTimeout(() => delete el.dataset.jumped, 3000);
    const settle = new ResizeObserver(pin);
    settle.observe(document.body);
    const userEvents = ["wheel", "touchstart", "keydown", "mousedown"] as const;
    const unpin = () => {
      settle.disconnect();
      for (const e of userEvents) window.removeEventListener(e, unpin);
    };
    for (const e of userEvents) window.addEventListener(e, unpin, { passive: true });
    const timer = setTimeout(unpin, 1000);
    cleanups.push(unpin, () => clearTimeout(timer), () => clearTimeout(unmark), () => delete el.dataset.jumped);
  };
  requestAnimationFrame(tryIt);
  return stop;
}
