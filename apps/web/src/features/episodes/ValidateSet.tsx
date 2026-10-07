"use client";
// Step 4 of plan-workflow.md: the clinician validates one Question Set. The clinician view shows
// the patient's answers beside the clinician questions; the clinician corrects answers (each
// change shows what the patient said) and fills in their own. Answers save as they go;
// "Validate" checks required questions and stamps the set.
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Alert, Anchor, Badge, Button, Card, Center, Group, Loader, Stack, Text, Title } from "@mantine/core";
import { IconArrowLeft, IconCircleCheck } from "@tabler/icons-react";
import type { Model } from "survey-core";
import { Survey } from "survey-react-ui";
import "survey-core/survey-core.css";
import { patientVariables, refreshClinicalSummaries, registerClinicalProperties, sameAnswer, type Answers } from "@poised/clinical";
import { api } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { clinicianModel } from "@/features/preview/clinicianView";
import { showCorrections } from "./corrections";
import { useEpisode } from "./EpisodePage";
import { currentAnswers, useEpisodeHQ, type EpisodeHQ, type HQSet } from "./hq";
import { patientName, type Episode } from "./types";

registerClinicalProperties();

export default function ValidateSet({ base, episodeId, chapterId }: { base: string; episodeId: string; chapterId: string }) {
  const { data, error: episodeError } = useEpisode(base, episodeId);
  const { hq, error: hqError } = useEpisodeHQ(base, episodeId);
  const error = episodeError || hqError;
  if (error) return <Alert color="red">{error}</Alert>;
  if (!data || !hq) return <Center h="40vh"><Loader /></Center>;
  const set = hq.chapters.find((c) => c.id === chapterId);
  if (!set) return <Alert color="red">This Question Set isn&apos;t part of the episode.</Alert>;
  return <Validate base={base} episode={data.episode} hq={hq} set={set} />;
}

const SAVE_DELAY_MS = 600;

function Validate({ base, episode, hq, set }: { base: string; episode: Episode; hq: EpisodeHQ; set: HQSet }) {
  const router = useRouter();
  const back = `${base}/episodes/${episode.id}`;
  const path = `${base}/episodes/${episode.id}/answers/${set.id}`;
  const editable = episode.status === "ready_for_review";
  const row = hq.clinician[set.id];
  const patient = hq.patient[set.id];
  const [status, setStatus] = useState<"saved" | "saving" | "error">("saved");
  const [error, setError] = useState("");
  const [validated, setValidated] = useState(row?.validatedAt ? row : null);
  const saved = useRef<Answers>(currentAnswers(hq, set.id));
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);

  const model: Model = useMemo(() => {
    const m = clinicianModel(set.content, patientVariables(episode));
    m.showCompleteButton = false; // "Validate" replaces it
    m.readOnly = !editable;
    if (patient) showCorrections(m, patient);
    m.data = currentAnswers(hq, set.id);
    return m;
    // Built once: answers then live in the model.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [set.id]);

  const save = useCallback(
    async (validate: boolean) => {
      clearTimeout(timer.current);
      const data = model.data as Answers;
      // Nothing changed (calculations re-run on load), so keep the stamp.
      if (!validate && sameAnswer(data, saved.current)) return true;
      setStatus("saving");
      try {
        await api(path, { method: "PUT", body: JSON.stringify({ data, validated: validate }) });
        saved.current = data;
        setStatus("saved");
        if (!validate) setValidated(null); // a change after validating needs validating again
        return true;
      } catch (e) {
        setError((e as Error).message);
        setStatus("error");
        return false;
      }
    },
    [model, path],
  );

  useEffect(() => {
    refreshClinicalSummaries(model);
    const changed = () => {
      refreshClinicalSummaries(model);
      clearTimeout(timer.current);
      timer.current = setTimeout(() => save(false), SAVE_DELAY_MS);
    };
    model.onValueChanged.add(changed);
    return () => model.onValueChanged.remove(changed);
  }, [model, save]);

  const validate = async () => {
    setError("");
    if (!model.validate(true, true)) {
      setError("Some required questions have no answer yet.");
      return;
    }
    if (await save(true)) router.push(back);
  };

  return (
    <Stack>
      <Anchor component={Link} href={back} size="sm">
        <Group gap={4}><IconArrowLeft size={14} /> {patientName(episode)}</Group>
      </Anchor>
      <Group justify="space-between">
        <Group gap="sm">
          <Title order={3}>{set.name}</Title>
          {validated?.validatedAt && (
            <Badge color="green" variant="light" leftSection={<IconCircleCheck size={12} />}>
              Validated by {validated.updatedByName} on {formatDateTime(validated.validatedAt)}
            </Badge>
          )}
        </Group>
        <Group gap="sm">
          {editable && <Text size="sm" c={status === "error" ? "red" : "dimmed"} data-testid="validate-save-status">{status === "saving" ? "Saving…" : status === "error" ? "Not saved" : "Saved"}</Text>}
          {editable && <Button onClick={validate} leftSection={<IconCircleCheck size={16} />}>Validate</Button>}
        </Group>
      </Group>
      {!editable && (
        <Alert color="gray">
          {episode.status === "hq_not_complete" ? "The patient hasn't sent their answers yet." : "The HQ review is complete, so these answers can't be changed."}
        </Alert>
      )}
      {editable && set.audience === "patient" && !patient && <Alert color="orange">The patient didn&apos;t answer this Question Set.</Alert>}
      {error && <Alert color="red">{error}</Alert>}
      <Card withBorder padding={0} className="sj-clinician-layout">
        <Survey model={model} />
      </Card>
    </Stack>
  );
}
