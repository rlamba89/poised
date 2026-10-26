// Step 3 of plan-workflow.md: the patient fills in their HQ, once their link and date of birth
// have started a session (S14). Staff use the same form to fill it in on the patient's behalf
// (PAT-09, `onBehalf`): only the API path and a few words differ.
// Question Sets are listed as tiles; each opens in the patient view, answers save as they go
// (PX-05), and "Send my answers" submits the whole HQ once every set is complete. Saves carry the
// set's revision: if the HQ is open on another device that saved since, the save gets a 409 and
// the page asks for a reload instead of overwriting (S02a). A 401 means the patient's session
// ended (30 minutes idle, a resend or a lock): the page says so, and the link in their message
// starts a new one. If a save or Send was what got refused, it also says what wasn't kept
// (`lostWork`): nothing may be lost without a word (S14 verification 1).
//
// Two copies of the answers, kept apart (S02a verify.md, verifications 1 and 2):
//   - what the server has: the answers loaded with the HQ, then each save it accepts (each set's
//     saver tracks them);
//   - what's on screen (drafts). A set saves whenever its draft differs from what the server has.
// A set counts as complete only when the server's answers are complete and its draft is saved
// (isSetReadyToSend), so a change whose save failed keeps "Send my answers" disabled.
//
// Pre-filled answers (S25): Go fills a set from the patient's earlier HQs when it's first opened.
// Such a set opens on a card ("We've filled in 23 answers…") with Check each answer and Everything
// is still correct; the button is recorded, and until one is pressed the set isn't complete. Each
// question still holding its pre-filled answer has a marker under it (prefill.ts has the rules).
//
// Answers from another trust (S26; S27: copied without asking, Rahul 24 Oct): the set's card names
// the hospital ("…from your questionnaire at Hospital B1 on 3 Mar 2026…") and adds Start fresh,
// which, once confirmed, has Go clear the set's pre-filled answers; the HQ then loads again, with
// the set empty. It's offered only until a button is pressed, and Go refuses it once the set changed.
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Alert, Anchor, Badge, Box, Button, Card, Center, Container, Group, Loader, Modal, Stack, Text, Title } from "@mantine/core";
import { Link, useNavigate } from "react-router";
import { IconArrowLeft, IconCircleCheck, IconHistory } from "@tabler/icons-react";

import { Model } from "survey-core";
import { Survey } from "survey-react-ui";
import "@/features/video/VideoQuestion"; // draws Video elements (S34)
import "@/features/information/InformationQuestion"; // draws Information, cleaned (S35)
import "survey-core/survey-core.css";
import {
  answerSaver, isSetReadyToSend, modelFor, registerClinicalProperties, sameAnswer, shownSets, type Answers, type AnswerSaver,
  type ChapterJson, type SamplePatient,
} from "@poised/clinical";
import { ApiError, api } from "@/lib/api";
import { Logo } from "@/features/brand/Logo";
import { lostWork, type Lost } from "./signedOut";
import { showUnits } from "@/features/preview/units";
import { formatDay } from "@/lib/format";
import {
  awaitsCheck, cardText, errorBannerShown, firstUnfilled, focusQuestion, offersStartFresh, showPrefillMarkers, shownPrefilled, type PrefillChoice, type SetPrefill,
} from "./prefill";

registerClinicalProperties();

type QuestionSet = { id: string; name: string; description: string; icon: string; content: ChapterJson };
/**
 * Who the HQ is for: the first name for "Hello Sam", and the survey's {patientName}, {patientAge}
 * and {patientSex}, worked out by the server. Never the date of birth (S14 answers 1).
 */
type FormPatient = { firstName: string; name: string; age: number; sex: string };
/** The HQ as loaded. Nothing changes it afterwards: later saves are tracked by each set's saver. */
type HQ = {
  patient: FormPatient;
  hospital: string;
  questionnaireName: string;
  submitted: boolean;
  /** The episode is archived (S28): its sent answers' View no longer says the team will review them. */
  archived?: boolean;
  chapters: QuestionSet[];
  answers: Record<string, Answers>;
  /** Each saved set's revision, which its first save sends back. A set with no answers yet has none. */
  revisions: Record<string, number>;
  /** The sets Go pre-filled, and where each answer came from (S25). */
  prefill?: Record<string, SetPrefill>;
};

const SAVE_DELAY_MS = 600;

type Props = {
  /** The form's API path: `/p/episodes/<id>` for the patient, `<hospital>/episodes/<id>/patient-form` for staff. */
  path: string;
  /** Staff filling it in for the patient (PAT-09): where to go back to, the episode's page. */
  onBehalf?: { back: string };
};

export default function PatientHQ({ path, onBehalf }: Props) {
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
  // The opening cards' buttons pressed since the HQ loaded (Go has recorded them).
  const [choices, setChoices] = useState<Record<string, PrefillChoice>>({});

  const [reloading, setReloading] = useState(false);
  // The open set's "save now": sends a change still waiting for its autosave (and a text box's
  // value not yet committed). Sign out and the on-behalf form's Episode link call it before they
  // leave, so the answer just typed is never dropped (S14 verification 2). True when nothing is
  // left unsaved.
  const flushRef = useRef<(() => Promise<boolean>) | null>(null);
  const navigate = useNavigate();
  // The patient's session ended (401): nothing more can be loaded or saved here. `lost` is what
  // the refused request was carrying.
  const [ended, setEnded] = useState<{ message: string; lost: Lost } | null>(null);
  // Start fresh cleared a set, but the HQ didn't load again (S28): the old answers mustn't stay on
  // screen as if they were still there, so the page asks for a reload instead.
  const [freshUnloaded, setFreshUnloaded] = useState(false);
  // A boolean, not the onBehalf object: a new object on each render would reload the form and
  // drop what staff were typing (S14 verification 1).
  const isPatient = !onBehalf;
  const endedBy = useCallback(
    (lost: Lost = {}) =>
      (e: unknown): never => {
        if (isPatient && e instanceof ApiError && e.status === 401) {
          setEnded((was) => ({ message: e.message, lost: { ...was?.lost, ...lost } }));
        }
        throw e;
      },
    [isPatient],
  );

  // Whether it loaded: on a failure the page shows why, and the caller may say more.
  const load = useCallback(() => {
    setReloading(true);
    return api<HQ>(path)
      .catch(endedBy())
      .then((h) => {
        savers.current = new Map();
        setDrafts({});
        setChoices({});
        setHq(h);
        setConflict("");
        setFreshUnloaded(false);
        setError(""); // an earlier failure's message no longer applies
        setLoads((n) => n + 1);
        return true;
      })
      .catch((e: Error) => {
        setError(e.message);
        return false;
      })
      .finally(() => setReloading(false));
  }, [path, endedBy]);
  useEffect(() => {
    load();
  }, [load]);

  const vars = useMemo<SamplePatient | null>(() => (hq ? { name: hq.patient.name, age: hq.patient.age, sex: hq.patient.sex } : null), [hq]);

  const setDraft = useCallback((id: string, data: Answers) => setDrafts((d) => ({ ...d, [id]: data })), []);

  // Staff leaving the on-behalf form: the answer just typed is saved first; if it can't be, they
  // stay, with the save's message showing.
  const leave = async (e: React.MouseEvent) => {
    if (!onBehalf || !flushRef.current) return;
    e.preventDefault();
    if (await flushRef.current()) navigate(onBehalf.back);
  };
  // The patient's way back to their home, every trust's questionnaires (S15). Like Sign out, it
  // saves the answer just typed first and waits for it (S15 verification 1), so the home shows
  // it; if the save is refused, the patient stays, with the save's message (a 401: "Not saved").
  const goHome = async (e: React.MouseEvent) => {
    if (!flushRef.current) return;
    e.preventDefault();
    if (await flushRef.current()) navigate("/home");
  };
  const homeLink = isPatient && (
    <Anchor component={Link} to="/home" size="sm" mb="md" display="inline-block" onClick={goHome}>
      <Group gap={4}><IconArrowLeft size={14} /> Your questionnaires</Group>
    </Anchor>
  );
  const backLink = onBehalf && (
    <Anchor component={Link} to={onBehalf.back} size="sm" onClick={leave} mb="md" display="inline-block">
      <Group gap={4}><IconArrowLeft size={14} /> Episode</Group>
    </Anchor>
  );

  if (ended) return <PatientFrame hospital={hq?.hospital}><SignedOut message={ended.message} lost={ended.lost} /></PatientFrame>;
  if (!hq || !vars) {
    const waiting = error ? <Alert color="red" m="md">{error}</Alert> : <Center h="60vh"><Loader /></Center>;
    if (isPatient) {
      // e.g. another person's episode address: "not found", or one the hospital cancelled (S20),
      // with the way back to their own (S15) and, once the refusal is in, Sign out (S20 test report 1).
      const signOutHere = async () => {
        await api("/p/logout", { method: "POST" }).catch(() => undefined);
        setEnded({ message: "You've signed out. To come back, open the link in your text message or email again.", lost: {} });
      };
      return <PatientFrame onSignOut={error ? signOutHere : undefined}>{homeLink}{waiting}</PatientFrame>;
    }
    return (
      <Box>
        {backLink}
        {waiting}
      </Box>
    );
  }

  const saverFor = (id: string) => {
    let saver = savers.current.get(id);
    if (!saver) {
      saver = answerSaver<Answers>(
        hq.revisions[id] ?? 0,
        (data, revision) =>
          api<{ revision: number }>(`${path}/answers/${id}`, { method: "PUT", body: JSON.stringify({ data, revision }) })
            .catch(endedBy({ unsavedSet: hq.chapters.find((c) => c.id === id)?.name ?? "a section" })),
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
  const prefillOf = (id: string): SetPrefill | undefined => {
    const p = hq.prefill?.[id];
    return p && choices[id] ? { ...p, choice: choices[id] } : p;
  };
  // Building a survey per set is not free, so only while the list is showing. A pre-filled set
  // waiting for its card's answer isn't complete yet.
  const listing = !open && !hq.submitted;
  const toCheck = new Set(listing ? sets.filter((s) => awaitsCheck(s.content, drafts[s.id] ?? saved[s.id], prefillOf(s.id), vars)).map((s) => s.id) : []);
  const done = new Set(
    listing ? sets.filter((s) => !toCheck.has(s.id) && isSetReadyToSend(s.content, saved[s.id], drafts[s.id], vars)).map((s) => s.id) : [],
  );
  // The card's button: Go records it, then the set goes on. Start fresh (S27) also has Go clear the
  // set, so the HQ loads again and the set opens empty (its saver then starts from the new revision).
  const choose = async (id: string, choice: PrefillChoice) => {
    try {
      await api(`${path}/prefill/${id}`, { method: "POST", body: JSON.stringify({ choice }) }).catch(endedBy());
      if (choice !== "start_fresh") {
        setChoices((c) => ({ ...c, [id]: choice }));
        return true;
      }
      if (await load()) return true;
      setFreshUnloaded(true); // the set was cleared: say so, rather than leave its card up (S27 verification 3)
      return false;
    } catch (e) {
      setError((e as Error).message);
      return false;
    }
  };

  // "Everything is still correct" with nothing left to answer: the next set shown, or the list.
  const openNext = (id: string) => setOpenId(sets[sets.findIndex((s) => s.id === id) + 1]?.id ?? null);

  const send = async () => {
    setSending(true);
    try {
      await api(`${path}/submit`, { method: "POST" }).catch(endedBy({ unsent: true }));
      setHq({ ...hq, submitted: true });
      setConfirming(false);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSending(false);
    }
  };

  const body = (
    <>
        {onBehalf && !hq.submitted && (
          <Alert color="blue" mb="md" data-testid="on-behalf-banner">
            You&apos;re filling in {hq.patient.name}&apos;s health questionnaire for them. The answers are saved as theirs, marked as entered by you.
          </Alert>
        )}
        {/* The cleared card already says the page couldn't load: no network banner over it (S29, S30). */}
        {errorBannerShown(error, freshUnloaded) && <Alert color="red" mb="md" withCloseButton onClose={() => setError("")}>{error}</Alert>}
        {freshUnloaded ? (
          <Card withBorder p="xl" data-testid="fresh-reload-failed">
            <Title order={3}>The answers in this section were cleared</Title>
            <Text mt="xs">The questionnaire couldn&apos;t be loaded again just now. Please try again.</Text>
            <Button mt="md" loading={reloading} onClick={load}>Reload</Button>
          </Card>
        ) : conflict ? (
          <Card withBorder p="xl" data-testid="patient-conflict">
            <Title order={3}>Your answers weren&apos;t saved</Title>
            <Text mt="xs">{conflict}</Text>
            <Button mt="md" loading={reloading} onClick={load}>Reload</Button>
          </Card>
        ) : hq.submitted && onBehalf ? (
          <Card withBorder p="xl" ta="center">
            <IconCircleCheck size={48} color="var(--mantine-color-green-6)" style={{ margin: "0 auto" }} />
            <Title order={3} mt="sm">Sent for {hq.patient.name}</Title>
            <Text mt="xs">The answers are with the team for review.</Text>
            <Button mt="md" component={Link} to={onBehalf.back}>Back to the episode</Button>
          </Card>
        ) : hq.submitted ? (
          <Card withBorder p="xl" ta="center">
            <IconCircleCheck size={48} color="var(--mantine-color-green-6)" style={{ margin: "0 auto" }} />
            <Title order={3} mt="sm">Thank you, {hq.patient.firstName}</Title>
            <Text mt="xs">
              Your answers have been sent to the hospital.{hq.archived ? "" : " A member of the team will review them."}
            </Text>
          </Card>
        ) : open ? (
          <SetForm
            key={`${open.id}-${loads}`}
            set={open}
            data={drafts[open.id] ?? saved[open.id] ?? {}}
            saver={saverFor(open.id)}
            vars={vars}
            prefill={prefillOf(open.id)}
            whose={onBehalf ? hq.patient.firstName : undefined}
            onChoose={(choice) => choose(open.id, choice)}
            onNextSet={() => openNext(open.id)}
            onChange={(data) => setDraft(open.id, data)}
            onConflict={setConflict}
            onError={setError}
            onClose={() => setOpenId(null)}
            flushRef={flushRef}
          />
        ) : (
          <Stack>
            <div>
              <Title order={2}>{hq.questionnaireName}</Title>
              <Text c="dimmed">
                {onBehalf
                  ? `For ${hq.patient.name}. Complete each section with them. Answers are saved as you go.`
                  : `Hello ${hq.patient.firstName}. Please complete each section. Your answers are saved as you go, so you can come back later.`}
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
                      ) : toCheck.has(s.id) ? (
                        <Badge color="brand" variant="light">Please check</Badge>
                      ) : (
                        started && <Badge color="orange" variant="light">In progress</Badge>
                      )}
                      <Button variant={done.has(s.id) ? "default" : "filled"} onClick={() => setOpenId(s.id)}>
                        {done.has(s.id) ? "Review" : toCheck.has(s.id) ? "Check" : started ? "Continue" : "Start"}
                      </Button>
                    </Group>
                  </Group>
                </Card>
              );
            })}
            <Group justify="flex-end">
              <Text size="sm" c="dimmed">{done.size} of {sets.length} sections complete</Text>
              <Button size="md" disabled={done.size < sets.length} onClick={() => setConfirming(true)}>
                {onBehalf ? "Send the answers" : "Send my answers"}
              </Button>
            </Group>
          </Stack>
        )}
      <Modal opened={confirming} onClose={() => setConfirming(false)} title={onBehalf ? "Send the answers?" : "Send your answers?"}>
        <Stack>
          <Text>
            {onBehalf
              ? "Once sent, the answers can't be changed here, and the episode moves to Ready for review."
              : "Once sent, you can't change your answers here. The hospital team will contact you if they need anything else."}
          </Text>
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setConfirming(false)}>Not yet</Button>
            <Button onClick={send} loading={sending}>{onBehalf ? "Send the answers" : "Send my answers"}</Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
  const signOut = async () => {
    const openSet = open?.name ?? "a section";
    const kept = flushRef.current ? await flushRef.current() : true; // the answer just typed goes first
    await api("/p/logout", { method: "POST" }).catch(() => undefined); // the cookie is gone either way once it expires
    setEnded((was) => ({
      message: "You've signed out. To come back, open the link in your text message or email again.",
      lost: { ...was?.lost, ...(kept ? {} : { unsavedSet: openSet }) },
    }));
  };
  if (onBehalf) {
    return (
      <Box>
        {backLink}
        {body}
      </Box>
    );
  }
  return <PatientFrame hospital={hq.hospital} onSignOut={signOut}>{homeLink}{body}</PatientFrame>;
}

/** The patient's pages: Preopsis's bar with their hospital's name, on a phone-sized column. */
export function PatientFrame({ hospital, onSignOut, children }: { hospital?: string; onSignOut?: () => void; children: React.ReactNode }) {
  return (
    <Box bg="gray.0" mih="100vh">
      <Group bg="brand.7" c="white" py="sm" px="md" justify="space-between" wrap="nowrap">
        <Text><Logo inverse /></Text>
        <Group gap="sm" wrap="nowrap" style={{ minWidth: 0 }}>
          {hospital && <Text size="sm" truncate>{hospital}</Text>}
          {onSignOut && <Button size="compact-sm" variant="white" onClick={onSignOut}>Sign out</Button>}
        </Group>
      </Group>
      <Container size="md" py="lg">{children}</Container>
    </Box>
  );
}

/**
 * The patient's session ended (30 minutes without a request, a resend, a lock, or signed out),
 * with what a refused save or Send didn't keep.
 */
export function SignedOut({ message, lost = {} }: { message: string; lost?: Lost }) {
  const notes = lostWork(lost);
  return (
    <Card withBorder p="xl" data-testid="patient-signed-out">
      <Title order={3}>You&apos;ve been signed out</Title>
      {notes.length > 0 && (
        <Alert color="orange" mt="sm" title="Not saved" data-testid="patient-lost-work">
          <Stack gap={4}>{notes.map((n) => <Text key={n} size="sm">{n}</Text>)}</Stack>
        </Alert>
      )}
      <Text mt="xs">{message}</Text>
      <Text mt="xs" size="sm" c="dimmed">You can also sign in with your mobile number or email address.</Text>
      <Button mt="md" variant="default" component={Link} to="/signin">Get a sign-in link</Button>
    </Card>
  );
}

type SetFormProps = {
  set: QuestionSet;
  /** What to show: the set's draft, else what the server has. */
  data: Answers;
  saver: AnswerSaver<Answers>;
  vars: SamplePatient;
  onChange: (data: Answers) => void;
  /** Called with the 409's message; the page then shows it with Reload instead of the set. */
  onConflict: (message: string) => void;
  onError: (message: string) => void;
  onClose: () => void;
  /** Set to this set's "save now" while it's open (see PatientHQ's flushRef). */
  flushRef: React.MutableRefObject<(() => Promise<boolean>) | null>;
  /** Where the set's pre-filled answers came from, if Go pre-filled it (S25). */
  prefill?: SetPrefill;
  /** The patient's first name when staff fill it in for them, for the card and the markers. */
  whose?: string;
  /** Records the opening card's button; false if it couldn't be (the page shows why). */
  onChoose: (choice: PrefillChoice) => Promise<boolean>;
  /** Opens the next set shown (or the list): "Everything is still correct" with nothing left here. */
  onNextSet: () => void;
};

/** One Question Set in the patient view. Answers save shortly after each change. */
function SetForm({ set, data, saver, vars, onChange, onConflict, onError, onClose, flushRef, prefill, whose, onChoose, onNextSet }: SetFormProps) {
  // "Not saved" while what's on screen differs from what the server has (e.g. after a failed save).
  const [status, setStatus] = useState<"saving" | "saved" | "unsaved">(() => (sameAnswer(data, saver.confirmed ?? {}) ? "saved" : "unsaved"));
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  // A change is waiting for its autosave (the timer is running).
  const pending = useRef(false);
  const latest = useRef(data);

  const model = useMemo(() => {
    const m: Model = modelFor(set.content, "patient", vars);
    showUnits(m);
    m.showPageTitles = false;
    m.completeText = "Finish this section";
    // A text answer counts as it's typed, not only when its box loses the focus, so "save now"
    // (Sign out, leaving the on-behalf form) always has it (S14 verification 2).
    m.textUpdateMode = "onTyping";
    m.data = data;
    // The marker (PRE-03) under each question still holding its pre-filled answer. Added before the
    // survey first draws, so no question is missed (as the clinician's corrections are).
    if (prefill) showPrefillMarkers(m, prefill, whose);
    return m;
    // Built once per set: answers then live in the model.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [set]);

  // The opening card (PRE-02): while pre-filled answers show and no button has been pressed.
  const [card, setCard] = useState(() => !prefill?.choice && shownPrefilled(model, prefill).length > 0);
  const [choosing, setChoosing] = useState(false);
  // Start fresh (S27) asks first: it clears the answers, and can't be undone.
  const [confirmFresh, setConfirmFresh] = useState(false);
  // The question "Everything is still correct" goes to, focused once the survey is showing.
  const [focusOn, setFocusOn] = useState<string | null>(null);
  const choose = async (choice: PrefillChoice) => {
    setChoosing(true);
    const ok = await onChoose(choice);
    setChoosing(false);
    if (!ok) {
      setConfirmFresh(false); // the page shows why (e.g. changed in another tab: reload)
      return;
    }
    if (choice === "start_fresh") return; // the HQ loads again, and the set with it
    if (choice === "check_each") {
      model.currentPageNo = 0;
      setCard(false);
      return;
    }
    const next = firstUnfilled(model, prefill);
    if (!next) {
      onNextSet();
      return;
    }
    model.currentPage = next.page;
    setFocusOn(next.name);
    setCard(false);
  };
  useEffect(() => {
    if (card || !focusOn) return;
    let stop = () => {};
    const t = setTimeout(() => (stop = focusQuestion(model, focusOn)));
    return () => {
      clearTimeout(t);
      stop();
    };
  }, [card, focusOn, model]);


  /** Saves the latest answers now; whether the server has them, and whether it refused them as stale. */
  const persist = useCallback(async () => {
    clearTimeout(timer.current);
    pending.current = false;
    // Nothing to save if the server already has these answers: a tab left open elsewhere mustn't
    // overwrite newer answers, or be told it can't. A save that failed is sent again here.
    const answers = latest.current;
    if (sameAnswer(answers, saver.confirmed ?? {})) {
      setStatus("saved");
      return { ok: true, conflict: false };
    }
    setStatus("saving");
    const result = await saver.save(answers);
    setStatus(result.ok ? "saved" : "unsaved");
    if (!result.ok && result.conflict) onConflict(result.message);
    else if (!result.ok) onError(result.message);
    return { ok: result.ok, conflict: !result.ok && result.conflict };
  }, [saver, onConflict, onError]);

  /** Saves the latest answers; false if the answers changed on another device. */
  const save = useCallback(async () => {
    const r = await persist();
    return r.ok || !r.conflict;
  }, [persist]);

  // "Save now" for the page (Sign out, the on-behalf Episode link): a text box commits its value
  // only when it loses the focus, so take the focus off first.
  useEffect(() => {
    flushRef.current = async () => {
      if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
      latest.current = model.data;
      return (await persist()).ok;
    };
    return () => {
      flushRef.current = null;
    };
  }, [flushRef, model, persist]);

  useEffect(() => {
    const changed = () => {
      latest.current = model.data;
      onChange(model.data);
      clearTimeout(timer.current);
      pending.current = true;
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

  // Leaving the set by an in-app change (a link, a route change) sends a save still waiting rather
  // than dropping it (S14 verification 2). After a 409 the saver refuses it without a request, so
  // the Reload message's set doesn't overwrite newer answers; a 401 shows the "Not saved" message.
  // This cleanup does NOT run when the page itself unloads (the patient's browser Back, a reload,
  // closing the tab): a change from the last ~0.6 s is then lost (S14 test report 4, X-1; in
  // pending.md).
  useEffect(
    () => () => {
      clearTimeout(timer.current);
      if (pending.current) void saver.save(latest.current);
    },
    [saver],
  );

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
      {card && prefill ? (
        <Card withBorder p="lg" data-testid="prefill-card">
          <Group gap="xs" wrap="nowrap">
            <IconHistory size={20} color="var(--mantine-color-brand-7)" />
            <Text fw={700}>Your earlier answers</Text>
          </Group>
          <Text mt="xs">{cardText(shownPrefilled(model, prefill).map((q) => prefill.answers[q.name]), formatDay, whose)}</Text>
          {confirmFresh ? (
            <Stack mt="md" gap="sm" data-testid="start-fresh-confirm">
              <Text size="sm" fw={600}>
                Clear these answers? You&apos;ll answer every question in this section again{whose ? ` with ${whose}` : ""}. This can&apos;t be undone.
              </Text>
              <Button color="red" onClick={() => choose("start_fresh")} loading={choosing}>Yes, start fresh</Button>
              <Button variant="default" onClick={() => setConfirmFresh(false)} disabled={choosing}>Keep the answers</Button>
            </Stack>
          ) : (
            <Stack mt="md" gap="sm">
              <Button onClick={() => choose("check_each")} loading={choosing}>Check each answer</Button>
              <Button variant="light" onClick={() => choose("still_correct")} disabled={choosing}>Everything is still correct</Button>
              {offersStartFresh(prefill, shownPrefilled(model, prefill).map((q) => prefill.answers[q.name])) && (
                <Button variant="subtle" onClick={() => setConfirmFresh(true)} disabled={choosing}>Start fresh</Button>
              )}
            </Stack>
          )}
        </Card>
      ) : (
        <Card withBorder padding={0}>
          <Survey model={model} />
        </Card>
      )}
    </Stack>
  );
}
