"use client";
// Step 3 of plan-workflow.md: the patient fills in their HQ through their link (no sign-in).
// Question Sets are listed as tiles; each opens in the patient view, answers save as they go
// (PX-05), and "Send my answers" submits the whole HQ once every set is complete.
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Alert, Badge, Box, Button, Card, Center, Container, Group, Loader, Modal, Stack, Text, Title } from "@mantine/core";
import { IconArrowLeft, IconCircleCheck } from "@tabler/icons-react";
import { Model } from "survey-core";
import { Survey } from "survey-react-ui";
import "survey-core/survey-core.css";
import {
  isSetComplete, modelFor, patientVariables, registerClinicalProperties, shownSets, type Answers, type ChapterJson, type EpisodePatient,
} from "@sj/clinical";
import { api, ApiError } from "@/lib/api";
import { showUnits } from "@/features/preview/units";

registerClinicalProperties();

type QuestionSet = { id: string; name: string; description: string; icon: string; content: ChapterJson };
type HQ = {
  patient: EpisodePatient;
  questionnaireName: string;
  submitted: boolean;
  chapters: QuestionSet[];
  answers: Record<string, Answers>;
};

const SAVE_DELAY_MS = 600;

export default function PatientHQ({ token }: { token: string }) {
  const [hq, setHq] = useState<HQ | null>(null);
  const [error, setError] = useState("");
  const [openId, setOpenId] = useState<string | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [sending, setSending] = useState(false);

  useEffect(() => {
    api<HQ>(`/p/${token}`)
      .then(setHq)
      .catch((e: Error) => setError(e.message));
  }, [token]);

  const vars = useMemo(() => (hq ? patientVariables(hq.patient) : null), [hq]);

  const setAnswers = useCallback((id: string, data: Answers) => {
    setHq((h) => (h ? { ...h, answers: { ...h.answers, [id]: data } } : h));
  }, []);

  if (!hq || !vars) {
    return error ? <Alert color="red" m="md">{error}</Alert> : <Center h="60vh"><Loader /></Center>;
  }

  const sets = shownSets(hq.chapters, hq.answers, "patient", vars);
  const open = sets.find((s) => s.id === openId);
  // Building a survey per set is not free, so only while the list is showing.
  const done = new Set(open || hq.submitted ? [] : sets.filter((s) => isSetComplete(s.content, hq.answers[s.id], "patient", vars)).map((s) => s.id));

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
        {hq.submitted ? (
          <Card withBorder p="xl" ta="center">
            <IconCircleCheck size={48} color="var(--mantine-color-green-6)" style={{ margin: "0 auto" }} />
            <Title order={3} mt="sm">Thank you, {hq.patient.firstName}</Title>
            <Text mt="xs">Your answers have been sent to the hospital. A member of the team will review them.</Text>
          </Card>
        ) : open ? (
          <SetForm
            key={open.id}
            token={token}
            set={open}
            data={hq.answers[open.id] ?? {}}
            vars={vars}
            onChange={(data) => setAnswers(open.id, data)}
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
              const started = Object.keys(hq.answers[s.id] ?? {}).length > 0;
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
  token: string;
  set: QuestionSet;
  data: Answers;
  vars: ReturnType<typeof patientVariables>;
  onChange: (data: Answers) => void;
  onError: (message: string) => void;
  onClose: () => void;
};

/** One Question Set in the patient view. Answers save shortly after each change. */
function SetForm({ token, set, data, vars, onChange, onError, onClose }: SetFormProps) {
  const [saving, setSaving] = useState(false);
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

  const save = useCallback(async () => {
    clearTimeout(timer.current);
    setSaving(true);
    try {
      await api(`/p/${token}/answers/${set.id}`, { method: "PUT", body: JSON.stringify({ data: latest.current }) });
    } catch (e) {
      onError(e instanceof ApiError && e.status === 409 ? `${e.message} Reload the page to see them.` : (e as Error).message);
    } finally {
      setSaving(false);
    }
  }, [token, set.id, onError]);

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
      await save();
      onClose();
    };
    model.onValueChanged.add(changed);
    model.onComplete.add(completed);
    return () => {
      model.onValueChanged.remove(changed);
      model.onComplete.remove(completed);
    };
  }, [model, save, onChange, onClose]);

  const back = async () => {
    await save();
    onClose();
  };

  return (
    <Stack>
      <Group justify="space-between">
        <Button variant="subtle" leftSection={<IconArrowLeft size={16} />} onClick={back}>All sections</Button>
        <Text size="sm" c="dimmed" data-testid="patient-save-status">{saving ? "Saving…" : "Saved"}</Text>
      </Group>
      <Title order={3}>{set.name}</Title>
      <Card withBorder padding={0}>
        <Survey model={model} />
      </Card>
    </Stack>
  );
}
