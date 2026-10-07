"use client";
// Signed-in user and their hospitals, loaded once from /api/me.
import { createContext, useContext, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Center, Loader } from "@mantine/core";
import { api, ApiError } from "./api";
import { hospitalPath } from "./paths";

/** The staff role ladder (A-5): each role can do everything the role below it can. */
export type Role = "clinician" | "super_clinician" | "admin";
/** A hospital the user can open, its trust, and the role that applies there. */
export type Hospital = { id: string; name: string; orgId: string; orgName: string; role: Role };
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

const rank: Record<Role, number> = { clinician: 1, super_clinician: 2, admin: 3 };

/** Whether the user's role in this hospital is `want` or above. The server checks the same. */
export function atLeast(me: Me, hospitalId: string, want: Role): boolean {
  const h = me.hospitals.find((h) => h.id === hospitalId);
  return !!h && rank[h.role] >= rank[want];
}

/** Where a user starts in a hospital: Questionnaires for super clinicians and admins, otherwise Episodes. */
export function homePath(h: Hospital): string {
  const page = rank[h.role] >= rank.super_clinician ? "questionnaires" : "episodes";
  return `${hospitalPath(h.orgId, h.id)}/${page}`;
}
