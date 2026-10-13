// The signed-in user, their trusts and the hospitals they can work in, loaded once from /api/me.
import { createContext, useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { Center, Loader } from "@mantine/core";
import { api, ApiError } from "./api";

/** The role ladder (A-5): each role can do everything the one before it can. */
export type Role = "clinician" | "super_clinician" | "admin";
/** A hospital the user can work in, with their role there. */
export type Hospital = { id: string; name: string; timeZone: string; role: Role };
/** A trust, its hospitals the user can work in, and their trust-wide role (null: some hospitals only). */
export type Trust = { id: string; name: string; code: string; role: Role | null; hospitals: Hospital[] };
export type Me = { user: { id: string; name: string; email: string }; platformAdmin: boolean; trusts: Trust[] };

const rank: Record<Role, number> = { clinician: 1, super_clinician: 2, admin: 3 };

/** Whether a role can do what `need` can, e.g. atLeast(role, "super_clinician") to publish. */
export const atLeast = (role: Role | null | undefined, need: Role) => !!role && rank[role] >= rank[need];

export const roleLabel: Record<Role, string> = { clinician: "Clinician", super_clinician: "Super clinician", admin: "Admin" };

/** The start of every URL in a hospital, for pages and API calls alike: /o/<trust>/h/<hospital>. */
export const hospitalBase = (trustId: string, hospitalId: string) => `/o/${trustId}/h/${hospitalId}`;

/** Where a user starts in a hospital: Episodes for clinicians, Questionnaires for the roles that author. */
export function homePath(t: Trust, h: Hospital): string {
  return `${hospitalBase(t.id, h.id)}/${atLeast(h.role, "super_clinician") ? "questionnaires" : "episodes"}`;
}

/**
 * Where switching to hospital `h` of trust `t` from the header goes, from the page at `pathname`.
 * It keeps the section: Episodes stays Episodes, Questionnaires (and the editor or preview) stays
 * Questionnaires (S08), and Staff or Settings stays so where the user manages that trust (S09, S10);
 * anything else goes to the role's start page.
 */
export function switchPath(pathname: string, t: Trust, h: Hospital): string {
  const base = hospitalBase(t.id, h.id);
  if (pathname.endsWith("/staff") && canManageStaff(t)) return `${base}/staff`;
  if (pathname.endsWith("/settings") && canManageSettings(t)) return `${base}/settings`;
  if (pathname.includes("/episodes")) return `${base}/episodes`;
  if (/\/(questionnaires|chapters)/.test(pathname)) return `${base}/questionnaires`;
  return homePath(t, h);
}

/** Every hospital the user can work in, with its trust, in the order /api/me gives them. */
export function allHospitals(me: Me): { trust: Trust; hospital: Hospital }[] {
  return me.trusts.flatMap((trust) => trust.hospitals.map((hospital) => ({ trust, hospital })));
}

/**
 * Where to go after sign-in: straight in with one hospital, else the hospital picker (STF-03). A
 * platform admin with no hospital of their own goes to the admin console (S10).
 */
export function startPath(me: Me): string {
  const all = allHospitals(me);
  if (all.length === 1) return homePath(all[0].trust, all[0].hospital);
  return all.length === 0 && me.platformAdmin ? "/admin" : "/";
}

/** The trust and hospital of a URL's ids, if the user can work there. */
export function findHospital(me: Me, trustId: string, hospitalId: string): { trust: Trust; hospital: Hospital } | null {
  const trust = me.trusts.find((t) => t.id === trustId);
  const hospital = trust?.hospitals.find((h) => h.id === hospitalId);
  return trust && hospital ? { trust, hospital } : null;
}

const AuthContext = createContext<Me | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate();
  const [me, setMe] = useState<Me | null>(null);
  useEffect(() => {
    api<Me>("/me")
      .then(setMe)
      .catch((e) => {
        if (e instanceof ApiError && e.status === 401) navigate("/login", { replace: true });
      });
  }, [navigate]);
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

/** The page's trust and hospital, the user's role there, and the URL base, set by the hospital layout. */
export type Here = { trust: Trust; hospital: Hospital; role: Role; base: string };

export const HospitalContext = createContext<Here | null>(null);

/**
 * Whether the user may change content (a questionnaire, an option list) with role `need`. The
 * trust's shared content (ownerHospitalId null) needs a whole-trust role; a hospital's own content
 * needs the role at this hospital (S08 answers, 5: Rahul, 12 Oct). The API checks the same.
 */
export function canChange(here: Here, ownerHospitalId: string | null | undefined, need: Role): boolean {
  return ownerHospitalId ? atLeast(here.role, need) : atLeast(here.trust.role, need);
}

/** Whether the user manages the trust's staff (S09): only admins of the whole trust can. The API checks the same. */
export const canManageStaff = (trust: Trust) => atLeast(trust.role, "admin");

/** Whether the user changes the trust's settings (S10): only admins of the whole trust, as for staff. The API checks the same. */
export const canManageSettings = canManageStaff;

export function useHospital(): Here {
  const here = useContext(HospitalContext);
  if (!here) throw new Error("useHospital must be used inside the hospital layout");
  return here;
}
