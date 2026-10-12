import { useParams } from "react-router";
import PreviewPage from "@/features/preview/PreviewPage";

export function Preview() {
  const { hospitalId, cid } = useParams() as { hospitalId: string; cid: string };
  return <PreviewPage key={cid} hospitalId={hospitalId} chapterId={cid} />;
}
