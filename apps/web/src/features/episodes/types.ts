// Episodes as the Go API returns them (apps/api/db/queries/episodes.sql).
export type EpisodeStatus =
  | "hq_not_complete" | "ready_for_review" | "ready_for_poa" | "poa_complete" | "on_hold" | "not_ready" | "ready_for_admission";

export type EpisodeRow = {
  id: string;
  status: EpisodeStatus;
  procedure: string;
  createdAt: string;
  firstName: string;
  lastName: string;
  hospitalNumber: string;
  questionnaireName: string;
  versionNo: number;
};

export type Episode = EpisodeRow & {
  anaesthetic: string;
  consultant: string;
  nurseAsa: number | null;
  anaesthetistAsa: number | null;
  patientSubmittedAt: string | null;
  reviewCompletedAt: string | null;
  reviewCompletedByName: string | null;
  versionId: string;
  patientId: string;
  dateOfBirth: string;
  sex: string;
  phone: string;
  email: string;
};

export type EpisodeEvent = { id: string; kind: "event" | "comment"; text: string; createdAt: string; userName: string | null };

export type Patient = {
  id: string;
  firstName: string;
  lastName: string;
  dateOfBirth: string;
  sex: string;
  hospitalNumber: string;
  phone: string;
  email: string;
};

// Lifebox's Full HQ states with readable keys (apps/api/internal/episode).
export const STATUS_LABELS: Record<EpisodeStatus, string> = {
  hq_not_complete: "HQ not complete",
  ready_for_review: "Ready for review",
  ready_for_poa: "Ready for POA",
  poa_complete: "POA complete",
  on_hold: "On hold",
  not_ready: "Not ready for admission",
  ready_for_admission: "Ready for admission",
};

export const STATUS_COLORS: Record<EpisodeStatus, string> = {
  hq_not_complete: "gray",
  ready_for_review: "orange",
  ready_for_poa: "blue",
  poa_complete: "teal",
  on_hold: "yellow",
  not_ready: "red",
  ready_for_admission: "green",
};

/** Statuses a clinician picks by hand, once the HQ review is complete. */
export const MANUAL_STATUSES: EpisodeStatus[] = ["poa_complete", "on_hold", "not_ready", "ready_for_admission"];

export const patientName = (p: { firstName: string; lastName: string }) => `${p.firstName} ${p.lastName}`;
