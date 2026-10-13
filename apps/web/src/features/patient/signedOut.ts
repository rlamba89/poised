// What the patient lost when their session ended under them (S14 verification 1): a save or Send
// that got a 401 because 30 minutes passed, staff resent the invite, or the link was locked.
// "You've been signed out" alone would hide it, and nothing may break silently.

/** What was refused: a change in a Question Set (named), and/or sending the answers. */
export type Lost = { unsavedSet?: string; unsent?: boolean };

/** The sentences the signed-out card adds, telling the patient what wasn't kept and what to do. */
export function lostWork(lost: Lost): string[] {
  const notes: string[] = [];
  if (lost.unsavedSet) {
    notes.push(`Your latest change in ${lost.unsavedSet} wasn't saved. When you open the link again, please check your answers in that section.`);
  }
  if (lost.unsent) {
    notes.push("Your answers haven't been sent yet. When you open the link again, check them and tap Send my answers.");
  }
  return notes;
}
