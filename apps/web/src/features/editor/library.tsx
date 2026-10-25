// The question library and "Same as…" (plan C4, session S24). Go owns the rules: what the library
// holds and who may change it (GET/POST/DELETE /api/o/:trust/library-questions), and which keys a
// question may take, with why not (POST …/questionnaires/:id/question-keys/search, keys.Compatible).
// These windows only list what Go sends and hand the chosen JSON back to the editor, which makes the
// change with insertFromLibrary or adoptKey (@poised/clinical, library.ts).
//   - LibraryWindow: "Add content → From library": search by title or tag, Insert, and Remove for
//     the trust's own questions.
//   - SaveToLibraryWindow: a question's "Save to library", with optional tags.
//   - SameAsWindow: a question's "Same as…": the trust's published questions and the library's,
//     with "Use this" only on the compatible ones; the others are greyed with Go's reason.
import { useEffect, useState } from "react";
import { Alert, Badge, Button, Group, Loader, Modal, ScrollArea, Stack, TagsInput, Text, TextInput } from "@mantine/core";
import { useDebouncedValue } from "@mantine/hooks";
import { IconBooks, IconSearch } from "@tabler/icons-react";
import { KIND_LABEL, kindOf, textOf, type ElementJson } from "@poised/clinical";
import { api } from "@/lib/api";

export type LibraryQuestion = {
  id: string;
  platform: boolean;
  key: string;
  title: string;
  element: ElementJson;
  tags: string[];
  createdByName: string;
};

export type Candidate = {
  key: string;
  title: string;
  kind: string;
  options: string[];
  sources: string[];
  library: boolean;
  compatible: boolean;
  reason: string;
  inThisHQ: boolean;
  /** One of its questions is Always ask fresh: adopting the key makes this one so too. */
  alwaysFresh: boolean;
  element: ElementJson;
};

type Losses = { disclosures: number; conditions: number };

/** What a library question is, for its row: "Select One", "Group of 2 questions"… */
function kindText(el: ElementJson): string {
  if (el.type === "panel") return `Group of ${(el.elements ?? []).length} questions`;
  const label = KIND_LABEL[kindOf(el)];
  return el.type === "paneldynamic" ? `${label} (repeatable)` : label;
}

/**
 * The library, to insert a question from. `onInsert` makes the change and returns a refusal to show
 * (the key is already in this HQ), or "" when it's done. `canRemove`: the user may remove the trust's
 * own questions (the same people who may insert).
 */
export function LibraryWindow(props: { opened: boolean; trustPath: string; onClose: () => void; onInsert: (q: LibraryQuestion) => string }) {
  const { opened, trustPath, onClose, onInsert } = props;
  const [search, setSearch] = useState("");
  const [debounced] = useDebouncedValue(search, 250);
  const [items, setItems] = useState<LibraryQuestion[] | null>(null);
  const [error, setError] = useState("");
  const [refusal, setRefusal] = useState<{ id: string; message: string } | null>(null);
  const [removing, setRemoving] = useState<LibraryQuestion | null>(null);
  const [asked, setAsked] = useState(0);
  // Each opening starts with an empty search (test report 1, found 2).
  const [wasOpen, setWasOpen] = useState(opened);
  if (opened !== wasOpen) {
    setWasOpen(opened);
    if (opened) setSearch("");
  }

  useEffect(() => {
    if (!opened) return;
    let live = true;
    api<LibraryQuestion[]>(`${trustPath}/library-questions?q=${encodeURIComponent(debounced.trim())}`)
      .then((l) => live && (setItems(l), setError("")))
      .catch((e: Error) => live && setError(e.message));
    return () => {
      live = false;
    };
  }, [opened, trustPath, debounced, asked]);

  const close = () => {
    setRefusal(null);
    setRemoving(null);
    onClose();
  };
  const remove = async (q: LibraryQuestion) => {
    try {
      await api(`${trustPath}/library-questions/${q.id}`, { method: "DELETE" });
      setRemoving(null);
      setAsked((n) => n + 1);
    } catch (e) {
      setError((e as Error).message);
      setRemoving(null);
    }
  };

  return (
    <Modal opened={opened} onClose={close} title="Insert from the question library" size="lg">
      <Stack gap="sm">
        <Text size="sm" c="dimmed">
          A library question keeps its ID wherever it&apos;s inserted, so a patient&apos;s answer to it in one questionnaire can fill it in for them in
          another.
        </Text>
        <TextInput
          label="Search"
          placeholder="Question text or tag"
          leftSection={<IconSearch size={14} />}
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
          data-autofocus
        />
        {error && <Alert color="red">{error}</Alert>}
        {!items && !error && <Loader size="sm" />}
        {items && items.length === 0 && <Text size="sm" c="dimmed">No library questions match.</Text>}
        <ScrollArea.Autosize mah={420}>
          <Stack gap={0}>
            {items?.map((q) => (
              <div key={q.id} data-testid={`library-${q.key}`} style={{ borderBottom: "1px solid var(--mantine-color-gray-2)", padding: "8px 0" }}>
                <Group justify="space-between" wrap="nowrap" align="flex-start">
                  <div style={{ minWidth: 0 }}>
                    <Text size="sm" fw={600}>{q.title || q.key}</Text>
                    <Group gap={6} mt={2}>
                      <Text size="xs" c="dimmed">{kindText(q.element)} · ID {q.key}</Text>
                      <Badge size="xs" variant="light" color={q.platform ? "gray" : "teal"}>{q.platform ? "Preopsis" : "This trust"}</Badge>
                      {q.tags.map((t) => (
                        <Badge key={t} size="xs" variant="outline" color="gray">{t}</Badge>
                      ))}
                    </Group>
                  </div>
                  <Group gap={6} wrap="nowrap">
                    {!q.platform && (
                      <Button size="xs" variant="subtle" color="red" onClick={() => setRemoving(q)}>
                        Remove
                      </Button>
                    )}
                    <Button
                      size="xs"
                      onClick={() => {
                        const refused = onInsert(q);
                        if (refused) setRefusal({ id: q.id, message: refused });
                        else close();
                      }}
                    >
                      Insert
                    </Button>
                  </Group>
                </Group>
                {refusal?.id === q.id && (
                  <Alert color="orange" mt={6} p="xs">
                    {refusal.message}
                  </Alert>
                )}
                {removing?.id === q.id && (
                  <Alert color="red" mt={6} p="xs" title="Remove from the library?">
                    <Text size="sm">Questionnaires it was inserted into keep their copy, with its ID.</Text>
                    <Group gap="xs" mt="xs">
                      <Button size="xs" color="red" onClick={() => remove(q)}>Remove</Button>
                      <Button size="xs" variant="default" onClick={() => setRemoving(null)}>Cancel</Button>
                    </Group>
                  </Alert>
                )}
              </div>
            ))}
          </Stack>
        </ScrollArea.Autosize>
      </Stack>
    </Modal>
  );
}

/** "Save to library": the selected question, with its ID and options, into the trust's library. */
export function SaveToLibraryWindow(props: { element: ElementJson | null; trustPath: string; onClose: () => void }) {
  const { element, trustPath, onClose } = props;
  const [tags, setTags] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);
  const [shown, setShown] = useState<ElementJson | null>(element);
  if (element && element !== shown) {
    setShown(element);
    setTags([]);
    setError("");
    setSaved(false);
  }
  const save = async () => {
    if (!element) return;
    setBusy(true);
    setError("");
    try {
      await api(`${trustPath}/library-questions`, { method: "POST", body: JSON.stringify({ element, tags }) });
      setSaved(true);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal opened={!!element} onClose={onClose} title="Save to the question library">
      <Stack gap="sm">
        <Text size="sm">
          “{textOf(shown?.title) || shown?.name}” <Text span size="xs" c="dimmed">(ID {shown?.name})</Text>
        </Text>
        {saved ? (
          <>
            <Alert color="teal">Saved. The trust&apos;s authors can now insert it into any questionnaire, keeping its ID.</Alert>
            <Group justify="flex-end">
              <Button onClick={onClose}>Done</Button>
            </Group>
          </>
        ) : (
          <>
            <Text size="sm" c="dimmed">
              It keeps its ID and its options wherever it&apos;s inserted, so a patient&apos;s answer can fill it in for them in another
              questionnaire. Its conditions aren&apos;t saved.
            </Text>
            <TagsInput label="Tags (optional)" description="Words to find it by, such as Lifestyle. Press Enter after each." value={tags} onChange={setTags} maxTags={10} />
            {error && <Alert color="red">{error}</Alert>}
            <Group justify="flex-end">
              <Button variant="default" onClick={onClose}>Cancel</Button>
              <Button leftSection={<IconBooks size={14} />} loading={busy} onClick={save}>Save to library</Button>
            </Group>
          </>
        )}
      </Stack>
    </Modal>
  );
}

/**
 * "Same as…": questions the trust has published, and the library's, that this question could be
 * linked to. `onAdopt` makes the change and returns a refusal to show, or "" when it's done.
 */
export function SameAsWindow(props: {
  element: ElementJson | null;
  searchPath: string;
  onClose: () => void;
  onAdopt: (c: Candidate) => string;
  /** What adopting would lose (adoptionLosses), said before Use this. */
  lossesFor: (c: Candidate) => Losses;
}) {
  const { element, searchPath, onClose, onAdopt, lossesFor } = props;
  const [search, setSearch] = useState("");
  const [debounced] = useDebouncedValue(search, 250);
  const [candidates, setCandidates] = useState<Candidate[] | null>(null);
  // The text of the question this one's key already links to (Go's linkedTo), or "" (S29).
  const [linkedTo, setLinkedTo] = useState("");
  const [error, setError] = useState("");
  const [shown, setShown] = useState<ElementJson | null>(null);
  // Each opening starts afresh, with an empty search (test report 1, found 2); the last question is
  // kept for the text while the window fades out.
  const [wasOpen, setWasOpen] = useState(false);
  if (!!element !== wasOpen) {
    setWasOpen(!!element);
    if (element) {
      setSearch("");
      setCandidates(null);
      setLinkedTo("");
      setError("");
    }
  }
  if (element && element !== shown) setShown(element);
  useEffect(() => {
    if (!element) return;
    let live = true;
    api<{ candidates: Candidate[]; linkedTo: string }>(searchPath, { method: "POST", body: JSON.stringify({ question: element, q: debounced.trim() }) })
      .then((r) => live && (setCandidates(r.candidates), setLinkedTo(r.linkedTo ?? ""), setError("")))
      .catch((e: Error) => live && setError(e.message));
    return () => {
      live = false;
    };
    // The question's JSON changes on every edit; the window asks again only when it opens (wasOpen) or the search changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [element?.name, wasOpen, searchPath, debounced]);

  return (
    <Modal opened={!!element} onClose={onClose} title="Same as…" size="lg">
      <Stack gap="sm">
        <Text size="sm" c="dimmed">
          Link “{textOf(shown?.title) || shown?.name}” to a question the trust already asks, so a patient&apos;s earlier answer to that one fills
          this one in. It takes that question&apos;s ID and options, and keeps its own wording.
        </Text>
        {/* Its key already comes from the library or a published HQ: Use this would change what it pre-fills from (S29). */}
        {linkedTo && (
          <Alert color="orange" p="xs" data-testid="same-as-linked">
            <Text size="sm">This question is already linked to “{linkedTo}”. Using this replaces that link.</Text>
          </Alert>
        )}
        <TextInput
          label="Search"
          placeholder="Question text"
          leftSection={<IconSearch size={14} />}
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
          data-autofocus
        />
        {error && <Alert color="red">{error}</Alert>}
        {!candidates && !error && <Loader size="sm" />}
        {candidates && candidates.length === 0 && <Text size="sm" c="dimmed">No published or library questions match.</Text>}
        <ScrollArea.Autosize mah={420}>
          <Stack gap={0}>
            {candidates?.map((c) => (
              <SameAsRow key={c.key} c={c} onAdopt={onAdopt} lossesFor={lossesFor} />
            ))}
          </Stack>
        </ScrollArea.Autosize>
      </Stack>
    </Modal>
  );
}

/** What adopting would lose, in a sentence, or "" (verification 1, finding 1). */
function lossText({ disclosures, conditions }: Losses): string {
  const parts = [
    disclosures > 0 && `Its own disclosures on ${disclosures} option${disclosures === 1 ? "" : "s"} will be removed.`,
    conditions > 0 &&
      `${conditions} condition${conditions === 1 ? " tests" : "s test"} its old answers and will need changing (Publish is refused until then).`,
  ].filter(Boolean);
  return parts.join(" ");
}

function SameAsRow({ c, onAdopt, lossesFor }: { c: Candidate; onAdopt: (c: Candidate) => string; lossesFor: (c: Candidate) => Losses }) {
  const [refused, setRefused] = useState("");
  const [warning, setWarning] = useState("");
  const adopt = () => {
    const r = onAdopt(c);
    if (r) setRefused(r);
  };
  return (
    <div data-testid={`candidate-${c.key}`} style={{ borderBottom: "1px solid var(--mantine-color-gray-2)", padding: "8px 0", opacity: c.compatible ? 1 : 0.55 }}>
      <Group justify="space-between" wrap="nowrap" align="flex-start">
        <div style={{ minWidth: 0 }}>
          <Text size="sm" fw={600}>{c.title}</Text>
          <Text size="xs" c="dimmed">
            {c.kind.charAt(0).toUpperCase() + c.kind.slice(1)}
            {c.options.length > 0 && `: ${c.options.join(" / ")}`} · ID {c.key}
          </Text>
          <Text size="xs" c="dimmed">In {c.sources.map((s) => (s === "Library" ? "the library" : s)).join(", ")}</Text>
          {c.alwaysFresh && (
            <Badge size="xs" variant="light" color="orange" mt={2} data-testid="candidate-fresh">
              Always ask fresh, like the original
            </Badge>
          )}
          {!c.compatible && (
            <Text size="xs" c="orange.8" data-testid="candidate-reason">
              {c.reason}
            </Text>
          )}
        </div>
        <Button
          size="xs"
          disabled={!c.compatible}
          onClick={() => {
            const loss = lossText(lossesFor(c));
            if (loss) setWarning(loss);
            else adopt();
          }}
        >
          Use this
        </Button>
      </Group>
      {warning && !refused && (
        <Alert color="orange" mt={6} p="xs" data-testid="candidate-losses">
          <Text size="sm">{warning}</Text>
          <Group gap="xs" mt="xs">
            <Button size="xs" color="orange" onClick={adopt}>Use this anyway</Button>
            <Button size="xs" variant="default" onClick={() => setWarning("")}>Cancel</Button>
          </Group>
        </Alert>
      )}
      {refused && (
        <Alert color="orange" mt={6} p="xs">
          {refused}
        </Alert>
      )}
    </div>
  );
}
