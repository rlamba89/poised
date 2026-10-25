// A Video element's settings (S34, INF-01/02): its title, its link (pasted, or chosen from the
// content library), its length, and the "I have watched this video" tick, which can be required.
// The rules are in @poised/clinical (video.ts): the link check (the same cases as Go's), and the
// library choice, which copies the link, title and length into the questionnaire (S34 answers 1).
import { useEffect, useState } from "react";
import { Alert, Button, Group, Loader, Modal, ScrollArea, Stack, Switch, Text, TextInput } from "@mantine/core";
import { useDebouncedValue } from "@mantine/hooks";
import { IconBooks, IconSearch } from "@tabler/icons-react";
import {
  checkVideo, chooseLibraryVideo, formatLength, parseLength, providerName, textOf, tickPatch, MAX_VIDEO_LENGTH,
  type ChapterJson, type ElementJson,
} from "@poised/clinical";
import { api } from "@/lib/api";
import { searchPath, tagsOf, videosOf, type Found } from "@/features/video/picker";
import { BlurText, BlurTextarea } from "./inputs";

export function VideoSettings(props: {
  el: ElementJson;
  readOnly: boolean;
  patch: (p: Partial<ElementJson>) => void;
  change: (edit: (doc: ChapterJson) => ChapterJson) => void;
  /** The trust's API path, when the user may search the content library (whole-trust super clinicians and admins). */
  libraryPath?: string;
}) {
  const { el, readOnly, patch, change, libraryPath } = props;
  const [picking, setPicking] = useState(false);
  const link = typeof el.videoUrl === "string" ? el.videoUrl : "";
  const video = link ? checkVideo(link) : null;
  const seconds = typeof el.videoLength === "number" ? el.videoLength : undefined;
  // What was typed in Length, kept while it can't be read, so the author sees what's wrong.
  const [lengthDraft, setLengthDraft] = useState<string | null>(null);
  useEffect(() => setLengthDraft(null), [el.name, seconds]);
  const lengthError =
    lengthDraft !== null ? "Type the length as minutes and seconds, for example 3:20, or 1:02:05 (4 hours at most)." : !seconds ? "Add the video's length." : undefined;

  return (
    <>
      <BlurTextarea label="Title" value={textOf(el.title)} disabled={readOnly} onSave={(title) => patch({ title })} />
      <BlurText
        label="Link"
        description="A YouTube or Vimeo link, for example https://www.youtube.com/watch?v=… or https://vimeo.com/…"
        type="url"
        autoCapitalize="none"
        value={link}
        disabled={readOnly}
        error={!link ? "Paste the video's link, or choose one from the library." : !video ? "Only YouTube or Vimeo links can be used." : undefined}
        onSave={(v) => patch({ videoUrl: v.trim() || undefined })}
      />
      {video && (
        <Text size="sm" c="dimmed" data-testid="video-plays">
          Plays from {providerName(video)}{video.provider === "youtube" ? " (privacy-enhanced mode)" : " (no tracking)"}
        </Text>
      )}
      {!readOnly && libraryPath && (
        <Group>
          <Button size="xs" variant="light" leftSection={<IconBooks size={14} />} onClick={() => setPicking(true)}>
            Choose from library
          </Button>
        </Group>
      )}
      <BlurText
        label="Length"
        placeholder="m:ss"
        value={lengthDraft ?? formatLength(seconds)}
        disabled={readOnly}
        error={lengthError}
        onSave={(v) => {
          if (!v.trim()) return patch({ videoLength: undefined });
          const s = parseLength(v);
          if (!s || s > MAX_VIDEO_LENGTH) return setLengthDraft(v);
          setLengthDraft(null);
          patch({ videoLength: s });
        }}
      />
      <Switch
        label="“I have watched this video” tick"
        description="The patient ticks it under the video. Whether they really watched it isn't tracked."
        checked={!!el.watchTick}
        disabled={readOnly}
        onChange={(e) => patch(tickPatch(e.currentTarget.checked))}
      />
      {!!el.watchTick && (
        <Switch
          label="Required"
          description="The patient must tick it before this Question Set is complete."
          checked={!!el.isRequired}
          disabled={readOnly}
          onChange={(e) => patch({ isRequired: e.currentTarget.checked || undefined })}
        />
      )}
      {libraryPath && (
        <VideoPicker
          opened={picking}
          trustPath={libraryPath}
          onClose={() => setPicking(false)}
          onPick={(item) => {
            change((d) => chooseLibraryVideo(d, el.name, item));
            setPicking(false);
          }}
        />
      )}
    </>
  );
}

/** The content library's videos, searched by title and tag; Use this copies one into the question. */
function VideoPicker(props: { opened: boolean; trustPath: string; onClose: () => void; onPick: (item: Found) => void }) {
  const { opened, trustPath, onClose, onPick } = props;
  const [search, setSearch] = useState("");
  const [debounced] = useDebouncedValue(search, 250);
  const [tag, setTag] = useState<string | null>(null);
  const [tags, setTags] = useState<string[]>([]);
  const [items, setItems] = useState<Found[] | null>(null);
  const [error, setError] = useState("");
  // Each opening starts afresh, with the tags of every video on offer.
  useEffect(() => {
    if (!opened) return;
    setSearch("");
    setTag(null);
    let live = true;
    api<Found[]>(searchPath(trustPath, "", null))
      .then((l) => live && setTags(tagsOf(l)))
      .catch((e: Error) => live && setError(e.message));
    return () => {
      live = false;
    };
  }, [opened, trustPath]);
  useEffect(() => {
    if (!opened) return;
    let live = true;
    api<Found[]>(searchPath(trustPath, debounced, tag))
      .then((l) => live && (setItems(videosOf(l)), setError("")))
      .catch((e: Error) => live && setError(e.message));
    return () => {
      live = false;
    };
  }, [opened, trustPath, debounced, tag]);

  return (
    <Modal opened={opened} onClose={onClose} title="Choose a video from the library" size="lg">
      <Stack>
        <TextInput
          leftSection={<IconSearch size={14} />}
          placeholder="Search by title or tag"
          aria-label="Search videos"
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
          data-autofocus
        />
        {tags.length > 0 && (
          <Group gap={6} data-testid="video-tags">
            <Text size="sm" c="dimmed">Tags:</Text>
            {tags.map((t) => (
              <Button
                key={t}
                size="compact-xs"
                variant={tag === t ? "filled" : "default"}
                aria-pressed={tag === t}
                styles={{ label: { overflow: "visible" } }}
                onClick={() => setTag(tag === t ? null : t)}
              >
                {t}
              </Button>
            ))}
          </Group>
        )}
        {error && <Alert color="red">{error}</Alert>}
        {!items && !error && <Loader size="sm" />}
        {items && items.length === 0 && <Text c="dimmed">No videos match.</Text>}
        {items && items.length > 0 && (
          <ScrollArea.Autosize mah={420} offsetScrollbars>
            <Stack gap={0}>
              {items.map((v) => (
                <Group
                  key={v.id}
                  justify="space-between"
                  wrap="nowrap"
                  py="sm"
                  style={{ borderBottom: "1px solid var(--mantine-color-gray-2)" }}
                  data-testid={`library-video-${v.title}`}
                >
                  <div style={{ minWidth: 0 }}>
                    <Text fw={600}>{v.title}</Text>
                    <Text size="sm" c="dimmed">
                      {[formatLength(v.lengthSeconds), v.platform ? "Preopsis" : "Your trust", v.tags.join(", ")].filter(Boolean).join(" · ")}
                    </Text>
                  </div>
                  <Button size="xs" style={{ flex: "none" }} onClick={() => onPick(v)}>Use this</Button>
                </Group>
              ))}
            </Stack>
          </ScrollArea.Autosize>
        )}
      </Stack>
    </Modal>
  );
}
