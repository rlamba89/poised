"use client";
// Signed-in user and their hospitals, loaded once from /api/me.
import { createContext, useContext, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Center, Loader } from "@mantine/core";
import { api, ApiError } from "./api";

export type Role = "viewer" | "author" | "reviewer" | "publisher" | "hospital_admin";
export type Hospital = { id: string; name: string; roles: Role[] };
export type Me = { user: { id: string; name: string; email: string }; hospitals: Hospital[] };

const AuthContext = createContext<Me | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  useEffect(() => {
    api<Me>("/me")
      .then(setMe)
      .catch((e) => {
        if (e instanceof ApiError && e.status === 401) router.replace("/login");
      });
  }, [router]);
  if (!me) {
    return (
      <Center h="100vh">
        <Loader />
      </Center>
    );
  }
  return <AuthContext.Provider value={me}>{children}</AuthContext.Provider>;
}

export function useMe(): Me {
  const me = useContext(AuthContext);
  if (!me) throw new Error("useMe must be used inside AuthProvider");
  return me;
}

export function hasRole(me: Me, hospitalId: string, role: Role): boolean {
  return me.hospitals.some((h) => h.id === hospitalId && h.roles.includes(role));
}
