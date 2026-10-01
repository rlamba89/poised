"use client";
import dynamic from "next/dynamic";
import { useParams } from "next/navigation";

// The survey is built in the browser only (plan 2.1), so skip server rendering.
const ValidateSet = dynamic(() => import("@/features/episodes/ValidateSet"), { ssr: false });

export default function Page() {
  const { hospitalId, eid, cid } = useParams<{ hospitalId: string; eid: string; cid: string }>();
  return <ValidateSet hospitalId={hospitalId} episodeId={eid} chapterId={cid} />;
}
