"use client";
import { useCallback, useState } from "react";
import dynamic from "next/dynamic";
import { useParams } from "next/navigation";
import { PatientSignIn } from "@/features/patient/PatientSignIn";

// The survey is built in the browser only (plan 2.1), so skip server rendering.
const PatientHQ = dynamic(() => import("@/features/patient/PatientHQ"), { ssr: false });

export default function Page() {
  const { token } = useParams<{ token: string }>();
  const [signedIn, setSignedIn] = useState(false);
  const signedOut = useCallback(() => setSignedIn(false), []);
  if (!signedIn) return <PatientSignIn token={token} onSignedIn={() => setSignedIn(true)} />;
  return <PatientHQ onSignedOut={signedOut} />;
}
