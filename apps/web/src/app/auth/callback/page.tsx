"use client";
// Cognito sends people back here after its managed login. The code goes to the API, which
// starts the session; then the user goes to their first hospital.
import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Alert, Anchor, Center, Container, Loader } from "@mantine/core";
import { api } from "@/lib/api";
import { homePath, type Me } from "@/lib/auth";
import { finishCognitoSignIn } from "@/lib/signin";

export default function CallbackPage() {
  const router = useRouter();
  const [error, setError] = useState("");
  const started = useRef(false); // the code works once; React runs effects twice in development

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    finishCognitoSignIn(new URLSearchParams(window.location.search))
      .then(() => api<Me>("/me"))
      .then((me) => router.replace(me.hospitals.length ? homePath(me.hospitals[0]) : "/login"))
      .catch((e: Error) => setError(e.message));
  }, [router]);

  if (!error) return <Center h="60vh"><Loader /></Center>;
  return (
    <Container size="xs" py="xl">
      <Alert color="red" mb="md">{error}</Alert>
      <Anchor component={Link} href="/login">Back to sign in</Anchor>
    </Container>
  );
}
