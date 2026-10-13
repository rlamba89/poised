// The patient's links (A-4, PAT-04): an invite, /i/:token (S14), or a sign-in link, /s/:token
// (S15). The page shows nothing about the patient: first a Continue button (a POST, so an email
// scanner opening the link can't use it), which returns only the hospital's name for an invite,
// and nothing for a sign-in link; then their date of birth. The right date starts a session (an
// HTTP-only cookie) and opens their questionnaire (an invite) or their home (a sign-in link).
// Five wrong dates lock the link. A sign-in link works once, within 15 minutes.
//
// The steps are this page's state, not URLs: reloading starts again at Continue, and once the
// date of birth is right the next page replaces this one in the history, so Back never returns
// to the date-of-birth form.
import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";
import { Alert, Button, Card, Group, Stack, Text, TextInput, Title } from "@mantine/core";
import { ApiError, api } from "@/lib/api";
import { dobValue } from "./dob";
import { linkStopped, type LinkKind, type Stopped } from "./links";
import { PatientFrame } from "./PatientHQ";

type Step = { kind: "start" } | { kind: "dob"; hospital: string } | ({ kind: "stopped" } & Stopped);

export function InviteLink({ token, kind = "invite" }: { token: string; kind?: LinkKind }) {
  const navigate = useNavigate();
  const [step, setStep] = useState<Step>({ kind: "start" });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [day, setDay] = useState("");
  const [month, setMonth] = useState("");
  const [year, setYear] = useState("");
  // A sign-in link to a contact two people with the same date of birth use (twins, or cousins, on a
  // parent's phone): the server asks for the first name and surname too (409, code "name"; S15
  // answers 6 and 8). A wrong name counts as a wrong try.
  const [askName, setAskName] = useState(false);
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const base = kind === "invite" ? `/p/links/${token}` : `/p/signin/${token}`;

  const stopOr = (e: unknown) => {
    const s = linkStopped(e, kind);
    if (s) setStep({ kind: "stopped", ...s });
    else setError((e as Error).message); // e.g. a wrong date: the message says how many tries are left
  };

  const proceed = async () => {
    setBusy(true);
    setError("");
    try {
      const { hospital } = await api<{ hospital?: string }>(`${base}/continue`, { method: "POST" });
      setStep({ kind: "dob", hospital: hospital ?? "" });
    } catch (e) {
      stopOr(e);
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
      const { episodeId } = await api<{ episodeId?: string }>(`${base}/dob`, {
        method: "POST",
        body: JSON.stringify(askName ? { dateOfBirth, firstName, lastName } : { dateOfBirth }),
      });
      navigate(kind === "invite" && episodeId ? `/home/episodes/${episodeId}` : "/home", { replace: true });
    } catch (e) {
      if (e instanceof ApiError && (e.body as { code?: string } | null)?.code === "name") {
        setAskName(true);
        setError(e.message);
      } else {
        stopOr(e);
      }
      setBusy(false);
    }
  };

  if (step.kind === "stopped") {
    return (
      <PatientFrame>
        <Card withBorder p="xl" data-testid="link-stopped">
          <Title order={3}>{step.title}</Title>
          <Text mt="xs">{step.message}</Text>
          {step.newLink && <Button mt="md" component={Link} to="/signin">Get a new sign-in link</Button>}
        </Card>
      </PatientFrame>
    );
  }

  if (step.kind === "start") {
    return (
      <PatientFrame>
        <Card withBorder p="xl">
          <Stack>
            {kind === "invite" ? (
              <>
                <Title order={2}>Your health questionnaire</Title>
                <Text>Your hospital has asked you to fill in a health questionnaire before your operation.</Text>
              </>
            ) : (
              <>
                <Title order={2}>Sign in to Poised</Title>
                <Text>Tap Continue to see your health questionnaires.</Text>
              </>
            )}
            {error && <Alert color="red">{error}</Alert>}
            <Button size="md" onClick={proceed} loading={busy}>Continue</Button>
          </Stack>
        </Card>
      </PatientFrame>
    );
  }

  return (
    <PatientFrame hospital={step.hospital || undefined}>
      <Card withBorder p="xl">
        <form onSubmit={check} noValidate>
          <Stack>
            <Title order={2}>Confirm it&apos;s you</Title>
            <Text>{step.hospital || "Poised"} needs your date of birth to make sure it&apos;s you.</Text>
            <Text size="sm" c="dimmed" id="dob-hint">For example, 27 3 1985</Text>
            <Group gap="sm" wrap="nowrap" role="group" aria-label="Date of birth" aria-describedby="dob-hint">
              <TextInput label="Day" w={64} inputMode="numeric" autoComplete="bday-day" maxLength={2} value={day}
                onChange={(e) => setDay(e.currentTarget.value)} />
              <TextInput label="Month" w={64} inputMode="numeric" autoComplete="bday-month" maxLength={2} value={month}
                onChange={(e) => setMonth(e.currentTarget.value)} />
              <TextInput label="Year" w={96} inputMode="numeric" autoComplete="bday-year" maxLength={4} value={year}
                onChange={(e) => setYear(e.currentTarget.value)} />
            </Group>
            {askName && (
              <>
                {/* autoFocus: the box takes the focus as it appears, so the patient types straight into it */}
                <TextInput label="First name" autoComplete="given-name" maxLength={100} autoFocus value={firstName}
                  onChange={(e) => setFirstName(e.currentTarget.value)} />
                <TextInput label="Surname" autoComplete="family-name" maxLength={100} value={lastName}
                  onChange={(e) => setLastName(e.currentTarget.value)} />
              </>
            )}
            {error && <Alert color="red" data-testid="dob-error">{error}</Alert>}
            <Button size="md" type="submit" loading={busy}>Continue</Button>
          </Stack>
        </form>
      </Card>
    </PatientFrame>
  );
}
