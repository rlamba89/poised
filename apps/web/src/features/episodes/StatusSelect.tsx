"use client";
// The "Episode status" dropdown. The first three statuses are set by the workflow; the rest are
// picked by hand once the HQ review is complete (apps/api/internal/episode.CanSetStatus).
import { useState } from "react";
import { Alert, Group, Select, Text } from "@mantine/core";
import { api } from "@/lib/api";
import { MANUAL_STATUSES, STATUS_LABELS, type EpisodeStatus } from "./types";

export function StatusSelect({ path, status, onChanged }: { path: string; status: EpisodeStatus; onChanged: () => void }) {
  const [error, setError] = useState("");
  const open = status === "ready_for_poa" || MANUAL_STATUSES.includes(status);
  const options = MANUAL_STATUSES.includes(status) ? MANUAL_STATUSES : [status, ...MANUAL_STATUSES];

  const change = async (next: string | null) => {
    if (!next || next === status) return;
    try {
      await api(path, { method: "PATCH", body: JSON.stringify({ status: next }) });
      setError("");
      onChanged();
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <Group gap="sm">
      <Text size="sm" fw={600}>Episode status:</Text>
      <Select
        aria-label="Episode status"
        data={options.map((s) => ({ value: s, label: STATUS_LABELS[s], disabled: !MANUAL_STATUSES.includes(s) }))}
        value={status}
        onChange={change}
        allowDeselect={false}
        disabled={!open}
        w={240}
      />
      {!open && <Text size="xs" c="dimmed">Can be changed once the HQ review is complete.</Text>}
      {error && <Alert color="red" py={4}>{error}</Alert>}
    </Group>
  );
}
