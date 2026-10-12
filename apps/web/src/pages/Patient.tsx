import { useParams } from "react-router";
import PatientHQ from "@/features/patient/PatientHQ";

export function Patient() {
  const { token } = useParams() as { token: string };
  return <PatientHQ key={token} token={token} />;
}
