import { useParams } from "react-router";
import ValidateSet from "@/features/episodes/ValidateSet";

export function ValidateSetPage() {
  const { hospitalId, eid, cid } = useParams() as { hospitalId: string; eid: string; cid: string };
  return <ValidateSet key={`${eid}/${cid}`} hospitalId={hospitalId} episodeId={eid} chapterId={cid} />;
}
