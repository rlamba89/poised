import { useParams } from "react-router";
import { QuestionnaireList } from "@/features/questionnaires/QuestionnaireList";

export function Questionnaires() {
  const { hospitalId } = useParams() as { hospitalId: string };
  return <QuestionnaireList hospitalId={hospitalId} />;
}
