// The scheduled-jobs page, /qa/jobs (S16): with MESSAGING=capture (local and QA) testers can't
// wait days for a reminder, so they can
//   - run the jobs now, as of a later time (POST /api/qa/jobs/run), e.g. two days on; and
//   - move the local timer's clock ahead (PUT /api/qa/jobs/clock), so its own next run (every
//     minute in make dev) acts as if that much time had passed.
// Like the captured messages: the API serves these only with MESSAGING=capture, and on QA only
// to a signed-in platform admin (S12; locally, with the dev login, they are open).
// Nothing here shows patient data: a run says only how many reminders and flags it made.
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { Link } from "react-router";
import { Alert, Anchor, Button, Card, Container, Group, Loader, NumberInput, Stack, Text, TextInput, Title } from "@mantine/core";
import { api } from "@/lib/api";
import { SIGN_IN_FIRST, qaError } from "./access";
import { formatDateTime } from "@/lib/format";
import { addDays, ceilMinute, fromLocalInput, offsetMinutes, toLocalInput } from "./jobsTime";

type Summary = { asOf: string; jobs: number; reminders: number; notResponding: number; failed: number };
type Clock = { timer: boolean; intervalSeconds: number; offsetMinutes: number; lastTickAt: string | null; lastTickAsOf: string | null };

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

/** "2 days 3 hours ahead", or "real time". */
function describeOffset(minutes: number): string {
  if (minutes === 0) return "real time";
  const days = Math.floor(minutes / (24 * 60));
  const hours = Math.floor((minutes % (24 * 60)) / 60);
  const parts = [days ? plural(days, "day", "days") : "", hours ? plural(hours, "hour", "hours") : ""].filter(Boolean);
  return `real time + ${parts.join(" ") || plural(minutes, "minute", "minutes")}`;
}

export function ScheduledJobs() {
  const [asOf, setAsOf] = useState(() => toLocalInput(ceilMinute(addDays(new Date(), 2))));
  const [running, setRunning] = useState(false);
  const [ran, setRan] = useState<{ ok: boolean; text: string } | null>(null);

  const [clock, setClock] = useState<Clock | null>(null);
  const [clockError, setClockError] = useState("");
  const [days, setDays] = useState<number | string>(0);
  const [hours, setHours] = useState<number | string>(0);
  const [saved, setSaved] = useState<{ ok: boolean; text: string } | null>(null);

  const loadClock = useCallback(
    () =>
      api<Clock>("/qa/jobs")
        .then((c) => {
          setClock(c);
          setClockError("");
        })
        .catch((e: unknown) => setClockError(qaError(e, "Jobs can be run from here only when the API runs with MESSAGING=capture."))),
    [],
  );
  useEffect(() => {
    loadClock();
    const id = setInterval(loadClock, 15_000); // the timer's last run moves on every minute
    return () => clearInterval(id);
  }, [loadClock]);

  const run = async (ev: FormEvent) => {
    ev.preventDefault();
    const when = fromLocalInput(asOf);
    if (!when) {
      setRan({ ok: false, text: "Enter a date and time." });
      return;
    }
    setRunning(true);
    try {
      const s = await api<Summary>("/qa/jobs/run", { method: "POST", body: JSON.stringify({ asOf: when }) });
      setRan({
        ok: s.failed === 0,
        text:
          `Ran ${plural(s.jobs, "job", "jobs")} as of ${formatDateTime(s.asOf)}: ` +
          `${plural(s.reminders, "reminder", "reminders")} sent, ${s.notResponding} marked "not responding"` +
          (s.failed ? `, ${s.failed} failed (see the API's log).` : "."),
      });
    } catch (e) {
      setRan({ ok: false, text: (e as Error).message });
    } finally {
      setRunning(false);
    }
  };

  const saveOffset = async (minutes: number | null) => {
    if (minutes === null) {
      setSaved({ ok: false, text: "Enter whole days and hours, 90 days at most." });
      return;
    }
    try {
      await api("/qa/jobs/clock", { method: "PUT", body: JSON.stringify({ offsetMinutes: minutes }) });
      setSaved({ ok: true, text: minutes ? `The timer now runs the jobs as of ${describeOffset(minutes)}, from its next run.` : "The timer is back on real time." });
      if (!minutes) {
        setDays(0);
        setHours(0);
      }
      loadClock();
    } catch (e) {
      setSaved({ ok: false, text: (e as Error).message });
    }
  };

  const preset = (n: number) => setAsOf(toLocalInput(ceilMinute(addDays(new Date(), n))));

  return (
    <Container size="md" py="lg">
      <Stack>
        <div>
          <Title order={2}>Scheduled jobs</Title>
          <Text c="dimmed" size="sm">
            Reminders go on the hospital&apos;s reminder days after the first invite (days 2 and 5 unless its settings say otherwise) until the
            patient starts answering, and an episode is marked &quot;Not responding&quot; on day 7 if the HQ hasn&apos;t been sent. Messages are
            captured, not sent: see <Anchor component={Link} to="/qa/messages">Captured messages</Anchor>.
          </Text>
        </div>

        <Card withBorder>
          <Title order={4} mb={4}>Run the jobs now, as of a later time</Title>
          <Text size="sm" c="dimmed" mb="xs">
            Runs every job that would be due at that time, for every hospital. Days count from each episode&apos;s first invite.
          </Text>
          {/* noValidate: a half-typed date would otherwise stop the form silently; run() says what's wrong. */}
          <form onSubmit={run} noValidate>
            <Stack gap="xs">
              <Group gap="xs">
                <Button size="xs" variant="default" onClick={() => preset(0)}>Now</Button>
                <Button size="xs" variant="default" onClick={() => preset(2)}>Now + 2 days</Button>
                <Button size="xs" variant="default" onClick={() => preset(5)}>Now + 5 days</Button>
                <Button size="xs" variant="default" onClick={() => preset(7)}>Now + 7 days</Button>
              </Group>
              <Group align="flex-end" wrap="wrap">
                <TextInput type="datetime-local" label="Run as of (your time zone)" value={asOf} onChange={(e) => setAsOf(e.currentTarget.value)} />
                <Button type="submit" loading={running}>Run jobs</Button>
              </Group>
              {ran && <Alert color={ran.ok ? "green" : "red"} data-testid="jobs-run-result">{ran.text}</Alert>}
            </Stack>
          </form>
        </Card>

        <Card withBorder>
          <Title order={4} mb={4}>The timer</Title>
          {clockError && (
            <Alert color="red">
              {clockError} {clockError === SIGN_IN_FIRST && <Anchor component={Link} to="/login">Sign in</Anchor>}
            </Alert>
          )}
          {!clock && !clockError && <Loader size="sm" />}
          {clock && (
            <Stack gap="xs">
              <Text size="sm" data-testid="jobs-timer">
                {clock.timer
                  ? `On: the local server runs the jobs every ${clock.intervalSeconds} seconds, as of ${describeOffset(clock.offsetMinutes)}.`
                  : "Off: the API was started with JOBS_TIMER=false, so jobs run only from this page."}
              </Text>
              <Text size="sm" c="dimmed" data-testid="jobs-last-tick">
                {clock.lastTickAt
                  ? `Last run ${formatDateTime(clock.lastTickAt)}, as of ${formatDateTime(clock.lastTickAsOf ?? clock.lastTickAt)}.`
                  : "It hasn't run yet."}
              </Text>
              <Text size="sm">
                Move the timer&apos;s clock ahead to see a reminder arrive by itself within a minute or two. Set it back afterwards (
                <code>make seed</code> does too), or new episodes get reminders straight away.
              </Text>
              <Group align="flex-end" wrap="wrap">
                <NumberInput label="Days ahead" min={0} max={90} clampBehavior="none" allowDecimal={false} value={days} onChange={setDays} w={120} />
                <NumberInput label="Hours ahead" min={0} max={23} clampBehavior="none" allowDecimal={false} value={hours} onChange={setHours} w={120} />
                <Button onClick={() => saveOffset(offsetMinutes(Number(days), Number(hours)))}>Move the clock</Button>
                <Button variant="default" onClick={() => saveOffset(0)} disabled={clock.offsetMinutes === 0}>Back to real time</Button>
              </Group>
              {saved && <Alert color={saved.ok ? "green" : "red"} data-testid="jobs-clock-result">{saved.text}</Alert>}
            </Stack>
          )}
        </Card>
      </Stack>
    </Container>
  );
}
