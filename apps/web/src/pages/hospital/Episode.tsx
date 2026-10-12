import { useParams } from "react-router";
import { EpisodePage } from "@/features/episodes/EpisodePage";

export function Episode() {
  const { hospitalId, eid } = useParams() as { hospitalId: string; eid: string };
  return <EpisodePage key={eid} hospitalId={hospitalId} episodeId={eid} />;
}
