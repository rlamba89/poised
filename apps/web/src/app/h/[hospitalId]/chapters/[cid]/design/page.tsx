"use client";
import dynamic from "next/dynamic";
import { useParams } from "next/navigation";

// SurveyJS Creator needs the browser, so it is never server-rendered.
const DesignerPage = dynamic(() => import("@/features/designer/DesignerPage"), { ssr: false });

export default function Page() {
  const { hospitalId, cid } = useParams<{ hospitalId: string; cid: string }>();
  return <DesignerPage hospitalId={hospitalId} chapterId={cid} />;
}
