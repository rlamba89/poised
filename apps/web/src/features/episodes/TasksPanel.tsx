// The episode's Tasks panel (S21, TSK-01): add a task for a team, complete it, re-open it. Anyone
// at the hospital may do any of these (S21 answers 1). Each task says who added it and when, and
// who completed or re-opened it. A change sends the episode's revision (S20): a stale page is
// refused with Reload, keeping what was typed. A cancelled or archived (S22) episode's tasks are
// shown, not changed.
//
// The add form is a real form with noValidate (S19's lesson), so Enter sends it with the app's own
// messages; the title is one line, at most 200 characters, as the API takes.
import { useEffect, useState, type FormEvent } from "react";
import { Badge, Button, Card, Group, NativeSelect, Stack, Text, TextInput, Title } from "@mantine/core";
import { api } from "@/lib/api";
import type { Episode } from "./types";
import { SaveError, isStale } from "./StatusControl";
import { MAX_TASK_TITLE, TEAMS, openFirst, openTaskCount, taskHistory, teamLabel, type Task, type Team } from "./tasks";

export function TasksPanel({ path, episode, onChanged }: { path: string; episode: Episode; onChanged: () => void }) {
  const [tasks, setTasks] = useState<Task[] | null>(null);
  const [loadError, setLoadError] = useState("");
  const [adding, setAdding] = useState(false);
  const [title, setTitle] = useState("");
  const [team, setTeam] = useState<Team | "">("");
  const [error, setError] = useState("");
  const [stale, setStale] = useState(false);
  const [busy, setBusy] = useState(""); // "add", or the id of the task being changed
  const readOnly = episode.status === "cancelled" || episode.status === "archived";

  // Loaded again whenever the episode changes (its revision), so after any change, here or by
  // someone else, the list is the server's.
  useEffect(() => {
    api<Task[]>(`${path}/tasks`)
      .then((t) => {
        setTasks(t);
        setLoadError("");
      })
      .catch((e: Error) => setLoadError(e.message));
  }, [path, episode.revision]);

  const fail = (err: unknown) => {
    setError((err as Error).message);
    setStale(isStale(err));
  };

  const add = async (ev?: FormEvent) => {
    ev?.preventDefault(); // Enter in the title sends the form
    if (busy) return;
    if (!title.trim()) return fail(new Error("Enter the task."));
    if (!team) return fail(new Error("Choose the team."));
    setBusy("add");
    try {
      await api(`${path}/tasks`, { method: "POST", body: JSON.stringify({ title, team, revision: episode.revision }) });
      setTitle("");
      setTeam("");
      setAdding(false);
      setError("");
      onChanged();
    } catch (err) {
      fail(err);
    } finally {
      setBusy("");
    }
  };

  const mark = async (t: Task, done: boolean) => {
    if (busy) return;
    setBusy(t.id);
    try {
      await api(`${path}/tasks/${t.id}`, { method: "PATCH", body: JSON.stringify({ done, revision: episode.revision }) });
      setError("");
      onChanged();
    } catch (err) {
      fail(err);
    } finally {
      setBusy("");
    }
  };

  const open = openTaskCount(episode.status, tasks?.filter((t) => !t.doneAt).length ?? 0); // none once archived (S29)
  return (
    <Card withBorder data-testid="tasks">
      <Group justify="space-between" mb="xs">
        <Title order={4}>Tasks {tasks && tasks.length > 0 && open !== null && <Text span size="sm" c="dimmed">({open} open)</Text>}</Title>
        {!readOnly && !adding && (
          <Button size="xs" onClick={() => { setAdding(true); setError(""); setStale(false); }}>Add task</Button>
        )}
      </Group>
      {loadError && <Text size="sm" c="red">{loadError}</Text>}
      {readOnly && <Text size="sm" c="dimmed" mb="xs">This episode is {episode.status === "archived" ? "archived" : "cancelled"}, so its tasks can&apos;t be changed.</Text>}
      {tasks && tasks.length === 0 && !adding && <Text size="sm" c="dimmed">No tasks yet.</Text>}

      <Stack gap="xs">
        {tasks && openFirst(tasks).map((t) => (
          <Group key={t.id} justify="space-between" wrap="nowrap" align="flex-start" data-testid="task" data-done={t.doneAt ? "true" : "false"}>
            <div>
              <Group gap="xs">
                <Text size="sm" fw={600} td={t.doneAt ? "line-through" : undefined} c={t.doneAt ? "dimmed" : undefined}>{t.title}</Text>
                <Badge size="sm" variant="light" color={t.team === "anaesthetist" ? "violet" : t.team === "nurse" ? "teal" : "gray"}>
                  {teamLabel(t.team)}
                </Badge>
                {t.doneAt && <Badge size="sm" variant="outline" color="green">Done</Badge>}
              </Group>
              {taskHistory(t).map((line) => <Text key={line} size="xs" c="dimmed">{line}</Text>)}
            </div>
            {!readOnly && (
              <Button size="xs" variant={t.doneAt ? "default" : "light"} loading={busy === t.id} onClick={() => mark(t, !t.doneAt)}
                aria-label={`${t.doneAt ? "Re-open" : "Complete"}: ${t.title}`}>
                {t.doneAt ? "Re-open" : "Complete"}
              </Button>
            )}
          </Group>
        ))}
      </Stack>

      {adding && (
        <Card withBorder component="form" onSubmit={add} noValidate mt="sm" bg="gray.0" data-testid="task-form">
          <Stack gap="sm">
            <Group align="flex-end" gap="sm">
              <TextInput label="Task" placeholder="e.g. Book anaesthetic review" maxLength={MAX_TASK_TITLE} value={title}
                onChange={(e) => setTitle(e.currentTarget.value)} style={{ flex: 1, minWidth: 220 }} data-autofocus />
              <NativeSelect label="Team" value={team} onChange={(e) => setTeam(e.currentTarget.value as Team | "")} w={180}
                data={[{ value: "", label: "Choose a team" }, ...TEAMS]} />
            </Group>
            <SaveError error={error} stale={stale} onReload={() => { setError(""); setStale(false); onChanged(); }} />
            <Group gap="xs">
              <Button type="submit" loading={busy === "add"}>Add task</Button>
              <Button variant="default" onClick={() => { setAdding(false); setError(""); }}>Back</Button>
            </Group>
          </Stack>
        </Card>
      )}
      {!adding && <SaveError error={error} stale={stale} onReload={() => { setError(""); setStale(false); onChanged(); }} />}
    </Card>
  );
}
