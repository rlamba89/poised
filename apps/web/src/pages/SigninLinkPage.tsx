import { useParams } from "react-router";
import { InviteLink } from "@/features/patient/InviteLink";

// A returning patient's sign-in link (S15): Continue, then their date of birth, then their home.
export function SigninLinkPage() {
  const { token } = useParams() as { token: string };
  return <InviteLink key={token} token={token} kind="signin" />;
}
