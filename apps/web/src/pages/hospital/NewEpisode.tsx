import { useSearchParams } from "react-router";
import { useHospital } from "@/lib/auth";
import { NewEpisodePage } from "@/features/episodes/NewEpisodePage";

export function NewEpisode() {
  const { base } = useHospital();
  const [params] = useSearchParams();
  const patient = params.get("patient"); // from a patient's page: that patient, already chosen
  // Keyed by the hospital only: Change patient drops ?patient= without starting the page afresh.
  return <NewEpisodePage key={base} base={base} patientId={patient} />;
}
