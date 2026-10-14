// The platform admin's pages (S10): /admin/…, for platform admins only (A-21). Everyone else sees
// a refusal and nothing is asked of the admin API, which refuses them too (403).
import { Link, Outlet, useLocation, useNavigate } from "react-router";
import { Alert, Anchor, AppShell, Badge, Button, Group, Text, Title } from "@mantine/core";
import { api } from "@/lib/api";
import { signOut as signOutStaff } from "@/lib/signin";
import { AuthProvider, allHospitals, useMe } from "@/lib/auth";

function Shell() {
  const me = useMe();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const signOut = () => signOutStaff(() => navigate("/login", { replace: true }), () => api("/logout", { method: "POST" }));
  return (
    <AppShell header={{ height: 56 }} padding="md">
      <AppShell.Header px="md">
        <Group h="100%" justify="space-between" wrap="nowrap">
          <Group wrap="nowrap">
            <Title order={4}>Poised</Title>
            <Badge variant="light" color="grape">Platform admin</Badge>
            {me.platformAdmin && (
              <Button component={Link} to="/admin" size="xs" variant={pathname.startsWith("/admin") ? "light" : "subtle"}>
                Trusts
              </Button>
            )}
            {allHospitals(me).length > 0 && (
              <Button component={Link} to="/" size="xs" variant="subtle">Your hospitals</Button>
            )}
          </Group>
          <Group wrap="nowrap">
            <Text size="sm" visibleFrom="sm">{me.user.name}</Text>
            <Button variant="default" size="xs" onClick={signOut}>Sign out</Button>
          </Group>
        </Group>
      </AppShell.Header>
      <AppShell.Main>
        {me.platformAdmin ? (
          <Outlet />
        ) : (
          <Alert color="red" title="Only Poised's platform admins can open this page">
            <Anchor component={Link} to="/">Go to your hospitals</Anchor>
          </Alert>
        )}
      </AppShell.Main>
    </AppShell>
  );
}

export function AdminLayout() {
  return (
    <AuthProvider>
      <Shell />
    </AuthProvider>
  );
}
