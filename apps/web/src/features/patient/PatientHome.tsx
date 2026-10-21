// The signed-in patient's home, /home (S15; PAT-06, PX-08): every questionnaire from every
// hospital, in every trust, that holds a record of them, newest first, each with its hospital's
// name and whether it's to do, in progress or sent. Their own contact details are shown in part
// (the last digits, the first letter and the domain), with Change (PAT-10): a link goes to the new
// mobile or email, and nothing changes until it's opened there.
//
// Close my account (S22, EPI-09): at the bottom, behind an inline confirmation that warns about
// any questionnaire not yet sent. The login goes; the hospitals keep their records.
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { Link } from "react-router";
import { Alert, Badge, Button, Card, Center, Group, Loader, Stack, Text, TextInput, Title } from "@mantine/core";
import { ApiError, api } from "@/lib/api";
import { formatShortDate } from "@/lib/format";
import { stateOf, unsentWarning, type EpisodeState } from "./home";
import { PatientFrame, SignedOut } from "./PatientHQ";

type HomeEpisode = { id: string; patientName: string; hospital: string; questionnaireName: string; procedure: string; createdAt: string; state: EpisodeState };
type Home = { firstName: string; episodes: HomeEpisode[]; contacts: { mobile: string; email: string } };

export function PatientHome() {
  const [home, setHome] = useState<Home | null>(null);
  const [error, setError] = useState("");
  const [ended, setEnded] = useState("");

  const load = useCallback(() => {
    api<Home>("/p/home")
      .then(setHome)
      .catch((e: Error) => (e instanceof ApiError && e.status === 401 ? setEnded(e.message) : setError(e.message)));
  }, []);
  useEffect(load, [load]);

  const signOut = async () => {
    await api("/p/logout", { method: "POST" }).catch(() => undefined);
    setEnded("You've signed out. To come back, open the link in your text message or email again.");
  };

  if (ended === CLOSED) return <PatientFrame><AccountClosed /></PatientFrame>;
  if (ended) return <PatientFrame><SignedOut message={ended} /></PatientFrame>;
  if (!home) {
    return (
      <PatientFrame>
        {error ? <Alert color="red">{error}</Alert> : <Center h="40vh"><Loader /></Center>}
      </PatientFrame>
    );
  }
  return (
    <PatientFrame onSignOut={signOut}>
      <Stack>
        <div>
          <Title order={2}>{home.firstName ? `Hello ${home.firstName}` : "Your health questionnaires"}</Title>
          <Text c="dimmed">Your health questionnaires from every hospital that has sent you one.</Text>
        </div>
        {home.episodes.length === 0 && (
          <Card withBorder p="lg"><Text>You have no health questionnaires at the moment.</Text></Card>
        )}
        {home.episodes.map((e) => {
          const s = stateOf(e.state);
          return (
            <Card key={e.id} withBorder data-testid="home-episode">
              <Stack gap={6}>
                <Group justify="space-between" wrap="nowrap" align="flex-start">
                  <Text fw={700}>{e.hospital}</Text>
                  <Badge color={s.color} variant="light" style={{ flexShrink: 0 }}>{s.label}</Badge>
                </Group>
                <Text>{e.questionnaireName}</Text>
                <Text size="sm">For {e.patientName}</Text>
                {e.procedure && <Text size="sm" c="dimmed">Procedure: {e.procedure}</Text>}
                {e.state === "cancelled" && (
                  <Text size="sm">The hospital has cancelled this, so there&apos;s nothing to fill in. Please contact them if you have any questions.</Text>
                )}
                {e.state === "closed" && (
                  <Text size="sm">The hospital has closed this, so there&apos;s nothing to fill in. Please contact them if you have any questions.</Text>
                )}
                <Group justify="space-between">
                  <Text size="sm" c="dimmed">Sent to you {formatShortDate(e.createdAt)}</Text>
                  {s.action && (
                    <Button component={Link} to={`/home/episodes/${e.id}`} variant={e.state === "sent" ? "default" : "filled"}>
                      {s.action}
                    </Button>
                  )}
                </Group>
              </Stack>
            </Card>
          );
        })}
        <Contacts contacts={home.contacts} />
        <CloseAccount episodes={home.episodes} onClosed={() => setEnded(CLOSED)} />
      </Stack>
    </PatientFrame>
  );
}

/** Their contact details, in part, with Change for each. */
function Contacts({ contacts }: { contacts: Home["contacts"] }) {
  return (
    <Card withBorder data-testid="home-contacts">
      <Title order={4}>Your contact details</Title>
      <Text size="sm" c="dimmed" mb="sm">We send your links and reminders here.</Text>
      <Stack gap="md">
        <ContactRow label="Mobile number" shown={contacts.mobile} email={false} />
        <ContactRow label="Email address" shown={contacts.email} email />
      </Stack>
      <Text size="xs" c="dimmed" mt="md">To change your name or date of birth, please contact your hospital.</Text>
    </Card>
  );
}

function ContactRow({ label, shown, email }: { label: string; shown: string; email: boolean }) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [sentTo, setSentTo] = useState("");

  const send = async (ev: FormEvent) => {
    ev.preventDefault();
    setBusy(true);
    setError("");
    try {
      // The box's kind goes too: the email box takes only an email address, the mobile box only a mobile.
      await api("/p/contact", { method: "POST", body: JSON.stringify({ contact: value, kind: email ? "email" : "mobile" }) });
      setSentTo(value.trim());
      setEditing(false);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const what = email ? "email address" : "mobile number";
  return (
    <div data-testid={email ? "contact-email" : "contact-mobile"}>
      <Group justify="space-between" wrap="nowrap">
        <div style={{ minWidth: 0 }}>
          <Text size="sm" fw={600}>{label}</Text>
          <Text size="sm" truncate>{shown || "Not given"}</Text>
        </div>
        {!editing && (
          <Button size="compact-sm" variant="default" onClick={() => { setEditing(true); setSentTo(""); setValue(""); }}>
            {shown ? "Change" : "Add"}
          </Button>
        )}
      </Group>
      {editing && (
        <form onSubmit={send} noValidate>
          <Stack gap="xs" mt="xs">
            <TextInput label={`New ${what}`} inputMode={email ? "email" : "tel"} autoComplete={email ? "email" : "tel"}
              value={value} onChange={(e) => setValue(e.currentTarget.value)} />
            {error && <Alert color="red" p="xs">{error}</Alert>}
            <Group gap="xs">
              <Button type="submit" size="sm" loading={busy}>Send a link to confirm</Button>
              <Button size="sm" variant="subtle" onClick={() => { setEditing(false); setError(""); }}>Cancel</Button>
            </Group>
          </Stack>
        </form>
      )}
      {sentTo && (
        <Alert color="green" mt="xs" p="xs" data-testid="contact-sent">
          {shown
            ? <>We&apos;ve sent a link to {sentTo}. Open it to finish the change. Until then, we&apos;ll keep using your current {what}.</>
            : <>We&apos;ve sent a link to {sentTo}. Open it to finish adding your {what}.</>}
        </Alert>
      )}
    </div>
  );
}

/** The `ended` value for a closed account: the page then says so, not "signed out". */
const CLOSED = "closed";

/**
 * Close my account (S22, EPI-09): an inline confirmation (no modal), which warns about any
 * questionnaire not yet sent. The server ends the session with it.
 */
function CloseAccount({ episodes, onClosed }: { episodes: HomeEpisode[]; onClosed: () => void }) {
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const warning = unsentWarning(episodes);
  const close = async () => {
    setBusy(true);
    setError("");
    try {
      await api("/p/account/close", { method: "POST" });
      onClosed();
    } catch (e) {
      setError((e as Error).message);
      setBusy(false);
    }
  };
  return (
    <Card withBorder data-testid="close-account">
      <Title order={4}>Close your account</Title>
      {!confirming ? (
        <>
          <Text size="sm" c="dimmed" mb="sm">Stop signing in to Preopsis. Your hospitals keep their records of you.</Text>
          <Button variant="light" color="red" onClick={() => setConfirming(true)} w="fit-content">Close my account</Button>
        </>
      ) : (
        <Stack gap="sm" mt="xs">
          <Text size="sm">
            If you close your account, you can&apos;t sign in any more, and your sign-in links stop working. Your hospitals keep their records
            of you and your questionnaires. If a hospital sends you a new questionnaire, its link starts a new account.
          </Text>
          {warning && <Alert color="orange" p="xs" data-testid="close-account-unsent">{warning}</Alert>}
          {error && <Alert color="red" p="xs">{error}</Alert>}
          <Group gap="xs">
            <Button color="red" onClick={close} loading={busy}>Close my account</Button>
            <Button variant="subtle" onClick={() => { setConfirming(false); setError(""); }} disabled={busy}>Keep my account</Button>
          </Group>
        </Stack>
      )}
    </Card>
  );
}

/** After closing: the account is gone, so there's nothing to sign in to. */
function AccountClosed() {
  return (
    <Card withBorder p="xl" data-testid="account-closed">
      <Title order={3}>Your account is closed</Title>
      <Text mt="xs">You&apos;ve been signed out, and you can&apos;t sign in with this account any more. Your hospitals keep their records of you.</Text>
      <Text mt="xs" size="sm" c="dimmed">If a hospital sends you a new questionnaire, open the link in its text message or email to start again.</Text>
    </Card>
  );
}
