"use client";
import dynamic from "next/dynamic";
import { useParams } from "next/navigation";
import { useHospitalPath } from "@/lib/paths";

// The survey is built in the browser only (plan 2.1), so skip server rendering.
const PreviewPage = dynamic(() => import("@/features/preview/PreviewPage"), { ssr: false });

export default function Page() {
  const { cid } = useParams<{ cid: string }>();
  return <PreviewPage base={useHospitalPath()} chapterId={cid} />;
}
