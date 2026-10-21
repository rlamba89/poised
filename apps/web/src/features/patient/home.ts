// The patient's home (S15): what each episode's state looks like, and what its button says.
// "cancelled" (S20): the hospital cancelled the episode, so there's no form and no button.
// "closed" (S22): the hospital archived it, so there's no form either.
export type EpisodeState = "to_do" | "in_progress" | "sent" | "cancelled" | "closed";

export function stateOf(state: EpisodeState): { label: string; color: string; action: string } {
  switch (state) {
    case "cancelled":
      return { label: "Cancelled by the hospital", color: "gray", action: "" };
    case "closed":
      return { label: "Closed by the hospital", color: "gray", action: "" };
    case "sent":
      return { label: "Sent", color: "green", action: "View" };
    case "in_progress":
      return { label: "In progress", color: "orange", action: "Continue" };
    default:
      return { label: "To do", color: "blue", action: "Start" };
  }
}

/**
 * Closing the account (S22, EPI-09): the confirmation's warning about the questionnaires not yet
 * sent (to do or in progress), which won't be sent once the account is closed; "" when there are none.
 */
export function unsentWarning(episodes: { hospital: string; questionnaireName: string; state: EpisodeState }[]): string {
  const unsent = episodes.filter((e) => e.state === "to_do" || e.state === "in_progress").map((e) => `${e.questionnaireName} for ${e.hospital}`);
  if (unsent.length === 0) return "";
  if (unsent.length === 1) return `Your ${unsent[0]} hasn't been sent. If you close your account, it won't be sent.`;
  const list = `${unsent.slice(0, -1).join(", ")}, and ${unsent[unsent.length - 1]}`;
  return `${unsent.length} health questionnaires haven't been sent: ${list}. If you close your account, they won't be sent.`;
}
