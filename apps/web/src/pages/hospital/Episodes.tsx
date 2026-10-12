import { useHospital } from "@/lib/auth";
import { EpisodeList } from "@/features/episodes/EpisodeList";

export function Episodes() {
  const { base } = useHospital();
  return <EpisodeList base={base} />;
}
