"use client";
import dynamic from "next/dynamic";
import { useParams } from "next/navigation";
import { useHospitalPath } from "@/lib/paths";

// The survey is built in the browser only (plan 2.1), so skip server rendering.
const ValidateSet = dynamic(() => import("@/features/episodes/ValidateSet"), { ssr: false });

export default function Page() {
  const { eid, cid } = useParams<{ eid: string; cid: string }>();
  return <ValidateSet base={useHospitalPath()} episodeId={eid} chapterId={cid} />;
}
