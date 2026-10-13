// The captured-messages page, /qa/messages (S14): with MESSAGING=capture (local and QA), the
// patients' texts and emails are stored instead of sent, and this lists the newest 100
// (GET /api/qa/messages?to=). Testers open the links in them as a patient would. No sign-in: it
// exists only where nothing is really sent: the API serves it only with MESSAGING=capture, which
// a strict (AWS) deployment must set explicitly (QA only; S17 refuses it elsewhere).
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { Alert, Anchor, Badge, Button, Card, Container, Group, Loader, Stack, Text, TextInput, Title } from "@mantine/core";
import { ApiError, api } from "@/lib/api";
import { formatDateTime } from "@/lib/format";

type Message = { id: number; channel: "sms" | "email"; recipient: string; subject: string; body: string; sentAt: string };

/** The message's text with its web addresses as links. */
function Body({ text }: { text: string }) {
  return (
    <Text size="sm" style={{ whiteSpace: "pre-wrap", overflowWrap: "anywhere" }}>
      {text.split(/(https?:\/\/\S+)/).map((part, i) =>
        /^https?:\/\//.test(part) ? <Anchor key={i} href={part}>{part}</Anchor> : part,
      )}
    </Text>
  );
}

export function CapturedMessages() {
  const [to, setTo] = useState("");
  const [shown, setShown] = useState<Message[] | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [reset, setReset] = useState("");

  // Sign-in links and contact changes are limited per address and per network (S15); every
  // request from this machine is one network, so a tester clears the counts here to start again.
  const resetLimits = () =>
    api("/qa/rate-limits", { method: "DELETE" })
      .then(() => setReset("Sign-in limits cleared: links can be asked for again."))
      .catch((e: Error) => setReset(e.message));

  const load = useCallback((recipient: string) => {
    setLoading(true);
    api<Message[]>(`/qa/messages?to=${encodeURIComponent(recipient)}`)
      .then((m) => {
        setShown(m);
        setError("");
      })
      .catch((e: unknown) =>
        setError(e instanceof ApiError && e.status === 404 ? "Messages are captured only when the API runs with MESSAGING=capture." : (e as Error).message),
      )
      .finally(() => setLoading(false));
  }, []);
  useEffect(() => load(""), [load]);

  const search = (ev: FormEvent) => {
    ev.preventDefault();
    load(to.trim());
  };

  return (
    <Container size="md" py="lg">
      <Stack>
        <div>
          <Title order={2}>Captured messages</Title>
          <Text c="dimmed" size="sm">Texts and emails to patients are stored here instead of being sent (MESSAGING=capture). Newest first.</Text>
        </div>
        <form onSubmit={search}>
          <Group align="flex-end" wrap="nowrap">
            <TextInput label="To" placeholder="A mobile number (as stored, with its country code) or an email address" value={to} onChange={(e) => setTo(e.currentTarget.value)} style={{ flex: 1 }} />
            <Button type="submit" loading={loading}>Show</Button>
          </Group>
        </form>
        <Group gap="sm">
          <Button variant="default" size="xs" onClick={resetLimits}>Reset sign-in limits</Button>
          {reset && <Text size="sm" c="dimmed" data-testid="limits-reset">{reset}</Text>}
        </Group>
        {error && <Alert color="red">{error}</Alert>}
        {!shown && !error && <Loader />}
        {shown && shown.length === 0 && <Text c="dimmed">No messages{to.trim() ? " to that address" : ""} yet.</Text>}
        {shown?.map((m) => (
          <Card key={m.id} withBorder data-testid="captured-message">
            <Group justify="space-between" mb={4} wrap="wrap">
              <Group gap="xs">
                <Badge variant="light" color={m.channel === "sms" ? "teal" : "blue"}>{m.channel === "sms" ? "Text message" : "Email"}</Badge>
                <Text fw={600} size="sm">{m.recipient}</Text>
              </Group>
              <Text size="xs" c="dimmed">{formatDateTime(m.sentAt)}</Text>
            </Group>
            {m.subject && <Text size="sm" fw={600}>{m.subject}</Text>}
            <Body text={m.body} />
          </Card>
        ))}
      </Stack>
    </Container>
  );
}
