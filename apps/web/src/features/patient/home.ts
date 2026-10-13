// The patient's home (S15): what each episode's state looks like, and what its button says.
export type EpisodeState = "to_do" | "in_progress" | "sent";

export function stateOf(state: EpisodeState): { label: string; color: string; action: string } {
  switch (state) {
    case "sent":
      return { label: "Sent", color: "green", action: "View" };
    case "in_progress":
      return { label: "In progress", color: "orange", action: "Continue" };
    default:
      return { label: "To do", color: "blue", action: "Start" };
  }
}
