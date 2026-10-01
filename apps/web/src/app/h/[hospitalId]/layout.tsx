"use client";
import Link from "next/link";
import { useParams, usePathname, useRouter } from "next/navigation";
import { Alert, AppShell, Button, Group, Select, Text, Title } from "@mantine/core";
import { api } from "@/lib/api";
import { AuthProvider, homePath, useMe } from "@/lib/auth";

function Shell({ children }: { children: React.ReactNode }) {
  const me = useMe();
  const router = useRouter();
  const pathname = usePathname();
  const { hospitalId } = useParams<{ hospitalId: string }>();
  const hospital = me.hospitals.find((h) => h.id === hospitalId);
  const member = !!hospital;
  const authoring = !!hospital?.roles.some((r) => r !== "clinician");
  const clinical = !!hospital?.roles.includes("clinician");

  const switchHospital = (id: string | null) => {
    const next = me.hospitals.find((h) => h.id === id);
    if (next && next.id !== hospitalId) router.push(homePath(next));
  };
  const nav = (href: string, label: string) => (
    <Button component={Link} href={href} size="xs" variant={pathname.startsWith(href) ? "light" : "subtle"}>
      {label}
    </Button>
  );
  const signOut = async () => {
    await api("/logout", { method: "POST" });
    router.replace("/login");
  };
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
              data={me.hospitals.map((h) => ({ value: h.id, label: h.name }))}
              value={member ? hospitalId : null}
              onChange={switchHospital}
              allowDeselect={false}
              w={200}
            />
            {authoring && nav(`/h/${hospitalId}/questionnaires`, "Questionnaires")}
            {clinical && nav(`/h/${hospitalId}/episodes`, "Episodes")}
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

export default function HospitalLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthProvider>
      <Shell>{children}</Shell>
    </AuthProvider>
  );
}
