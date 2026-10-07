"use client";
import dynamic from "next/dynamic";
import { useParams } from "next/navigation";
import { useHospitalPath } from "@/lib/paths";

// The summary runs the surveys in the browser (plan 2.1), so skip server rendering.
const PoaSummary = dynamic(() => import("@/features/episodes/PoaSummary"), { ssr: false });

export default function Page() {
  const { eid } = useParams<{ eid: string }>();
  return <PoaSummary base={useHospitalPath()} episodeId={eid} />;
}
