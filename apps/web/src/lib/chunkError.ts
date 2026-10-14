// A page's code is downloaded when the page is first opened (routes.tsx). After a deploy, the old
// file names are gone, so that download fails; reloading the page fetches the new ones.

/** Whether an error is a page's code (or its CSS) failing to download, in any browser. */
export function isChunkLoadError(error: unknown): boolean {
  if (!(error instanceof Error)) return false;
  return /Failed to fetch dynamically imported module|error loading dynamically imported module|Importing a module script failed|Unable to preload CSS/i.test(
    error.message,
  );
}

const KEY = "poised:chunk-reload";

type Store = Pick<Storage, "getItem" | "setItem" | "removeItem">;

/**
 * Reloads the page, unless it already did so and no page has rendered since (so a file that's
 * really missing can't reload forever, however slow each try is). Without storage it can't
 * tell, so it doesn't reload. Returns whether it reloaded.
 */
export function reloadOnce(storage: Store | null, reload: () => void): boolean {
  if (!storage) return false;
  try {
    if (storage.getItem(KEY) !== null) return false;
    storage.setItem(KEY, "1");
  } catch {
    return false;
  }
  reload();
  return true;
}

/** A page rendered, so the next missing file may reload again (Root calls it). */
export function pageRendered(storage: Store | null): void {
  try {
    storage?.removeItem(KEY);
  } catch {
    // Nothing to clear without storage: reloadOnce doesn't reload then either.
  }
}

/** The tab's sessionStorage, or null where the browser refuses it (e.g. some private modes). */
export function sessionStore(): Storage | null {
  try {
    return window.sessionStorage;
  } catch {
    return null;
  }
}
