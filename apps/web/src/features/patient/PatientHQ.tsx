"use client";
// Step 3 of plan-workflow.md: the patient fills in their HQ through their link (no sign-in).
// Question Sets are listed as tiles; each opens in the patient view, answers save as they go
// (PX-05), and "Send my answers" submits the whole HQ once every set is complete. Saves carry the
// set's revision: if the link is open on another device that saved since, the save gets a 409 and
// the page asks for a reload instead of overwriting (S02a).
//
// Two copies of the answers, kept apart (S02a verify.md, verifications 1 and 2):
//   - what the server has: the answers loaded with the HQ, then each save it accepts (each set's
//     saver tracks them);
//   - what's on screen (drafts). A set saves whenever its draft differs from what the server has.
// A set counts as complete only when the server's answers are complete and its draft is saved
// (isSetReadyToSend), so a change whose save failed keeps "Send my answers" disabled.
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Alert, Badge, Box, Button, Card, Center, Container, Group, Loader, Modal, Stack, Text, Title } from "@mantine/core";
import { IconArrowLeft, IconCircleCheck } from "@tabler/icons-react";
import { Model } from "survey-core";
import { Survey } from "survey-react-ui";
import "survey-core/survey-core.css";
import {
  answerSaver, isSetReadyToSend, modelFor, patientVariables, registerClinicalProperties, sameAnswer, shownSets, type Answers, type AnswerSaver,
  type ChapterJson, type EpisodePatient,
} from "@poised/clinical";
import { api } from "@/lib/api";
import { showUnits } from "@/features/preview/units";

registerClinicalProperties();

type QuestionSet = { id: string; name: string; description: string; icon: string; content: ChapterJson };
/** The HQ as loaded. Nothing changes it afterwards: later saves are tracked by each set's saver. */
type HQ = {
  patient: EpisodePatient;
  questionnaireName: string;
  submitted: boolean;
  chapters: QuestionSet[];
  answers: Record<string, Answers>;
  /** Each saved set's revision, which its first save sends back. A set with no answers yet has none. */
  revisions: Record<string, number>;
};

const SAVE_DELAY_MS = 600;

export default function PatientHQ({ token }: { token: string }) {
  const [hq, setHq] = useState<HQ | null>(null);
  const [error, setError] = useState("");
  const [openId, setOpenId] = useState<string | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [sending, setSending] = useState(false);
  // The 409's message: the answers changed on another device (or were sent), so nothing more is
  // saved until Reload.
  const [conflict, setConflict] = useState("");
  // Bumped by each load, so an open set is built again from the latest answers.
  const [loads, setLoads] = useState(0);
  // What's on screen, per set, where it differs from what was loaded.
  const [drafts, setDrafts] = useState<Record<string, Answers>>({});
  // One saver per set for the life of the page (until Reload): its revision and what the server
  // has confirmed survive closing and reopening the set.
  const savers = useRef(new Map<string, AnswerSaver<Answers>>());

  const [reloading, setReloading] = useState(false);

  const load = useCallback(() => {
    setReloading(true);
    return api<HQ>(`/p/${token}`)
      .then((h) => {
        savers.current = new Map();
        setDrafts({});
        setHq(h);
        setConflict("");
        setError(""); // an earlier failure's message no longer applies
        setLoads((n) => n + 1);
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setReloading(false));
  }, [token]);
  useEffect(() => {
    load();
  }, [load]);

  const vars = useMemo(() => (hq ? patientVariables(hq.patient) : null), [hq]);

  const setDraft = useCallback((id: string, data: Answers) => setDrafts((d) => ({ ...d, [id]: data })), []);

  if (!hq || !vars) {
    return error ? <Alert color="red" m="md">{error}</Alert> : <Center h="60vh"><Loader /></Center>;
  }

  const saverFor = (id: string) => {
    let saver = savers.current.get(id);
    if (!saver) {
      saver = answerSaver<Answers>(
        hq.revisions[id] ?? 0,
        (data, revision) => api<{ revision: number }>(`/p/${token}/answers/${id}`, { method: "PUT", body: JSON.stringify({ data, revision }) }),
        hq.answers[id] ?? {},
      );
      savers.current.set(id, saver);
    }
    return saver;
  };
  // What the server has for each set.
  const saved: Record<string, Answers> = { ...hq.answers };
  for (const [id, saver] of savers.current) saved[id] = saver.confirmed ?? {};

  const sets = shownSets(hq.chapters, saved, "patient", vars);
  const open = sets.find((s) => s.id === openId);
  // Building a survey per set is not free, so only while the list is showing.
  const done = new Set(open || hq.submitted ? [] : sets.filter((s) => isSetReadyToSend(s.content, saved[s.id], drafts[s.id], vars)).map((s) => s.id));

  const send = async () => {
    setSending(true);
    try {
      await api(`/p/${token}/submit`, { method: "POST" });
      setHq({ ...hq, submitted: true });
      setConfirming(false);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSending(false);
    }
  };

  return (
    <Box bg="gray.0" mih="100vh">
      <Box bg="dark.8" c="white" py="sm" px="md">
        <Text fw={700}>Lifebox</Text>
      </Box>
      <Container size="md" py="lg">
        {error && <Alert color="red" mb="md" withCloseButton onClose={() => setError("")}>{error}</Alert>}
        {conflict ? (
          <Card withBorder p="xl" data-testid="patient-conflict">
            <Title order={3}>Your answers weren&apos;t saved</Title>
            <Text mt="xs">{conflict}</Text>
            <Button mt="md" loading={reloading} onClick={load}>Reload</Button>
          </Card>
        ) : hq.submitted ? (
          <Card withBorder p="xl" ta="center">
            <IconCircleCheck size={48} color="var(--mantine-color-green-6)" style={{ margin: "0 auto" }} />
            <Title order={3} mt="sm">Thank you, {hq.patient.firstName}</Title>
            <Text mt="xs">Your answers have been sent to the hospital. A member of the team will review them.</Text>
          </Card>
        ) : open ? (
          <SetForm
            key={`${open.id}-${loads}`}
            set={open}
            data={drafts[open.id] ?? saved[open.id] ?? {}}
            saver={saverFor(open.id)}
            vars={vars}
            onChange={(data) => setDraft(open.id, data)}
            onConflict={setConflict}
            onError={setError}
            onClose={() => setOpenId(null)}
          />
        ) : (
          <Stack>
            <div>
              <Title order={2}>{hq.questionnaireName}</Title>
              <Text c="dimmed">
                Hello {hq.patient.firstName}. Please complete each section. Your answers are saved as you go, so you can come back later.
              </Text>
            </div>
            {sets.map((s) => {
              const started = Object.keys(drafts[s.id] ?? saved[s.id] ?? {}).length > 0;
              return (
                <Card key={s.id} withBorder data-testid={`set-${s.name}`}>
                  <Group justify="space-between" wrap="nowrap">
                    <div>
                      <Text fw={700}>{s.name}</Text>
                      {s.description && <Text size="sm" c="dimmed">{s.description}</Text>}
                    </div>
                    <Group gap="sm" wrap="nowrap">
                      {done.has(s.id) ? (
                        <Badge color="green" variant="light">Complete</Badge>
                      ) : (
                        started && <Badge color="orange" variant="light">In progress</Badge>
                      )}
                      <Button variant={done.has(s.id) ? "default" : "filled"} onClick={() => setOpenId(s.id)}>
                        {done.has(s.id) ? "Review" : started ? "Continue" : "Start"}
                      </Button>
                    </Group>
                  </Group>
                </Card>
              );
            })}
            <Group justify="flex-end">
              <Text size="sm" c="dimmed">{done.size} of {sets.length} sections complete</Text>
              <Button size="md" disabled={done.size < sets.length} onClick={() => setConfirming(true)}>Send my answers</Button>
            </Group>
          </Stack>
        )}
      </Container>
      <Modal opened={confirming} onClose={() => setConfirming(false)} title="Send your answers?">
        <Stack>
          <Text>Once sent, you can&apos;t change your answers here. The hospital team will contact you if they need anything else.</Text>
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setConfirming(false)}>Not yet</Button>
            <Button onClick={send} loading={sending}>Send my answers</Button>
          </Group>
        </Stack>
      </Modal>
    </Box>
  );
}

type SetFormProps = {
  set: QuestionSet;
  /** What to show: the set's draft, else what the server has. */
  data: Answers;
  saver: AnswerSaver<Answers>;
  vars: ReturnType<typeof patientVariables>;
  onChange: (data: Answers) => void;
  /** Called with the 409's message; the page then shows it with Reload instead of the set. */
  onConflict: (message: string) => void;
  onError: (message: string) => void;
  onClose: () => void;
};

/** One Question Set in the patient view. Answers save shortly after each change. */
function SetForm({ set, data, saver, vars, onChange, onConflict, onError, onClose }: SetFormProps) {
  // "Not saved" while what's on screen differs from what the server has (e.g. after a failed save).
  const [status, setStatus] = useState<"saving" | "saved" | "unsaved">(() => (sameAnswer(data, saver.confirmed ?? {}) ? "saved" : "unsaved"));
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const latest = useRef(data);

  const model = useMemo(() => {
    const m: Model = modelFor(set.content, "patient", vars);
    showUnits(m);
    m.showPageTitles = false;
    m.completeText = "Finish this section";
    m.data = data;
    return m;
    // Built once per set: answers then live in the model.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [set]);

  /** Saves the latest answers; false if the answers changed on another device. */
  const save = useCallback(async () => {
    clearTimeout(timer.current);
    // Nothing to save if the server already has these answers: a tab left open elsewhere mustn't
    // overwrite newer answers, or be told it can't. A save that failed is sent again here.
    const answers = latest.current;
    if (sameAnswer(answers, saver.confirmed ?? {})) {
      setStatus("saved");
      return true;
    }
    setStatus("saving");
    const result = await saver.save(answers);
    setStatus(result.ok ? "saved" : "unsaved");
    if (!result.ok && result.conflict) onConflict(result.message);
    else if (!result.ok) onError(result.message);
    return result.ok || !result.conflict;
  }, [saver, onConflict, onError]);

  useEffect(() => {
    const changed = () => {
      latest.current = model.data;
      onChange(model.data);
      clearTimeout(timer.current);
      timer.current = setTimeout(save, SAVE_DELAY_MS);
    };
    const completed = async () => {
      latest.current = model.data;
      onChange(model.data);
      if (await save()) onClose();
    };
    model.onValueChanged.add(changed);
    model.onComplete.add(completed);
    return () => {
      model.onValueChanged.remove(changed);
      model.onComplete.remove(completed);
    };
  }, [model, save, onChange, onClose]);

  // A save still waiting when the set closes (e.g. for the Reload message) mustn't run later.
  useEffect(() => () => clearTimeout(timer.current), []);

  const back = async () => {
    if (await save()) onClose();
  };

  return (
    <Stack>
      <Group justify="space-between">
        <Button variant="subtle" leftSection={<IconArrowLeft size={16} />} onClick={back}>All sections</Button>
        <Text size="sm" c={status === "unsaved" ? "red" : "dimmed"} data-testid="patient-save-status">
          {status === "saving" ? "Saving…" : status === "unsaved" ? "Not saved" : "Saved"}
        </Text>
      </Group>
      <Title order={3}>{set.name}</Title>
      <Card withBorder padding={0}>
        <Survey model={model} />
      </Card>
    </Stack>
  );
}
