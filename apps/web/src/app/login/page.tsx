"use client";
// Stub login: pick a seeded user. Replaced by SSO later.
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, Button, Card, Container, Stack, Text, Title } from "@mantine/core";
import { api } from "@/lib/api";
import type { Me } from "@/lib/auth";

type User = { id: string; name: string; email: string };

export default function LoginPage() {
  const router = useRouter();
  const [users, setUsers] = useState<User[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    api<User[]>("/dev/users").then(setUsers).catch((e: Error) => setError(e.message));
  }, []);

  const signIn = async (userId: string) => {
    setError("");
    try {
      await api("/dev/login", { method: "POST", body: JSON.stringify({ userId }) });
      const me = await api<Me>("/me");
      router.push(me.hospitals.length ? `/h/${me.hospitals[0].id}/questionnaires` : "/");
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <Container size="xs" py="xl">
      <Title order={2} mb="xs">Lifebox Authoring</Title>
      <Text c="dimmed" mb="lg">Development sign-in: choose a user.</Text>
      {error && <Alert color="red" mb="md">{error}</Alert>}
      <Stack>
        {users.map((u) => (
          <Card key={u.id} withBorder padding="sm">
            <Button variant="subtle" justify="space-between" fullWidth onClick={() => signIn(u.id)}>
              {u.name}
              <Text span size="sm" c="dimmed" ml="sm">{u.email}</Text>
            </Button>
          </Card>
        ))}
      </Stack>
    </Container>
  );
}
