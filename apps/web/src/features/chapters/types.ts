// A Question Set (chapter) as the questionnaire API lists it, without its content.
export type Audience = "patient" | "clinician" | "clinician_document";

export type Chapter = { id: string; position: number; name: string; description: string; icon: string; audience: Audience; revision: number };

export const AUDIENCE_LABELS: Record<Audience, string> = {
  patient: "Patient",
  clinician: "Clinician",
  clinician_document: "Clinician document",
};
