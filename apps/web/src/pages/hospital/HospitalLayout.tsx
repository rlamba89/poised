import { Link, Outlet, useLocation, useNavigate, useParams } from "react-router";
import { Alert, Anchor, AppShell, Badge, Button, Group, Select, Text, Title } from "@mantine/core";
import { api } from "@/lib/api";
import { AuthProvider, HospitalContext, canManageSettings, canManageStaff, findHospital, hospitalBase, roleLabel, switchPath, useMe, type Here } from "@/lib/auth";

function NoAccess() {
  return (
    <Alert color="red" m="md" title="You don't have access to this hospital">
      <Anchor component={Link} to="/">Choose a hospital</Anchor>
    </Alert>
  );
}

function Shell({ children }: { children: React.ReactNode }) {
  const me = useMe();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const { orgId, hospitalId } = useParams() as { orgId: string; hospitalId: string };
  const found = findHospital(me, orgId, hospitalId);
  const here: Here | null = found && { ...found, role: found.hospital.role, base: hospitalBase(orgId, hospitalId) };

  // Every hospital the user can work in, grouped by trust (STF-03). Hospital ids are unique.
  // Switching keeps you in the same section (switchPath).
  const switchHospital = (id: string | null) => {
    for (const t of me.trusts) {
      const h = t.hospitals.find((x) => x.id === id);
      if (h && h.id !== hospitalId) navigate(switchPath(pathname, t, h));
    }
  };
  const nav = (href: string, label: string) => (
    <Button component={Link} to={href} size="xs" variant={pathname.startsWith(href) ? "light" : "subtle"}>
      {label}
    </Button>
  );
  const signOut = async () => {
    await api("/logout", { method: "POST" });
    navigate("/login", { replace: true });
  };
  const page = here ? <HospitalContext.Provider value={here}>{children}</HospitalContext.Provider> : <NoAccess />;
  // The questionnaire editor has its own header (Lifebox layout), so it gets the whole screen.
  if (/\/questionnaires\/[^/]+$/.test(pathname)) return page;
  // The preview needs the full width and height, so it gets no page padding.
  const fullBleed = pathname.includes("/preview");

  return (
    <AppShell header={{ height: 56 }} padding={fullBleed ? 0 : "md"}>
      <AppShell.Header px="md">
        {/* In a narrow window the header scrolls sideways rather than pushing Episodes and Sign out
            out of reach (S08 test report 2). */}
        <Group h="100%" justify="space-between" wrap="nowrap" style={{ overflowX: "auto", overflowY: "hidden" }}>
          <Group wrap="nowrap" style={{ flexShrink: 0 }}>
            <Title order={4}>Poised</Title>
            {here && (
              <Text fw={500} size="sm" aria-label="Trust">
                {here.trust.name}
              </Text>
            )}
            <Select
              aria-label="Hospital"
              data={me.trusts.map((t) => ({ group: t.name, items: t.hospitals.map((h) => ({ value: h.id, label: h.name })) }))}
              value={here ? hospitalId : null}
              placeholder="Choose a hospital"
              onChange={switchHospital}
              allowDeselect={false}
              w={{ base: 150, sm: 200 }}
            />
            {here && nav(`${here.base}/questionnaires`, "Questionnaires")}
            {here && nav(`${here.base}/episodes`, "Episodes")}
            {here && canManageStaff(here.trust) && nav(`${here.base}/staff`, "Staff")}
            {here && canManageSettings(here.trust) && nav(`${here.base}/settings`, "Settings")}
            {me.platformAdmin && nav("/admin", "Platform admin")}
          </Group>
          <Group wrap="nowrap" style={{ flexShrink: 0 }}>
            <Text size="sm" visibleFrom="md">{me.user.name}</Text>
            {here && (
              <Badge variant="light" aria-label="Your role here">
                {roleLabel[here.role]}
              </Badge>
            )}
            <Button variant="default" size="xs" onClick={signOut}>Sign out</Button>
          </Group>
        </Group>
      </AppShell.Header>
      <AppShell.Main>{page}</AppShell.Main>
    </AppShell>
  );
}

// Every /o/:orgId/h/:hospitalId/… page. Keyed by hospital, so switching hospitals starts afresh, and
// /api/me is read again, so a changed membership shows at once.
export function HospitalLayout() {
  const { hospitalId } = useParams() as { hospitalId: string };
  return (
    <AuthProvider key={hospitalId}>
      <Shell>
        <Outlet />
      </Shell>
    </AuthProvider>
  );
}
