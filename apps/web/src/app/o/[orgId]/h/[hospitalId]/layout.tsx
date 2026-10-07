"use client";
import Link from "next/link";
import { useParams, usePathname, useRouter } from "next/navigation";
import { Alert, AppShell, Button, Group, Select, Text, Title } from "@mantine/core";
import { AuthProvider, atLeast, homePath, useMe, type Hospital } from "@/lib/auth";
import { useHospitalPath } from "@/lib/paths";
import { signOut } from "@/lib/signin";

function Shell({ children }: { children: React.ReactNode }) {
  const me = useMe();
  const router = useRouter();
  const pathname = usePathname();
  const { orgId, hospitalId } = useParams<{ orgId: string; hospitalId: string }>();
  const base = useHospitalPath();
  // The hospital must be reached through its own trust, as the API checks too.
  const hospital = me.hospitals.find((h) => h.id === hospitalId && h.orgId === orgId);
  const member = !!hospital;
  const authoring = atLeast(me, hospitalId, "super_clinician");

  const switchHospital = (id: string | null) => {
    const next = me.hospitals.find((h) => h.id === id);
    if (next && next.id !== hospitalId) router.push(homePath(next));
  };
  const nav = (href: string, label: string) => (
    <Button component={Link} href={href} size="xs" variant={pathname.startsWith(href) ? "light" : "subtle"}>
      {label}
    </Button>
  );
  // The questionnaire editor has its own header (Lifebox layout), so it gets the whole screen.
  const editor = /\/questionnaires\/[^/]+$/.test(pathname);
  // The preview needs the full width and height, so it gets no page padding.
  const fullBleed = pathname.includes("/preview");
  if (editor) return member ? <>{children}</> : <Alert color="red" m="md">You don&apos;t have access to this hospital.</Alert>;

  return (
    <AppShell header={{ height: 56 }} padding={fullBleed ? 0 : "md"}>
      <AppShell.Header px="md">
        <Group h="100%" justify="space-between">
          <Group>
            <Title order={4}>Lifebox</Title>
            <Select
              aria-label="Hospital"
              data={byTrust(me.hospitals)}
              value={member ? hospitalId : null}
              onChange={switchHospital}
              allowDeselect={false}
              w={240}
            />
            {authoring && nav(`${base}/questionnaires`, "Questionnaires")}
            {member && nav(`${base}/episodes`, "Episodes")}
            {atLeast(me, hospitalId, "admin") && nav(`${base}/staff`, "Staff")}
          </Group>
          <Group>
            <Text size="sm">{me.user.name}</Text>
            <Button variant="default" size="xs" onClick={signOut}>Sign out</Button>
          </Group>
        </Group>
      </AppShell.Header>
      <AppShell.Main>
        {member ? children : <Alert color="red" m="md">You don&apos;t have access to this hospital.</Alert>}
      </AppShell.Main>
    </AppShell>
  );
}

/** The hospital picker's options, grouped under each trust's name (STF-03). */
function byTrust(hospitals: Hospital[]) {
  const groups = new Map<string, { group: string; items: { value: string; label: string }[] }>();
  for (const h of hospitals) {
    const g = groups.get(h.orgId) ?? { group: h.orgName, items: [] };
    g.items.push({ value: h.id, label: h.name });
    groups.set(h.orgId, g);
  }
  return [...groups.values()];
}

export default function HospitalLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthProvider>
      <Shell>{children}</Shell>
    </AuthProvider>
  );
}
