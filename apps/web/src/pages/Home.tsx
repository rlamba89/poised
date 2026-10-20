// After sign-in: straight into the user's only hospital, or the hospital picker (STF-03) when
// they have more than one, grouped by trust. A trust with no hospitals yet is listed with a note
// (S10), and a platform admin gets a link to the admin console.
import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router";
import { Alert, Badge, Button, Card, Center, Container, Group, Loader, Stack, Text, Title } from "@mantine/core";
import { api } from "@/lib/api";
import { Logo } from "@/features/brand/Logo";
import { signOut as signOutStaff } from "@/lib/signin";
import { allHospitals, homePath, roleLabel, type Me } from "@/lib/auth";

export function Home() {
  const navigate = useNavigate();
  const [me, setMe] = useState<Me | null>(null);
  useEffect(() => {
    api<Me>("/me")
      .then((m) => {
        const all = allHospitals(m);
        if (all.length === 1) navigate(homePath(all[0].trust, all[0].hospital), { replace: true });
        else setMe(m);
      })
      .catch(() => navigate("/login", { replace: true }));
  }, [navigate]);
  const signOut = () => signOutStaff(() => navigate("/login", { replace: true }), () => api("/logout", { method: "POST" }));

  if (!me) {
    return (
      <Center h="100vh">
        <Loader />
      </Center>
    );
  }
  return (
    <Container size="sm" py="xl">
      <Group justify="space-between" mb="lg">
        <Title order={2}><Logo /></Title>
        <Group>
          <Text size="sm">{me.user.name}</Text>
          <Button variant="default" size="xs" onClick={signOut}>Sign out</Button>
        </Group>
      </Group>
      {me.platformAdmin && (
        <Card withBorder mb="lg">
          <Group justify="space-between">
            <Text>You're one of Preopsis's platform admins.</Text>
            <Button component={Link} to="/admin">Open the platform admin console</Button>
          </Group>
        </Card>
      )}
      {me.trusts.length === 0 ? (
        !me.platformAdmin && (
          <Alert title="You don't have access to any hospital yet">Ask an admin of your trust to invite you.</Alert>
        )
      ) : (
        <>
          {/* Nothing to choose while their trusts have no hospitals yet (S10 test report 1). */}
          <Title order={3} mb="md">{allHospitals(me).length > 0 ? "Choose a hospital" : "Your trusts"}</Title>
          <Stack>
            {me.trusts.map((t) => (
              <Card key={t.id} withBorder component="section" aria-label={t.name}>
                <Group justify="space-between" mb="sm">
                  <Title order={4}>{t.name}</Title>
                  {t.role && <Badge variant="light">{roleLabel[t.role]}, whole trust</Badge>}
                </Group>
                <Stack gap="xs">
                  {t.hospitals.length === 0 && (
                    <Text size="sm" c="dimmed">
                      {t.name} has no hospitals in Preopsis yet. Preopsis's platform admin adds them; you'll find them here once they have.
                    </Text>
                  )}
                  {t.hospitals.map((h) => (
                    <Button key={h.id} component={Link} to={homePath(t, h)} variant="default" justify="space-between" fullWidth
                      rightSection={<Text span size="sm" c="dimmed">{roleLabel[h.role]}</Text>}>
                      {h.name}
                    </Button>
                  ))}
                </Stack>
              </Card>
            ))}
          </Stack>
        </>
      )}
    </Container>
  );
}
