// The staff sign-in page. On AWS (S12) staff sign in through Cognito's managed login: a password,
// then a code from their authenticator app. Locally the dev login lists every user to sign in as
// (DEV_LOGIN; the API never serves it on AWS). Both show when both are on.
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { Alert, Button, Card, Container, Divider, Stack, Text, Title } from "@mantine/core";
import { api } from "@/lib/api";
import { Logo } from "@/features/brand/Logo";
import { startPath, type Me } from "@/lib/auth";
import { staffSignIn } from "@/lib/config";
import { clearRefusal, refusal, startSignIn } from "@/lib/signin";

type User = { id: string; name: string; email: string };

export function Login() {
  const navigate = useNavigate();
  const { devLogin, cognito } = staffSignIn();
  const [users, setUsers] = useState<User[]>([]);
  // A refused sign-in comes back here with its message (/auth/callback, through the managed
  // login's sign-out), shown once.
  const [error, setError] = useState(refusal);
  useEffect(clearRefusal, []);
  const [going, setGoing] = useState(false);

  useEffect(() => {
    if (devLogin) api<User[]>("/dev/users").then(setUsers).catch((e: Error) => setError(e.message));
  }, [devLogin]);

  const devSignIn = async (userId: string) => {
    setError("");
    try {
      await api("/dev/login", { method: "POST", body: JSON.stringify({ userId }) });
      const me = await api<Me>("/me");
      navigate(startPath(me));
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <Container size="xs" py="xl">
      <Title order={2} mb="xs"><Logo /></Title>
      {error && <Alert color="red" mb="md">{error}</Alert>}
      {cognito && (
        <Stack mb="xl">
          <Text c="dimmed">Sign in with your email address, your password and a code from your authenticator app.</Text>
          <Button
            size="md"
            loading={going}
            onClick={() => {
              setGoing(true);
              void startSignIn(cognito);
            }}
          >
            Sign in
          </Button>
          <Text size="sm" c="dimmed">
            New to Preopsis? Use the temporary password in your invitation email. Forgotten your password? Choose Sign in,
            then "Forgot your password?".
          </Text>
        </Stack>
      )}
      {cognito && devLogin && <Divider mb="lg" label="or, on this machine only" />}
      {devLogin && (
        <>
          <Text c="dimmed" mb="lg">Development sign-in (local only): choose a user.</Text>
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
    </Container>
  );
}
