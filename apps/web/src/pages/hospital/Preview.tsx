import { useParams } from "react-router";
import { useHospital } from "@/lib/auth";
import PreviewPage from "@/features/preview/PreviewPage";

export function Preview() {
  const { cid } = useParams() as { cid: string };
  const { base } = useHospital();
  return <PreviewPage key={cid} base={base} chapterId={cid} />;
}
