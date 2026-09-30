"use client";
import { useParams, usePathname, useRouter } from "next/navigation";
import { Alert, AppShell, Button, Group, Select, Text, Title } from "@mantine/core";
import { api } from "@/lib/api";
import { AuthProvider, useMe } from "@/lib/auth";

function Shell({ children }: { children: React.ReactNode }) {
  const me = useMe();
  const router = useRouter();
  const pathname = usePathname();
  const { hospitalId } = useParams<{ hospitalId: string }>();
  const member = me.hospitals.some((h) => h.id === hospitalId);

  const switchHospital = (id: string | null) => {
    if (id && id !== hospitalId) router.push(`/h/${id}/questionnaires`);
  };
  const signOut = async () => {
    await api("/logout", { method: "POST" });
    router.replace("/login");
  };
  // The designer needs the full width and height, so it gets no page padding.
  const fullBleed = pathname.includes("/design") || pathname.includes("/preview");

  return (
    <AppShell header={{ height: 56 }} padding={fullBleed ? 0 : "md"}>
      <AppShell.Header px="md">
        <Group h="100%" justify="space-between">
          <Group>
            <Title order={4}>Lifebox Authoring</Title>
            <Select
              aria-label="Hospital"
              data={me.hospitals.map((h) => ({ value: h.id, label: h.name }))}
              value={member ? hospitalId : null}
              onChange={switchHospital}
              allowDeselect={false}
              w={200}
            />
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
