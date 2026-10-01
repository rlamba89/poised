"use client";
import dynamic from "next/dynamic";
import { useParams } from "next/navigation";

// The summary runs the surveys in the browser (plan 2.1), so skip server rendering.
const PoaSummary = dynamic(() => import("@/features/episodes/PoaSummary"), { ssr: false });

export default function Page() {
  const { hospitalId, eid } = useParams<{ hospitalId: string; eid: string }>();
  return <PoaSummary hospitalId={hospitalId} episodeId={eid} />;
}
