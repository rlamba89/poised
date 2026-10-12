import { useParams } from "react-router";
import { EpisodeList } from "@/features/episodes/EpisodeList";

export function Episodes() {
  const { hospitalId } = useParams() as { hospitalId: string };
  return <EpisodeList hospitalId={hospitalId} />;
}
