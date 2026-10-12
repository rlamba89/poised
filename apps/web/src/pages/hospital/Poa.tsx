import { useParams } from "react-router";
import { useHospital } from "@/lib/auth";
import PoaSummary from "@/features/episodes/PoaSummary";

export function Poa() {
  const { eid } = useParams() as { eid: string };
  const { base } = useHospital();
  return <PoaSummary key={eid} base={base} episodeId={eid} />;
}
