// The patient's home (S15): what each episode's state looks like, and what its button says.
// "cancelled" (S20): the hospital cancelled the episode, so there's no form and no button.
export type EpisodeState = "to_do" | "in_progress" | "sent" | "cancelled";

export function stateOf(state: EpisodeState): { label: string; color: string; action: string } {
  switch (state) {
    case "cancelled":
      return { label: "Cancelled by the hospital", color: "gray", action: "" };
    case "sent":
      return { label: "Sent", color: "green", action: "View" };
    case "in_progress":
      return { label: "In progress", color: "orange", action: "Continue" };
    default:
      return { label: "To do", color: "blue", action: "Start" };
  }
}
