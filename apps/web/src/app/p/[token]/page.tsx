"use client";
import dynamic from "next/dynamic";
import { useParams } from "next/navigation";

// The survey is built in the browser only (plan 2.1), so skip server rendering.
const PatientHQ = dynamic(() => import("@/features/patient/PatientHQ"), { ssr: false });

export default function Page() {
  const { token } = useParams<{ token: string }>();
  return <PatientHQ token={token} />;
}
