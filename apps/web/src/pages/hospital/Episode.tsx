import { useParams } from "react-router";
import { useHospital } from "@/lib/auth";
import { EpisodePage } from "@/features/episodes/EpisodePage";

export function Episode() {
  const { eid } = useParams() as { eid: string };
  const { base } = useHospital();
  return <EpisodePage key={eid} base={base} episodeId={eid} />;
}
