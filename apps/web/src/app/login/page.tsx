"use client";
// Sign-in: Cognito's managed login when it is set up, and the development stub (pick a seeded
// user) when DEV_LOGIN is on. The API's /auth/config says which.
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Alert, Button, Card, Container, Divider, Stack, Text, Title } from "@mantine/core";
import { api } from "@/lib/api";
import { homePath, type Me } from "@/lib/auth";
import { startCognitoSignIn, type AuthConfig } from "@/lib/signin";

type User = { id: string; name: string; email: string };

export default function LoginPage() {
  const router = useRouter();
  const [config, setConfig] = useState<AuthConfig | null>(null);
  const [users, setUsers] = useState<User[]>([]);
  const [error, setError] = useState("");
  const [redirecting, setRedirecting] = useState(false);

  useEffect(() => {
    api<AuthConfig>("/auth/config")
      .then((cfg) => {
        setConfig(cfg);
        if (cfg.devLogin) return api<User[]>("/dev/users").then(setUsers);
      })
      .catch((e: Error) => setError(e.message));
  }, []);

  const devSignIn = async (userId: string) => {
    setError("");
    try {
      await api("/dev/login", { method: "POST", body: JSON.stringify({ userId }) });
      const me = await api<Me>("/me");
      router.push(me.hospitals.length ? homePath(me.hospitals[0]) : "/");
    } catch (e) {
      setError((e as Error).message);
    }
  };

  const cognitoSignIn = async () => {
    if (!config?.cognito) return;
    setRedirecting(true);
    await startCognitoSignIn(config.cognito);
  };

  return (
    <Container size="xs" py="xl">
      <Title order={2} mb="lg">Sign in</Title>
      {error && <Alert color="red" mb="md">{error}</Alert>}
      {config?.cognito && (
        <Button size="md" fullWidth onClick={cognitoSignIn} loading={redirecting} mb="lg">
          Sign in with your work email
        </Button>
      )}
      {config?.devLogin && (
        <>
          {config.cognito && <Divider label="or, for development" mb="md" />}
          <Text c="dimmed" mb="md">Development sign-in: choose a user.</Text>
          <Stack>
            {users.map((u) => (
              <Card key={u.id} withBorder padding="sm">
                <Button variant="subtle" justify="space-between" fullWidth onClick={() => devSignIn(u.id)}>
                  {u.name}
                  <Text span size="sm" c="dimmed" ml="sm">{u.email}</Text>
                </Button>
              </Card>
            ))}
          </Stack>
        </>
      )}
      {config && !config.cognito && !config.devLogin && <Alert>No way to sign in is set up on this server.</Alert>}
    </Container>
  );
}
