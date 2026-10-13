import { useParams } from "react-router";
import { InviteLink } from "@/features/patient/InviteLink";

// The patient's invite link (S14): Continue, then their date of birth.
export function InviteLinkPage() {
  const { token } = useParams() as { token: string };
  return <InviteLink key={token} token={token} />;
}
