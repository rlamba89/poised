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
const WINDOW_MS = 10_000;

type Store = Pick<Storage, "getItem" | "setItem">;

/**
 * Reloads the page, unless it already did so in the last 10 seconds (so a file that's really
 * missing can't reload forever). Without storage it can't tell, so it doesn't reload.
 * Returns whether it reloaded.
 */
export function reloadOnce(storage: Store | null, reload: () => void, now: number): boolean {
  if (!storage) return false;
  try {
    const last = Number(storage.getItem(KEY) ?? 0);
    if (now - last < WINDOW_MS) return false;
    storage.setItem(KEY, String(now));
  } catch {
    return false;
  }
  reload();
  return true;
}
