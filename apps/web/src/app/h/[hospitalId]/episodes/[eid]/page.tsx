"use client";
import { useParams } from "next/navigation";
import { EpisodePage } from "@/features/episodes/EpisodePage";

export default function Page() {
  const { hospitalId, eid } = useParams<{ hospitalId: string; eid: string }>();
  return <EpisodePage hospitalId={hospitalId} episodeId={eid} />;
}
