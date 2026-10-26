// The patient's Information for one episode (S36, INF-04), on its card on their home (S36 answers 1):
// an Information button that opens a simple list of the episode's videos (each embedded with the
// video site's own player, built only from a recognised YouTube or Vimeo ID: S34's videoEmbedFor)
// and leaflets (links that open in a new tab), then the hospital's phone number and email.
//
// Everything here came from authors or staff, so it's shown as plain text (React escapes it) and
// links are only ever built from checked parts: a leaflet's https link (Go checks it, and the
// database refuses anything else), a tel: link of the phone's digits, a mailto: link of a plain
// address. Nothing is put on the page as HTML.
import { useState } from "react";
import { Alert, Anchor, Button, Group, Loader, Stack, Text } from "@mantine/core";
import { IconChevronDown, IconChevronUp, IconFileText, IconMail, IconPhone } from "@tabler/icons-react";
import { formatLength, videoEmbedFor } from "@poised/clinical";
import { api } from "@/lib/api";
import { mailtoHref, telHref, type PatientInfo } from "./info";

export function PatientInformation({ episodeId }: { episodeId: string }) {
  const [open, setOpen] = useState(false);
  const [info, setInfo] = useState<PatientInfo | null>(null);
  const [error, setError] = useState("");

  const toggle = () => {
    if (!open && !info) {
      setError("");
      api<PatientInfo>(`/p/episodes/${episodeId}/information`)
        .then(setInfo)
        .catch((e: Error) => setError(e.message));
    }
    setOpen(!open);
  };

  return (
    <div data-testid="home-information">
      <Button variant="light" size="sm" onClick={toggle} aria-expanded={open}
        rightSection={open ? <IconChevronUp size={16} /> : <IconChevronDown size={16} />}>
        Information
      </Button>
      {open && (
        <Stack gap="md" mt="sm" data-testid="home-information-list">
          {error && <Alert color="red" p="xs">{error}</Alert>}
          {!info && !error && <Loader size="sm" />}
          {info && <InformationList info={info} />}
        </Stack>
      )}
    </div>
  );
}

function InformationList({ info }: { info: PatientInfo }) {
  const tel = telHref(info.phone);
  const mail = mailtoHref(info.email);
  return (
    <>
      {info.items.length === 0 && <Text size="sm" c="dimmed">There are no videos or leaflets for this yet.</Text>}
      {info.items.map((i) =>
        i.kind === "video" ? (
          <Video key={i.id} title={i.title} url={i.url} lengthSeconds={i.lengthSeconds} />
        ) : (
          <Group key={i.id} gap="xs" wrap="nowrap" align="flex-start" data-testid={`info-leaflet-${i.title}`}>
            <IconFileText size={20} style={{ flexShrink: 0, marginTop: 2 }} aria-hidden />
            {i.url.startsWith("https://") ? (
              <Anchor href={i.url} target="_blank" rel="noopener noreferrer" style={{ overflowWrap: "anywhere" }}>{i.title}</Anchor>
            ) : (
              <Text style={{ overflowWrap: "anywhere" }}>{i.title}</Text>
            )}
          </Group>
        ),
      )}
      {(info.phone || info.email) && (
        <Stack gap={6} data-testid="info-contact">
          <Text size="sm" fw={600}>Questions? Contact {info.hospital}</Text>
          {info.phone && (
            <Group gap="xs" wrap="nowrap">
              <IconPhone size={18} style={{ flexShrink: 0 }} aria-hidden />
              {tel ? <Anchor href={tel}>{info.phone}</Anchor> : <Text>{info.phone}</Text>}
            </Group>
          )}
          {info.email && (
            <Group gap="xs" wrap="nowrap">
              <IconMail size={18} style={{ flexShrink: 0 }} aria-hidden />
              {mail ? <Anchor href={mail} style={{ overflowWrap: "anywhere" }}>{info.email}</Anchor> : <Text style={{ overflowWrap: "anywhere" }}>{info.email}</Text>}
            </Group>
          )}
        </Stack>
      )}
    </>
  );
}

/** A video, as in a questionnaire (S34): its title, length and the video site's own player. */
function Video({ title, url, lengthSeconds }: { title: string; url: string; lengthSeconds: number | null }) {
  const src = videoEmbedFor(url);
  return (
    <div className="sj-video" data-testid={`info-video-${title}`}>
      <Text fw={600} style={{ overflowWrap: "anywhere" }}>{title}</Text>
      {lengthSeconds ? <p className="sj-video-length">Length {formatLength(lengthSeconds)}</p> : null}
      {src ? (
        <div className="sj-video-frame">
          <iframe src={src} title={title} loading="lazy" allow="encrypted-media; picture-in-picture; fullscreen" allowFullScreen
            referrerPolicy="strict-origin-when-cross-origin" />
        </div>
      ) : (
        <p className="sj-video-missing">This video can&apos;t be shown.</p>
      )}
    </div>
  );
}
