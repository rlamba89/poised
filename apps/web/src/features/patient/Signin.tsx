// A returning patient's sign-in (S15, PAT-06): they type their mobile or email and get a sign-in
// link (/s/:token) by text or email, which works once, within 15 minutes. The answer is the same
// whether or not that contact has an account, so this page never says who has one. A mobile
// outside the deployment's country is refused (PAT-11), and so are too many requests (3 per
// contact per 15 minutes, 10 an hour from one network): the server's message says which.
import { useState, type FormEvent } from "react";
import { Alert, Button, Card, Stack, Text, TextInput, Title } from "@mantine/core";
import { api } from "@/lib/api";
import { PatientFrame } from "./PatientHQ";

export function Signin() {
  const [contact, setContact] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [sent, setSent] = useState<"sms" | "email" | null>(null);

  const send = async (ev: FormEvent) => {
    ev.preventDefault();
    setBusy(true);
    setError("");
    try {
      const { channel } = await api<{ channel: "sms" | "email" }>("/p/signin", { method: "POST", body: JSON.stringify({ contact }) });
      setSent(channel);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  if (sent) {
    return (
      <PatientFrame>
        <Card withBorder p="xl" data-testid="signin-sent">
          <Stack>
            <Title order={2}>{sent === "email" ? "Check your email" : "Check your text messages"}</Title>
            <Text>
              If you use Preopsis with {sent === "email" ? "that email address" : "that mobile number"}, we&apos;ve sent you a link to sign in.
              It works once, within 15 minutes.
            </Text>
            <Text size="sm" c="dimmed">
              If you haven&apos;t used Preopsis before, please use the link in the text message or email your hospital sent you.
            </Text>
            <Button variant="default" onClick={() => setSent(null)}>Send another link</Button>
          </Stack>
        </Card>
      </PatientFrame>
    );
  }

  return (
    <PatientFrame>
      <Card withBorder p="xl">
        <form onSubmit={send} noValidate>
          <Stack>
            <Title order={2}>Sign in</Title>
            <Text>Enter your mobile number or email address. We&apos;ll send you a link to sign in and see your health questionnaires.</Text>
            <TextInput label="Mobile number or email address" autoComplete="username" value={contact}
              onChange={(e) => setContact(e.currentTarget.value)} />
            {error && <Alert color="red" data-testid="signin-error">{error}</Alert>}
            <Button size="md" type="submit" loading={busy}>Send me a link</Button>
          </Stack>
        </form>
      </Card>
    </PatientFrame>
  );
}
