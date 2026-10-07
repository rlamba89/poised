"use client";
// The URL prefix of one hospital, under its trust: /o/{orgId}/h/{hospitalId}. Pages and the API
// share it (api() adds /api), so the same prefix builds both.
import { useParams } from "next/navigation";

export function hospitalPath(orgId: string, hospitalId: string): string {
  return `/o/${orgId}/h/${hospitalId}`;
}

/** The current page's hospital prefix, from the route's orgId and hospitalId. */
export function useHospitalPath(): string {
  const { orgId, hospitalId } = useParams<{ orgId: string; hospitalId: string }>();
  return hospitalPath(orgId, hospitalId);
}
