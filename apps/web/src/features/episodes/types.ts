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
  /** Who sent the HQ on the patient's behalf (PAT-09), or null: the patient did. */
  patientSubmittedByName: string | null;
  reviewCompletedAt: string | null;
  reviewCompletedByName: string | null;
  versionId: string;
  patientId: string;
  dateOfBirth: string;
  sex: string;
  nationalId: string | null;
  nationalIdType: string | null;
  mobile: string;
  email: string;
  /** The follow-up (S16): when the first invite reached the patient (null: none has), how many reminder days have been dealt with, and when it was marked "not responding". */
  invitedAt: string | null;
  remindersSent: number;
  notRespondingAt: string | null;
};

/** The patient's working invite (S14), or null before one is sent: when, and whether five wrong dates of birth locked it. */
export type EpisodeInvite = { sentAt: string; lockedAt: string | null } | null;

/** A General notes entry. userName is null for the patient's and the automatic ones (reminders, "not responding": S16). */
export type EpisodeEvent = { id: string; kind: "event" | "comment"; text: string; createdAt: string; userName: string | null; automatic: boolean };

/** A search result, or a possible duplicate shown on the New episode page: one of this hospital's patients. */
export type FoundPatient = {
  id: string;
  firstName: string;
  lastName: string;
  dateOfBirth: string;
  nationalId: string | null;
  hospitalNumber: string; // their latest here, or ""
};

/** The trust's record of one of this hospital's patients (GET …/patients/:id). */
export type PatientRecord = {
  id: string;
  firstName: string;
  lastName: string;
  dateOfBirth: string;
  sex: string;
  nationalId: string | null;
  nationalIdType: string | null;
  mobile: string;
  email: string;
};

export const SEXES = [
  { value: "female", label: "Female" },
  { value: "male", label: "Male" },
  { value: "other", label: "Other" },
  { value: "unknown", label: "Not known" },
];

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
