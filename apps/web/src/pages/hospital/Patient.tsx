import { useParams } from "react-router";
import { useHospital } from "@/lib/auth";
import { PatientPage } from "@/features/episodes/PatientPage";

export function Patient() {
  const { pid } = useParams() as { pid: string };
  const { base } = useHospital();
  return <PatientPage key={pid} base={base} patientId={pid} />;
}
