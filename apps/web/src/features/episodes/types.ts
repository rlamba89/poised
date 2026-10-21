// Episodes as the Go API returns them (apps/api/db/queries/episodes.sql).
export type EpisodeStatus =
  | "hq_not_complete" | "ready_for_review" | "ready_for_poa" | "poa_complete" | "on_hold" | "not_ready" | "ready_for_admission"
  | "admitted" | "discharged" | "cancelled"; // S20

export type EpisodeRow = {
  id: string;
  status: EpisodeStatus;
  /** The procedure's name from the trust's list (S19), or "" for none. */
  procedure: string;
  /** The planned procedure date and the POA appointment date (EPI-01): YYYY-MM-DD, or null. */
  plannedDate: string | null;
  poaDate: string | null;
  createdAt: string;
  firstName: string;
  lastName: string;
  hospitalNumber: string;
  questionnaireName: string;
  versionNo: number;
};

export type Episode = EpisodeRow & {
  /** The procedure in the trust's list, or null; it may since have been retired. */
  procedureId: string | null;
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
  /** S20: counts the changes staff make; every write sends the one the page loaded (a stale one gets 409). */
  revision: number;
  /** The admission and discharge dates (YYYY-MM-DD), or null. */
  admittedAt: string | null;
  dischargedAt: string | null;
  /** The cancel reason's key (cancelReasons in status.ts), or null. */
  cancelReason: string | null;
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
  admitted: "Admitted",
  discharged: "Discharged",
  cancelled: "Cancelled",
};

export const STATUS_COLORS: Record<EpisodeStatus, string> = {
  hq_not_complete: "gray",
  ready_for_review: "orange",
  ready_for_poa: "blue",
  poa_complete: "teal",
  on_hold: "yellow",
  not_ready: "red",
  ready_for_admission: "green",
  admitted: "indigo",
  discharged: "dark",
  cancelled: "gray",
};

export const patientName = (p: { firstName: string; lastName: string }) => `${p.firstName} ${p.lastName}`;
