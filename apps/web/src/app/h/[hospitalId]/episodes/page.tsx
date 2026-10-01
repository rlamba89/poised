"use client";
import { useParams } from "next/navigation";
import { EpisodeList } from "@/features/episodes/EpisodeList";

export default function Page() {
  const { hospitalId } = useParams<{ hospitalId: string }>();
  return <EpisodeList hospitalId={hospitalId} />;
}
