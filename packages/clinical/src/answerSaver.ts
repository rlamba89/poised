// Saving a Question Set's answers (the clinician's or the patient's) with the revision check that
// stops a stale tab or device overwriting newer answers (S02a). Each save sends the revision it
// started from; the server answers with the next one, or 409 if someone saved in between.

export type SaveResult = { ok: true; revision: number } | { ok: false; conflict: boolean; message: string };

/**
 * Runs one Question Set's saves one at a time, each with the revision the last save returned, so a
 * tab's own autosave and Validate never clash. After a 409 it stops: later saves aren't sent and
 * report the conflict, until the page reloads the set and makes a new saver.
 *
 * `send` makes the request. A 409 is recognised by the error's `status`, as the web app's ApiError has.
 * `confirmed` is what the server has: the loaded answers, then each body it accepts. Compare with it,
 * not with what's on screen, to decide whether there's anything to save (S02a verify.md finding 1).
 */
export function answerSaver<T>(start: number, send: (body: T, revision: number) => Promise<{ revision: number }>, loaded?: T) {
  let revision = start;
  let confirmed = loaded;
  let conflict: string | null = null;
  let queue: Promise<unknown> = Promise.resolve();

  const run = async (body: T): Promise<SaveResult> => {
    if (conflict !== null) return { ok: false, conflict: true, message: conflict };
    try {
      revision = (await send(body, revision)).revision;
      confirmed = body;
      return { ok: true, revision };
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e);
      if ((e as { status?: number }).status === 409) conflict = message;
      return { ok: false, conflict: conflict !== null, message };
    }
  };

  return {
    /** Sends `body` once every earlier save has finished. */
    save(body: T): Promise<SaveResult> {
      const result = queue.then(() => run(body));
      queue = result;
      return result;
    },
    get revision() {
      return revision;
    },
    /** The last body the server accepted, or the loaded one. */
    get confirmed() {
      return confirmed;
    },
    /** The 409's message once the answers have changed somewhere else, else null. */
    get conflict() {
      return conflict;
    },
  };
}

export type AnswerSaver<T> = ReturnType<typeof answerSaver<T>>;
