import { useParams } from "react-router";
import PoaSummary from "@/features/episodes/PoaSummary";

export function Poa() {
  const { hospitalId, eid } = useParams() as { hospitalId: string; eid: string };
  return <PoaSummary key={eid} hospitalId={hospitalId} episodeId={eid} />;
}
