import { useParams } from "react-router";
import { ConfirmContact } from "@/features/patient/ConfirmContact";

// The link sent to a patient's new mobile or email, to confirm it (S15, PAT-10).
export function VerifyContact() {
  const { token } = useParams() as { token: string };
  return <ConfirmContact key={token} token={token} />;
}
