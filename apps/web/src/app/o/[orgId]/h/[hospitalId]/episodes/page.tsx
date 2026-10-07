"use client";
import { useHospitalPath } from "@/lib/paths";
import { EpisodeList } from "@/features/episodes/EpisodeList";

export default function Page() {
  return <EpisodeList base={useHospitalPath()} />;
}
