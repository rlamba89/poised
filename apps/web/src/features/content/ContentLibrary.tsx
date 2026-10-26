// The content library (plan C5, S33; INF-05, PX-13, A-6): videos (YouTube or Vimeo links) and
// leaflets (links to PDFs on the web until C8), with free tags, PX-13's checklist topics offered as
// ready-made tags, and the checklist showing which topics have nothing yet.
//   - AdminContentPage (/admin/content): Preopsis's own items, for platform admins, and the
//     platform's starter questions, which they retire or offer again.
//   - TrustContentPage (…/content): the trust's own items, changed by its whole-trust admins and
//     read by its whole-trust super clinicians, with Preopsis's active items read-only.
// Go checks every link, length and tag, and refuses with a plain message, shown under the form.
// Forms are inline, not modals: testers' browser tabs can hide Mantine's modals (decisions.md §4).
import { useCallback, useEffect, useRef, useState } from "react";
import { Alert, Anchor, Button, Card, Group, SegmentedControl, SimpleGrid, Stack, Table, TagsInput, Text, TextInput, Title } from "@mantine/core";
import { IconCheck, IconFileText, IconMovie, IconPlus } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { WholeBadge } from "@/lib/WholeBadge";
import { canManageContent, canSeeContent, useHospital } from "@/lib/auth";
import { addTag, bodyOf, changedFields, cleanTags, emptyForm, formOf, formatLength, type ContentForm, type ContentItem, type ContentKind, type ContentList, type StarterQuestion, type Topic } from "./content";

export function TrustContentPage() {
  const { trust } = useHospital();
  if (!canSeeContent(trust)) {
    return <Alert color="red" title="Only super clinicians and admins of the whole trust can see the content library" />;
  }
  return (
    <ContentLibrary
      path={`/o/${trust.id}/content`}
      canChange={canManageContent(trust)}
      intro={`${trust.name}'s videos and leaflets, for questionnaires and patients' information. Preopsis's own are listed below them, read-only.`}
      ownHeading={`${trust.name}'s items`}
      readOnlyNote={canManageContent(trust) ? "" : "Only the trust's admins can add or change items."}
    />
  );
}

export function AdminContentPage() {
  return (
    <Stack maw={1100}>
      <ContentLibrary
        path="/admin/content"
        canChange
        intro="Preopsis's videos and leaflets. Every trust sees the active ones, beside its own."
        ownHeading="Preopsis's items"
        readOnlyNote=""
        admin
      />
      <StarterQuestions />
    </Stack>
  );
}

function ContentLibrary({ path, canChange, intro, ownHeading, readOnlyNote, admin = false }: {
  path: string;
  admin?: boolean; // the platform admin's page: every item is Preopsis's own
  canChange: boolean;
  intro: string;
  ownHeading: string;
  readOnlyNote: string;
}) {
  const [list, setList] = useState<ContentList | null>(null);
  const [editing, setEditing] = useState<ContentItem | ContentKind | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  // A new message is at the top of the page: bring it into view after the list reloads (as Procedures).
  useEffect(() => {
    if (notice) window.scrollTo({ top: 0 });
  }, [notice, list]);

  const load = useCallback(() => {
    api<ContentList>(path).then(setList).catch((e: Error) => setError(e.message));
  }, [path]);
  useEffect(load, [load]);

  if (!list) return error ? <Alert color="red">{error}</Alert> : null;
  const own = list.items.filter((i) => admin || !i.platform);
  const platform = list.items.filter((i) => !admin && i.platform);

  const saved = (text: string) => {
    setEditing(null);
    setError("");
    setNotice(text);
    load();
  };
  const setStatus = async (c: ContentItem, status: ContentItem["status"]) => {
    setNotice("");
    try {
      await api(`${path}/${c.id}`, { method: "PATCH", body: JSON.stringify({ status }) });
      saved(status === "retired" ? `${c.title} is retired: it isn't offered any more.` : `${c.title} is offered again.`);
    } catch (e) {
      setError((e as Error).message);
    }
  };
  // Opening a form clears the last message, so Cancel doesn't bring an old error back (verification 1).
  const open = (e: ContentItem | ContentKind) => {
    setEditing(e);
    setNotice("");
    setError("");
  };

  return (
    <Stack maw={1100}>
      <Group justify="space-between" align="flex-start">
        <div>
          <Title order={3}>Content library</Title>
          <Text size="sm" c="dimmed">
            {intro} Videos are YouTube or Vimeo links; a leaflet is a link to a PDF on the web. A retired item isn&apos;t offered any more:
            questionnaires keep their own copy of a video, but it disappears from every patient&apos;s Information.
          </Text>
          {readOnlyNote && <Text size="sm" c="dimmed" mt={4}>{readOnlyNote}</Text>}
        </div>
        {canChange && editing === null && (
          <Group gap="xs" wrap="nowrap">
            <Button leftSection={<IconPlus size={16} />} onClick={() => open("video")}>Add video</Button>
            <Button leftSection={<IconPlus size={16} />} variant="light" onClick={() => open("leaflet")}>Add leaflet</Button>
          </Group>
        )}
      </Group>
      {notice && <Alert color="green">{notice}</Alert>}
      {error && editing === null && <Alert color="red">{error}</Alert>}
      {editing !== null && (
        <ItemForm
          key={typeof editing === "string" ? editing : editing.id}
          path={path}
          item={typeof editing === "string" ? null : editing}
          kind={typeof editing === "string" ? editing : editing.kind}
          topics={list.checklist}
          onSaved={saved}
          onCancel={() => setEditing(null)}
        />
      )}
      <Checklist topics={list.checklist} trust={!admin} />
      <Title order={4}>{ownHeading}</Title>
      <ItemTable label={ownHeading} items={own} empty="No items yet." actions={canChange ? (c) => (
        <Group gap="xs" justify="flex-end" wrap="nowrap">
          <Button size="xs" variant="default" disabled={editing !== null} onClick={() => open(c)}>Edit</Button>
          <Button size="xs" variant="default" color={c.status === "retired" ? undefined : "red"} disabled={editing !== null}
            onClick={() => setStatus(c, c.status === "retired" ? "active" : "retired")}>
            {c.status === "retired" ? "Bring back" : "Retire"}
          </Button>
        </Group>
      ) : undefined} />
      {!admin && (
        <>
          <Title order={4}>From Preopsis (read-only)</Title>
          <ItemTable label="From Preopsis" items={platform} empty="Preopsis has no items yet." />
        </>
      )}
    </Stack>
  );
}

/** PX-13's checklist: each topic, ticked when an active item has its tag. */
function Checklist({ topics, trust }: { topics: Topic[]; trust: boolean }) {
  const missing = topics.filter((t) => !t.covered).length;
  return (
    <Card withBorder data-testid="checklist">
      <Title order={5}>Checklist</Title>
      <Text size="sm" c="dimmed" mb="xs">
        What patients should be told before an operation. A topic is ticked once an active item has it as a tag
        {trust ? " (Preopsis's items count too)" : ""}. {missing === 0 ? "Every topic has an item." : `${missing} of ${topics.length} have nothing yet.`}
      </Text>
      <Group gap="xs">
        {topics.map((t) => (
          <WholeBadge key={t.topic} data-testid={`topic-${t.topic}`} data-covered={t.covered} variant={t.covered ? "light" : "outline"}
            color={t.covered ? "green" : "gray"} leftSection={t.covered ? <IconCheck size={12} /> : undefined}>
            {t.topic}: {t.covered ? "covered" : "nothing yet"}
          </WholeBadge>
        ))}
      </Group>
    </Card>
  );
}

function ItemTable({ label, items, empty, actions }: {
  label: string;
  items: ContentItem[];
  empty: string;
  actions?: (c: ContentItem) => React.ReactNode;
}) {
  if (items.length === 0) return <Text c="dimmed">{empty}</Text>;
  // At phone width the table scrolls sideways in its own box, wide enough for whole tags (test report 1, T-14).
  return (
    <Table.ScrollContainer minWidth={actions ? 820 : 680}>
      <Table verticalSpacing="sm" aria-label={label}>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Title</Table.Th>
            <Table.Th>Kind</Table.Th>
            <Table.Th>Length</Table.Th>
            <Table.Th>Tags</Table.Th>
            {actions && <Table.Th />}
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {items.map((c) => (
            <Table.Tr key={c.id} data-testid={`content-${c.title}`} c={c.status === "retired" ? "dimmed" : undefined}>
              <Table.Td>
                <Group gap="xs" wrap="nowrap">
                  <Text fw={600} size="sm">{c.title}</Text>
                  {c.status === "retired" && <WholeBadge color="gray" variant="light">Retired</WholeBadge>}
                </Group>
                <Anchor href={c.url} target="_blank" rel="noopener noreferrer" size="xs" style={{ wordBreak: "break-all" }}>{c.url}</Anchor>
              </Table.Td>
              <Table.Td>
                <Group gap={4} wrap="nowrap">
                  {c.kind === "video" ? <IconMovie size={14} /> : <IconFileText size={14} />}
                  <Text size="sm">{c.kind === "video" ? "Video" : "Leaflet"}</Text>
                </Group>
              </Table.Td>
              <Table.Td>{c.kind === "video" ? formatLength(c.lengthSeconds) : "—"}</Table.Td>
              <Table.Td>
                <Group gap={4}>
                  {c.tags.length === 0 ? "—" : c.tags.map((t) => <WholeBadge key={t} variant="light" color="gray">{t}</WholeBadge>)}
                </Group>
              </Table.Td>
              {actions && <Table.Td ta="right">{actions(c)}</Table.Td>}
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </Table.ScrollContainer>
  );
}

function ItemForm({ path, item, kind, topics, onSaved, onCancel }: {
  path: string;
  item: ContentItem | null; // null: a new one
  kind: ContentKind;
  topics: Topic[];
  onSaved: (notice: string) => void;
  onCancel: () => void;
}) {
  const [form, setForm] = useState<ContentForm>(item ? formOf(item) : emptyForm(kind));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  // Edit on a row further down opens the form at the top: bring it into view (as Procedures).
  const card = useRef<HTMLFormElement>(null);
  useEffect(() => {
    card.current?.scrollIntoView?.({ block: "nearest", behavior: "smooth" });
  }, []);
  const text = (key: "title" | "url" | "length") => ({
    value: form[key],
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: e.currentTarget.value }),
  });
  const video = form.kind === "video";

  const save = async (e?: React.FormEvent) => {
    e?.preventDefault(); // Enter in a field sends the form
    if (saving) return;
    setError("");
    setSaving(true);
    try {
      const body = bodyOf(form);
      if (item) {
        const changed = changedFields(bodyOf(formOf(item)), body);
        if (Object.keys(changed).length === 0) {
          onCancel();
          return;
        }
        await api(`${path}/${item.id}`, { method: "PATCH", body: JSON.stringify(changed) });
        onSaved(`${form.title.trim()} is saved.`);
      } else {
        await api(path, { method: "POST", body: JSON.stringify(body) });
        onSaved(`${form.title.trim()} is added.`);
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  // noValidate: the API's own messages, not the browser's.
  return (
    <Card withBorder data-testid="content-form" ref={card} component="form" onSubmit={save} noValidate>
      <Title order={4} mb="sm">{item ? `Edit ${item.title}` : video ? "Add a video" : "Add a leaflet"}</Title>
      {!item && (
        <SegmentedControl mb="sm" aria-label="Kind" value={form.kind} onChange={(v) => { setForm({ ...form, kind: v as ContentKind }); setError(""); }}
          data={[{ value: "video", label: "Video" }, { value: "leaflet", label: "Leaflet" }]} />
      )}
      <SimpleGrid cols={{ base: 1, sm: 2 }}>
        <TextInput label="Title" required {...text("title")} />
        <TextInput
          label={video ? "YouTube or Vimeo link" : "Link to the PDF"}
          description={video ? "For example https://www.youtube.com/watch?v=… or https://vimeo.com/…" : "A web address starting https://"}
          required
          // A web address field: a phone shows its URL keyboard and doesn't capitalise the first letter.
          type="url"
          autoCapitalize="none"
          autoCorrect="off"
          spellCheck={false}
          {...text("url")}
        />
        {video && <TextInput label="Length" description="Minutes and seconds, for example 3:20" required {...text("length")} />}
        {/* No suggestion list: its dropdown covers the form's buttons. The checklist topics are buttons below. */}
        {/* Enter adds the tag typed; with nothing typed it does nothing, rather than send the form (test report 1). */}
        <TagsInput label="Tags" description="Words to find it by. Press Enter after each." value={form.tags} maxTags={10}
          onChange={(tags) => setForm({ ...form, tags: cleanTags(tags) })}
          onKeyDown={(e) => { if (e.key === "Enter" && !e.currentTarget.value.trim()) e.preventDefault(); }} />
      </SimpleGrid>
      <Group gap={6} mt="xs" aria-label="Checklist topics to add">
        <Text size="xs" c="dimmed">Add a checklist topic as a tag:</Text>
        {topics.map((t) => (
          <Button key={t.topic} size="compact-xs" variant="default" disabled={form.tags.some((x) => x.toLowerCase() === t.topic.toLowerCase())}
            onClick={() => setForm({ ...form, tags: addTag(form.tags, t.topic) })}>
            + {t.topic}
          </Button>
        ))}
      </Group>
      {error && <Alert color="red" mt="sm">{error}</Alert>}
      <Group justify="flex-end" mt="md">
        <Button variant="default" onClick={onCancel}>Cancel</Button>
        <Button type="submit" loading={saving}>{item ? "Save changes" : video ? "Add video" : "Add leaflet"}</Button>
      </Group>
    </Card>
  );
}

/** The platform's starter questions (S33 answers 4): retire one, or offer it again. New ones come from the seed. */
function StarterQuestions() {
  const [list, setList] = useState<StarterQuestion[] | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const load = useCallback(() => {
    api<StarterQuestion[]>("/admin/library-questions").then(setList).catch((e: Error) => setError(e.message));
  }, []);
  useEffect(load, [load]);
  const setRetired = async (q: StarterQuestion, retired: boolean) => {
    setNotice("");
    setError("");
    try {
      await api(`/admin/library-questions/${q.id}`, { method: "PATCH", body: JSON.stringify({ retired }) });
      setNotice(retired
        ? `“${q.title}” is retired: Insert from library no longer offers it. Questionnaires that already have it keep it.`
        : `“${q.title}” is offered again.`);
      load();
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <Stack mt="lg" data-testid="starter-questions">
      <div>
        <Title order={3}>Starter questions</Title>
        <Text size="sm" c="dimmed">
          The question library every trust starts with. A retired question isn&apos;t offered by Insert from library any more; questionnaires
          that already have it keep it, with its ID. New starter questions are added by Preopsis&apos;s developers.
        </Text>
      </div>
      {notice && <Alert color="green">{notice}</Alert>}
      {error && <Alert color="red">{error}</Alert>}
      {list && (list.length === 0 ? <Text c="dimmed">No starter questions.</Text> : (
        <Table.ScrollContainer minWidth={760}>
          <Table verticalSpacing="sm" aria-label="Starter questions">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Question</Table.Th>
                <Table.Th>ID</Table.Th>
                <Table.Th>Tags</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {list.map((q) => (
                <Table.Tr key={q.id} data-testid={`starter-${q.key}`} c={q.retired ? "dimmed" : undefined}>
                  <Table.Td>
                    <Group gap="xs" wrap="nowrap">
                      <Text size="sm" fw={600}>{q.title || q.key}</Text>
                      {q.retired && <WholeBadge color="gray" variant="light">Retired</WholeBadge>}
                    </Group>
                  </Table.Td>
                  <Table.Td><Text size="xs" ff="monospace">{q.key}</Text></Table.Td>
                  <Table.Td>
                    <Group gap={4}>{q.tags.length === 0 ? "—" : q.tags.map((t) => <WholeBadge key={t} variant="light" color="gray">{t}</WholeBadge>)}</Group>
                  </Table.Td>
                  <Table.Td ta="right">
                    <Button size="xs" variant="default" color={q.retired ? undefined : "red"} onClick={() => setRetired(q, !q.retired)}>
                      {q.retired ? "Offer again" : "Retire"}
                    </Button>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      ))}
    </Stack>
  );
}
