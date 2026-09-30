"use client";
import dynamic from "next/dynamic";
import { useParams } from "next/navigation";

// The survey is built in the browser only (plan 2.1), so skip server rendering.
const PreviewPage = dynamic(() => import("@/features/preview/PreviewPage"), { ssr: false });

export default function Page() {
  const { hospitalId, cid } = useParams<{ hospitalId: string; cid: string }>();
  return <PreviewPage hospitalId={hospitalId} chapterId={cid} />;
}
