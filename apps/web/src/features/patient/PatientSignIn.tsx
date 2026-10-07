"use client";
// Patient sign-in (PAT-04/05): the link alone shows nothing. The patient taps Continue (so an
// email scanner that opens the link uses nothing up), then confirms their date of birth.
import { useState } from "react";
import { Alert, Button, Container, Group, Stack, Text, TextInput, Title } from "@mantine/core";
import { api, ApiError } from "@/lib/api";

type Step = "start" | "dob" | "stopped";

export function PatientSignIn({ token, onSignedIn }: { token: string; onSignedIn: () => void }) {
  const [step, setStep] = useState<Step>("start");
  const [day, setDay] = useState("");
  const [month, setMonth] = useState("");
  const [year, setYear] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  // A 404 (unknown link) or 403 (locked) can't be fixed by trying again here.
  const fail = (e: unknown) => {
    setError(e instanceof Error ? e.message : "Something went wrong. Please try again.");
    if (e instanceof ApiError && (e.status === 404 || e.status === 403)) setStep("stopped");
  };

  const start = async () => {
    setBusy(true);
    setError("");
    try {
      const res = await api<{ signedIn: boolean }>(`/p/links/${token}/continue`, { method: "POST" });
      if (res.signedIn) onSignedIn();
      else setStep("dob");
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  };

  const confirm = async (ev: React.FormEvent) => {
    ev.preventDefault();
    const d = Number(day), m = Number(month), y = Number(year);
    if (!Number.isInteger(d) || !Number.isInteger(m) || !Number.isInteger(y) || d < 1 || d > 31 || m < 1 || m > 12 || y < 1900 || y > 2100) {
      setError("Enter your date of birth as numbers, for example 27 3 1960.");
      return;
    }
    const dateOfBirth = `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
    setBusy(true);
    setError("");
    try {
      await api(`/p/links/${token}/dob`, { method: "POST", body: JSON.stringify({ dateOfBirth }) });
      onSignedIn();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Container size="xs" py="xl">
      <Stack>
        <Title order={2}>Your pre-operative health questionnaire</Title>
        {error && <Alert color="red">{error}</Alert>}
        {step === "start" && (
          <>
            <Text>Your hospital has asked you to answer some questions before your operation.</Text>
            <Button size="lg" onClick={start} loading={busy}>Continue</Button>
          </>
        )}
        {step === "dob" && (
          <form onSubmit={confirm}>
            <Stack>
              <Text>To keep your answers private, please enter your date of birth.</Text>
              <Group gap="sm" align="flex-end" wrap="nowrap">
                <TextInput label="Day" inputMode="numeric" maxLength={2} w={70} value={day} onChange={(e) => setDay(e.currentTarget.value)} />
                <TextInput label="Month" inputMode="numeric" maxLength={2} w={70} value={month} onChange={(e) => setMonth(e.currentTarget.value)} />
                <TextInput label="Year" inputMode="numeric" maxLength={4} w={100} value={year} onChange={(e) => setYear(e.currentTarget.value)} />
              </Group>
              <Button type="submit" size="lg" loading={busy}>Continue</Button>
            </Stack>
          </form>
        )}
      </Stack>
    </Container>
  );
}
