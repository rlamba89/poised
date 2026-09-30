"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Center, Loader } from "@mantine/core";
import { api } from "@/lib/api";
import type { Me } from "@/lib/auth";

// Sends the user to their first hospital, or to sign-in.
export default function Home() {
  const router = useRouter();
  useEffect(() => {
    api<Me>("/me")
      .then((me) => router.replace(me.hospitals.length ? `/h/${me.hospitals[0].id}/questionnaires` : "/login"))
      .catch(() => router.replace("/login"));
  }, [router]);
  return (
    <Center h="100vh">
      <Loader />
    </Center>
  );
}
