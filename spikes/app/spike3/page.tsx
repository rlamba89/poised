"use client";
import dynamic from "next/dynamic";

// SurveyJS Creator touches window/document, so it is client-only.
const SpikeCreator = dynamic(() => import("../../components/SpikeCreator"), { ssr: false });

export default function Page() {
  return <SpikeCreator spike={3} />;
}
