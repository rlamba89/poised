// Publishing a draft (LCY-01). Logic problems and failing test cases in any Question Set
// block it (SGN-04); review and sign-off come later. So do the question-key checks (S23): a key
// used twice in the HQ, or a published question whose answer changed. Those are Go's: the window
// shows them from GET …/question-keys (fetched again as it opens), and Publish's own 422 the same way.
import { useEffect, useRef, useState } from "react";
import { Alert, Button, Group, List, Modal, Stack, Text } from "@mantine/core";
import { publishProblems, type ChapterJson, type PublishProblem } from "@poised/clinical";
import { api, ApiError } from "@/lib/api";

type Props = {
  opened: boolean;
  onClose: () => void;
  onPublished: () => void;
  path: string;
  versionId: string;
  versionNo: number;
  /** Every Question Set in order, or null while some are still loading. */
  sets: { name: string; doc: ChapterJson }[] | null;
  unsaved: boolean;
  /** Go's key problems for the saved draft, or null while they're being fetched. */
  keyProblems: PublishProblem[] | null;
  /** Called as the window opens and after a refusal, to fetch Go's key problems again. */
  onOpen: () => void;
};

export function PublishModal({ opened, onClose, onPublished, path, versionId, versionNo, sets, unsaved, keyProblems, onOpen }: Props) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  // Go's own problems from its last refusal (422), until the dialog is opened again.
  const [refused, setRefused] = useState<PublishProblem[]>([]);
  const [wasOpen, setWasOpen] = useState(opened);
  if (opened !== wasOpen) {
    setWasOpen(opened);
    if (opened) {
      setRefused([]);
      setError("");
    }
  }
  useEffect(() => {
    if (opened) onOpen();
  }, [opened, onOpen]);
  // Go's refusal is shown only until its key problems are fetched again: a fresh list (even an
  // empty one, if the trust changed back) replaces it (verification 2, item 3). Compared as text,
  // so a new but equal array from a re-render doesn't count.
  const freshProblems = JSON.stringify(keyProblems);
  useEffect(() => {
    if (keyProblems) setRefused([]);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [freshProblems]);
  // The problems are worked out while the dialog is open, and the last ones kept while it fades out,
  // so Cancel doesn't flash the "ready to publish" text (S08 test report 2).
  const shown = useRef<ReturnType<typeof publishProblems>>([]);
  if (opened) shown.current = sets && keyProblems ? [...publishProblems(sets), ...keyProblems] : [];
  const problems = shown.current.length ? shown.current : refused;
  const checking = !sets || !keyProblems;
  const blocked = checking || unsaved || problems.length > 0 || sets.length === 0;

  const publish = async () => {
    setBusy(true);
    setError("");
    try {
      await api(`${path}/publish`, { method: "POST", body: JSON.stringify({ versionId }) });
      onPublished();
      onClose();
    } catch (e) {
      const body = e instanceof ApiError ? (e.body as { problems?: PublishProblem[] } | null) : null;
      if (body?.problems?.length) {
        setRefused(body.problems);
        onOpen(); // the keys changed since they were fetched: fetch them again
      } else setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal opened={opened} onClose={onClose} title={`Publish version ${versionNo}?`}>
      <Stack>
        {checking && <Text c="dimmed">Checking the Question Sets…</Text>}
        {!checking && sets.length === 0 && <Alert color="orange">Add at least one Question Set before publishing.</Alert>}
        {unsaved && <Alert color="orange">Wait until all changes are saved.</Alert>}
        {problems.length > 0 && (
          <Alert color="orange" title={`${problems.length} problem${problems.length === 1 ? "" : "s"} to fix first`}>
            <List size="sm" data-testid="publish-problems">
              {problems.map((p, i) => (
                <List.Item key={i}>
                  <b>{p.set}</b> · {p.message}
                </List.Item>
              ))}
            </List>
          </Alert>
        )}
        {!blocked && (
          <Text>
            Patients can then be given this version. It becomes read-only: to change it later, create a new version.
          </Text>
        )}
        {error && <Alert color="red">{error}</Alert>}
        <Group justify="flex-end">
          <Button variant="default" onClick={onClose}>Cancel</Button>
          <Button onClick={publish} disabled={blocked} loading={busy}>Publish</Button>
        </Group>
      </Stack>
    </Modal>
  );
}
