"use client";
import { useParams } from "next/navigation";
import { useHospitalPath } from "@/lib/paths";
import { EpisodePage } from "@/features/episodes/EpisodePage";

export default function Page() {
  const { eid } = useParams<{ eid: string }>();
  return <EpisodePage base={useHospitalPath()} episodeId={eid} />;
}
