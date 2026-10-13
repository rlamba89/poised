import { useParams } from "react-router";
import { useHospital } from "@/lib/auth";
import { PatientFormPage } from "@/features/episodes/PatientFormPage";

// Staff filling in the patient's HQ for them (PAT-09, S14).
export function PatientForm() {
  const { eid } = useParams() as { eid: string };
  const { base } = useHospital();
  return <PatientFormPage key={eid} base={base} episodeId={eid} />;
}
