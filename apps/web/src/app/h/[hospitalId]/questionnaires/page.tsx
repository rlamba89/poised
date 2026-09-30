"use client";
import { useParams } from "next/navigation";
import { QuestionnaireList } from "@/features/questionnaires/QuestionnaireList";

export default function Page() {
  const { hospitalId } = useParams<{ hospitalId: string }>();
  return <QuestionnaireList hospitalId={hospitalId} />;
}
