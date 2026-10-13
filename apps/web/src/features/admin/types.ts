// What the platform admin's API (/api/admin/…, S10) returns. Go builds these in
// apps/api/internal/httpapi/admin.go; the names match its JSON.

/** A setting's definition (Go: settings.Def). The opt-ins are the ones with optIn true. */
export type SettingDef = {
  key: string;
  label: string;
  help?: string;
  group: string;
  level: "trust" | "hospital";
  kind: "bool" | "int" | "dayList" | "text" | "url" | "email" | "phone" | "questionnaire" | "template";
  optIn: boolean;
  default: unknown;
  min?: number;
  max?: number;
};

/** A trust in the list. */
export type TrustRow = { id: string; name: string; code: string; status: Status; hospitalCount: number; createdAt: string };

export type Status = "active" | "suspended";

/** On/off per opt-in key, e.g. { "feature.proms": true }. */
export type OptIns = Record<string, boolean>;

export type AdminHospital = {
  id: string;
  name: string;
  address: string;
  timeZone: string; // its own, or the deployment's
  ownTimeZone: string | null; // null: the deployment's
  status: Status;
  phone: string; // its own; "" follows the trust
  email: string;
  logoUrl: string;
  optIns: OptIns; // set for this hospital; a missing key follows the trust
  effectiveOptIns: OptIns;
};

export type AdminTrust = {
  id: string;
  name: string;
  code: string;
  status: Status;
  contactName: string;
  contactEmail: string;
  contactPhone: string;
  logoUrl: string;
  createdAt: string;
  optIns: OptIns; // set for the trust; a missing key uses the platform's default
  effectiveOptIns: OptIns;
  hospitals: AdminHospital[];
  admins: { userId: string; name: string; email: string; status: "active" | "invited" | "expired"; inviteExpiresAt: string | null }[];
  optInDefs: SettingDef[];
  existingAccount?: boolean; // after inviting an admin: they already had an account
};
