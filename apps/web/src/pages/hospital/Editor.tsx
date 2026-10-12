import { useParams } from "react-router";
import { QuestionnaireEditor } from "@/features/editor/QuestionnaireEditor";

export function Editor() {
  const { hospitalId, qid } = useParams() as { hospitalId: string; qid: string };
  return <QuestionnaireEditor key={qid} hospitalId={hospitalId} questionnaireId={qid} />;
}
