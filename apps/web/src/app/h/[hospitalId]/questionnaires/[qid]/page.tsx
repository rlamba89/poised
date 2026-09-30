"use client";
import { useParams } from "next/navigation";
import { QuestionnairePage } from "@/features/chapters/QuestionnairePage";

export default function Page() {
  const { hospitalId, qid } = useParams<{ hospitalId: string; qid: string }>();
  return <QuestionnairePage hospitalId={hospitalId} questionnaireId={qid} />;
}
