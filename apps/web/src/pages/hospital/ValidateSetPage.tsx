import { useParams } from "react-router";
import { useHospital } from "@/lib/auth";
import ValidateSet from "@/features/episodes/ValidateSet";

export function ValidateSetPage() {
  const { eid, cid } = useParams() as { eid: string; cid: string };
  const { base } = useHospital();
  return <ValidateSet key={`${eid}/${cid}`} base={base} episodeId={eid} chapterId={cid} />;
}
