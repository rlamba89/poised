import { useParams } from "react-router";
import { TrustPage } from "@/features/admin/TrustPage";

export function Trust() {
  const { orgId } = useParams() as { orgId: string };
  return <TrustPage key={orgId} orgId={orgId} />;
}
