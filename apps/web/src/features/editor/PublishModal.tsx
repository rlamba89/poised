"use client";
// Publishing a draft (LCY-01). Logic problems and failing test cases in any Question Set
// block it (SGN-04); review and sign-off come later.
import { useState } from "react";
import { Alert, Button, Group, List, Modal, Stack, Text } from "@mantine/core";
import { publishProblems, type ChapterJson } from "@poised/clinical";
import { api } from "@/lib/api";

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
};

export function PublishModal({ opened, onClose, onPublished, path, versionId, versionNo, sets, unsaved }: Props) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const problems = opened && sets ? publishProblems(sets) : [];
  const blocked = !sets || unsaved || problems.length > 0 || sets.length === 0;

  const publish = async () => {
    setBusy(true);
    setError("");
    try {
      await api(`${path}/publish`, { method: "POST", body: JSON.stringify({ versionId }) });
      onPublished();
      onClose();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal opened={opened} onClose={onClose} title={`Publish version ${versionNo}?`}>
      <Stack>
        {!sets && <Text c="dimmed">Checking the Question Sets…</Text>}
        {sets && sets.length === 0 && <Alert color="orange">Add at least one Question Set before publishing.</Alert>}
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
