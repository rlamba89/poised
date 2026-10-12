import { useParams } from "react-router";
import { useHospital } from "@/lib/auth";
import { QuestionnaireEditor } from "@/features/editor/QuestionnaireEditor";

export function Editor() {
  const { qid } = useParams() as { qid: string };
  const { base } = useHospital();
  return <QuestionnaireEditor key={qid} base={base} questionnaireId={qid} />;
}
