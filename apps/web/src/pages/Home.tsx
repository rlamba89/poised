import { useEffect } from "react";
import { useNavigate } from "react-router";
import { Center, Loader } from "@mantine/core";
import { api } from "@/lib/api";
import { homePath, type Me } from "@/lib/auth";

// Sends the user to their first hospital, or to sign-in.
export function Home() {
  const navigate = useNavigate();
  useEffect(() => {
    api<Me>("/me")
      .then((me) => navigate(me.hospitals.length ? homePath(me.hospitals[0]) : "/login", { replace: true }))
      .catch(() => navigate("/login", { replace: true }));
  }, [navigate]);
  return (
    <Center h="100vh">
      <Loader />
    </Center>
  );
}
