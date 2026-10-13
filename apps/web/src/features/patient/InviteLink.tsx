// The patient's invite link, /i/:token (S14; A-4, PAT-04). The page shows nothing about the
// patient: first a Continue button (a POST, so an email scanner opening the link can't use it),
// which returns only the hospital's name; then their date of birth. The right date starts a
// session (an HTTP-only cookie) and opens their questionnaire; five wrong dates lock the link.
//
// The steps are this page's state, not URLs: reloading starts again at Continue, and once the
// date of birth is right the questionnaire replaces this page in the history, so Back never
// returns to the date-of-birth form.
import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router";
import { Alert, Button, Card, Group, Stack, Text, TextInput, Title } from "@mantine/core";
import { ApiError, api } from "@/lib/api";
import { dobValue } from "./dob";
import { PatientFrame } from "./PatientHQ";

type Step = { kind: "start" } | { kind: "dob"; hospital: string } | { kind: "stopped"; title: string; message: string };

/** A refusal the patient can do nothing about here: a dead link, a locked one, or a suspended hospital. */
function stopped(e: unknown): Step | null {
  if (!(e instanceof ApiError)) return null;
  const locked = (e.body as { code?: string } | null)?.code === "locked";
  if (locked) return { kind: "stopped", title: "This link is locked", message: e.message };
  if (e.status === 404) return { kind: "stopped", title: "This link doesn't work", message: e.message };
  if (e.status === 403) return { kind: "stopped", title: "Not available at the moment", message: e.message };
  return null;
}

export function InviteLink({ token }: { token: string }) {
  const navigate = useNavigate();
  const [step, setStep] = useState<Step>({ kind: "start" });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [day, setDay] = useState("");
  const [month, setMonth] = useState("");
  const [year, setYear] = useState("");

  const proceed = async () => {
    setBusy(true);
    setError("");
    try {
      const { hospital } = await api<{ hospital: string }>(`/p/links/${token}/continue`, { method: "POST" });
      setStep({ kind: "dob", hospital });
    } catch (e) {
      const s = stopped(e);
      if (s) setStep(s);
      else setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const check = async (ev: FormEvent) => {
    ev.preventDefault();
    const dateOfBirth = dobValue(day, month, year);
    if (!dateOfBirth) {
      setError("Enter your date of birth as numbers: the day, the month and the year, for example 27 3 1985.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const { episodeId } = await api<{ episodeId: string }>(`/p/links/${token}/dob`, {
        method: "POST",
        body: JSON.stringify({ dateOfBirth }),
      });
      navigate(`/home/episodes/${episodeId}`, { replace: true });
    } catch (e) {
      const s = stopped(e);
      if (s) setStep(s);
      else setError((e as Error).message); // a wrong date: the message says how many tries are left
      setBusy(false);
    }
  };

  if (step.kind === "stopped") {
    return (
      <PatientFrame>
        <Card withBorder p="xl" data-testid="link-stopped">
          <Title order={3}>{step.title}</Title>
          <Text mt="xs">{step.message}</Text>
        </Card>
      </PatientFrame>
    );
  }

  if (step.kind === "start") {
    return (
      <PatientFrame>
        <Card withBorder p="xl">
          <Stack>
            <Title order={2}>Your health questionnaire</Title>
            <Text>Your hospital has asked you to fill in a health questionnaire before your operation.</Text>
            {error && <Alert color="red">{error}</Alert>}
            <Button size="md" onClick={proceed} loading={busy}>Continue</Button>
          </Stack>
        </Card>
      </PatientFrame>
    );
  }

  return (
    <PatientFrame hospital={step.hospital}>
      <Card withBorder p="xl">
        <form onSubmit={check} noValidate>
          <Stack>
            <Title order={2}>Confirm it&apos;s you</Title>
            <Text>{step.hospital} needs your date of birth to make sure it&apos;s you.</Text>
            <Text size="sm" c="dimmed" id="dob-hint">For example, 27 3 1985</Text>
            <Group gap="sm" wrap="nowrap" role="group" aria-label="Date of birth" aria-describedby="dob-hint">
              <TextInput label="Day" w={64} inputMode="numeric" autoComplete="bday-day" maxLength={2} value={day}
                onChange={(e) => setDay(e.currentTarget.value)} />
              <TextInput label="Month" w={64} inputMode="numeric" autoComplete="bday-month" maxLength={2} value={month}
                onChange={(e) => setMonth(e.currentTarget.value)} />
              <TextInput label="Year" w={96} inputMode="numeric" autoComplete="bday-year" maxLength={4} value={year}
                onChange={(e) => setYear(e.currentTarget.value)} />
            </Group>
            {error && <Alert color="red" data-testid="dob-error">{error}</Alert>}
            <Button size="md" type="submit" loading={busy}>Continue</Button>
          </Stack>
        </form>
      </Card>
    </PatientFrame>
  );
}
