import { useHospital } from "@/lib/auth";
import { QuestionnaireList } from "@/features/questionnaires/QuestionnaireList";

export function Questionnaires() {
  const { base } = useHospital();
  return <QuestionnaireList base={base} />;
}
