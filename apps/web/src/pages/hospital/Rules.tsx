import { useParams } from "react-router";
import { useHospital } from "@/lib/auth";
import { RulesPage } from "@/features/rules/RulesPage";
import { HqTestPage } from "@/features/rules/HqTestPage";

// An HQ's ASA rules (S40), and testing them on the whole HQ.
export function Rules() {
  const { qid } = useParams() as { qid: string };
  const { base } = useHospital();
  return <RulesPage key={qid} base={base} questionnaireId={qid} />;
}

export function RulesTest() {
  const { qid } = useParams() as { qid: string };
  const { base } = useHospital();
  return <HqTestPage key={qid} base={base} questionnaireId={qid} />;
}
