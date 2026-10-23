// Step 4 of plan-workflow.md: the clinician validates one Question Set. The clinician view shows
// the patient's answers beside the clinician questions; the clinician corrects answers (each
// change shows what the patient said) and fills in their own. Answers save as they go;
// "Validate" checks required questions and stamps the set. Saves carry the set's revision, so a
// save from a stale tab gets a 409 and the page asks for a reload instead of overwriting (S02a).
// Pre-filled answers (S26, PRE-04) say where they came from, and which the patient changed; the set
// says which button of its opening card the patient pressed.
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate } from "react-router";
import { Alert, Anchor, Badge, Button, Card, Center, Group, Loader, Stack, Text, Title } from "@mantine/core";
import { IconArrowLeft, IconCircleCheck } from "@tabler/icons-react";
import type { Model } from "survey-core";
import { Survey } from "survey-react-ui";
import "survey-core/survey-core.css";
import {
  answerSaver, patientVariables, prefillChoiceText, prefillMarks, refreshClinicalSummaries, registerClinicalProperties, sameAnswer, type Answers,
} from "@poised/clinical";
import { api } from "@/lib/api";
import { formatDateTime, formatDay } from "@/lib/format";
import { clinicianModel } from "@/features/preview/clinicianView";
import { showCorrections } from "./corrections";
import { showPrefillOrigins } from "./prefillOrigins";
import { useEpisode } from "./EpisodePage";
import { currentAnswers, useEpisodeHQ, type EpisodeHQ, type HQSet } from "./hq";
import { patientName, type Episode } from "./types";

registerClinicalProperties();

export default function ValidateSet({ base, episodeId, chapterId }: { base: string; episodeId: string; chapterId: string }) {
  const { data, error: episodeError, reload: reloadEpisode } = useEpisode(base, episodeId);
  const { hq, error: hqError, reload: reloadHQ } = useEpisodeHQ(base, episodeId);
  // Bumped by Reload, so the set is built again from the latest answers.
  const [loads, setLoads] = useState(0);
  const reload = async () => {
    await Promise.all([reloadEpisode(), reloadHQ()]);
    setLoads((n) => n + 1);
  };
  const error = episodeError || hqError;
  if (error) return <Alert color="red">{error}</Alert>;
  if (!data || !hq) return <Center h="40vh"><Loader /></Center>;
  const set = hq.chapters.find((c) => c.id === chapterId);
  if (!set) return <Alert color="red">This Question Set isn&apos;t part of the episode.</Alert>;
  return <Validate key={loads} base={base} episode={data.episode} hq={hq} set={set} onReload={reload} />;
}

const SAVE_DELAY_MS = 600;

type ValidateProps = { base: string; episode: Episode; hq: EpisodeHQ; set: HQSet; onReload: () => Promise<void> };

function Validate({ base, episode, hq, set, onReload }: ValidateProps) {
  const navigate = useNavigate();
  const back = `${base}/episodes/${episode.id}`;
  const path = `${base}/episodes/${episode.id}/answers/${set.id}`;
  const editable = episode.status === "ready_for_review";
  const row = hq.clinician[set.id];
  const patient = hq.patient[set.id];
  const prefill = hq.prefill[set.id];
  const [status, setStatus] = useState<"saved" | "saving" | "error">("saved");
  const [error, setError] = useState("");
  // The 409's message: the answers changed somewhere else, so nothing more is saved until Reload.
  const [conflict, setConflict] = useState("");
  const [reloading, setReloading] = useState(false);
  const [validated, setValidated] = useState(row?.validatedAt ? row : null);
  const saved = useRef<Answers>(currentAnswers(hq, set.id));
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);

  const model: Model = useMemo(() => {
    const m = clinicianModel(set.content, patientVariables(episode));
    m.showCompleteButton = false; // "Validate" replaces it
    m.readOnly = !editable;
    if (patient) showCorrections(m, patient);
    if (patient && prefill) showPrefillOrigins(m, prefill, patient);
    m.data = currentAnswers(hq, set.id);
    return m;
    // Built once: answers then live in the model.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [set.id]);

  // The set's pre-filled answers shown when it opened (S26), for the line above it.
  const prefilled = useMemo(() => (patient && prefill ? prefillMarks(model, prefill.answers, patient) : []), [model, patient, prefill]);

  // Saves run one at a time, each with the revision the last one returned. Built once, like the model.
  const saver = useMemo(
    () =>
      answerSaver<{ data: Answers; validated: boolean }>(row?.revision ?? 0, (body, revision) =>
        api<{ revision: number }>(path, { method: "PUT", body: JSON.stringify({ ...body, revision }) }),
      ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [path],
  );

  const save = useCallback(
    async (validate: boolean) => {
      clearTimeout(timer.current);
      const data = model.data as Answers;
      // Nothing changed (calculations re-run on load), so keep the stamp.
      if (!validate && sameAnswer(data, saved.current)) return true;
      setStatus("saving");
      const result = await saver.save({ data, validated: validate });
      if (result.ok) {
        saved.current = data;
        setStatus("saved");
        if (!validate) setValidated(null); // a change after validating needs validating again
        return true;
      }
      if (result.conflict) {
        model.readOnly = true; // no more changes that couldn't be saved
        setConflict(result.message);
      } else {
        setError(result.message);
      }
      setStatus("error");
      return false;
    },
    [model, saver],
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
    if (await save(true)) navigate(back);
  };

  return (
    <Stack>
      <Anchor component={Link} to={back} size="sm">
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
          {editable && <Button onClick={validate} disabled={!!conflict} leftSection={<IconCircleCheck size={16} />}>Validate</Button>}
        </Group>
      </Group>
      {!editable && (
        <Alert color="gray">
          {episode.status === "hq_not_complete" ? "The patient hasn't sent their answers yet."
            : episode.status === "archived" ? "This episode is archived, so these answers can't be changed."
              : episode.status === "cancelled" ? "This episode is cancelled, so these answers can't be changed."
                : "The HQ review is complete, so these answers can't be changed."}
        </Alert>
      )}
      {editable && set.audience === "patient" && !patient && <Alert color="orange">The patient didn&apos;t answer this Question Set.</Alert>}
      {hq.enteredBy[set.id] && (
        <Alert color="orange" data-testid="entered-by">
          The patient&apos;s answers here were entered by {hq.enteredBy[set.id]} on the patient&apos;s behalf.
        </Alert>
      )}
      {prefill && prefilled.length > 0 && (
        <Alert color="brand" variant="light" data-testid="prefill-set">
          {prefilled.length === 1 ? "1 answer here was" : `${prefilled.length} answers here were`} pre-filled from the patient&apos;s earlier
          questionnaires{prefilled.some((m) => m.changed) ? `, and the patient changed ${prefilled.filter((m) => m.changed).length}` : ""}.{" "}
          {prefillChoiceText(prefill, formatDay)}
        </Alert>
      )}
      {conflict && (
        <Alert color="red" title="Not saved">
          <Group justify="space-between" wrap="nowrap">
            <Text size="sm">{conflict}</Text>
            <Button size="xs" style={{ flexShrink: 0 }} loading={reloading} onClick={() => { setReloading(true); onReload(); }}>Reload</Button>
          </Group>
        </Alert>
      )}
      {error && <Alert color="red">{error}</Alert>}
      <Card withBorder padding={0} className="sj-clinician-layout">
        <Survey model={model} />
      </Card>
    </Stack>
  );
}
