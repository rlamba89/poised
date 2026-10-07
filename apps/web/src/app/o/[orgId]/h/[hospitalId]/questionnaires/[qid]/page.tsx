"use client";
import { useParams } from "next/navigation";
import { useHospitalPath } from "@/lib/paths";
import { QuestionnaireEditor } from "@/features/editor/QuestionnaireEditor";

export default function Page() {
  const { hospitalId, qid } = useParams<{ hospitalId: string; qid: string }>();
  return <QuestionnaireEditor hospitalId={hospitalId} base={useHospitalPath()} questionnaireId={qid} />;
}
