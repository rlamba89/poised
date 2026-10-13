import { useParams } from "react-router";
import PatientHQ from "@/features/patient/PatientHQ";

// One of the signed-in patient's episodes: their questionnaire (S14). S15 adds their home, /home.
export function PatientEpisode() {
  const { eid } = useParams() as { eid: string };
  return <PatientHQ key={eid} path={`/p/episodes/${eid}`} />;
}
