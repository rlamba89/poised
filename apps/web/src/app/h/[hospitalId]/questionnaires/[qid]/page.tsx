"use client";
import { useParams } from "next/navigation";
import { QuestionnaireEditor } from "@/features/editor/QuestionnaireEditor";

export default function Page() {
  const { hospitalId, qid } = useParams<{ hospitalId: string; qid: string }>();
  return <QuestionnaireEditor hospitalId={hospitalId} questionnaireId={qid} />;
}
