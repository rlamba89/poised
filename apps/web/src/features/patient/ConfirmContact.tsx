// Confirming a new mobile or email (S15, PAT-10): the link sent to the new contact, /v/:token.
// Confirm (a POST, so a link scanner can't use it) makes it the patient's contact, for Preopsis and
// for every hospital that holds their record. No sign-in is needed: the link proves the new
// contact is theirs, and it may be opened on another device. It works once, within 24 hours.
import { useState } from "react";
import { Link } from "react-router";
import { Alert, Button, Card, Stack, Text, Title } from "@mantine/core";
import { ApiError, api } from "@/lib/api";
import { PatientFrame } from "./PatientHQ";

export function ConfirmContact({ token }: { token: string }) {
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState<"sms" | "email" | null>(null);
  const [stopped, setStopped] = useState<{ title: string; message: string } | null>(null);
  const [error, setError] = useState("");

  const confirm = async () => {
    setBusy(true);
    setError("");
    try {
      const { channel } = await api<{ channel: "sms" | "email" }>(`/p/contact/${token}/confirm`, { method: "POST" });
      setDone(channel);
    } catch (e) {
      const code = e instanceof ApiError ? (e.body as { code?: string } | null)?.code : undefined;
      if (code === "expired") setStopped({ title: "This link has expired", message: (e as Error).message });
      else if (e instanceof ApiError && e.status === 404) setStopped({ title: "This link doesn't work", message: e.message });
      else if (e instanceof ApiError && e.status === 409) setStopped({ title: "We couldn't change your details", message: e.message });
      else setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <PatientFrame>
      <Card withBorder p="xl" data-testid="confirm-contact">
        {done ? (
          <Stack>
            <Title order={2}>{done === "email" ? "Your email address has been updated" : "Your mobile number has been updated"}</Title>
            <Text>We&apos;ll use it from now on, and so will the hospitals that sent you questionnaires.</Text>
            <Button component={Link} to="/home">Go to your questionnaires</Button>
          </Stack>
        ) : stopped ? (
          <Stack>
            <Title order={3}>{stopped.title}</Title>
            <Text>{stopped.message}</Text>
            <Button variant="default" component={Link} to="/home">Go to your questionnaires</Button>
          </Stack>
        ) : (
          <Stack>
            <Title order={2}>Confirm your new contact details</Title>
            <Text>Tap Confirm to use this mobile number or email address for Preopsis.</Text>
            {error && <Alert color="red">{error}</Alert>}
            <Button size="md" onClick={confirm} loading={busy}>Confirm</Button>
          </Stack>
        )}
      </Card>
    </PatientFrame>
  );
}
