"use client";
import { useParams } from "next/navigation";
import { StaffPage } from "@/features/staff/StaffPage";

export default function Page() {
  const { orgId, hospitalId } = useParams<{ orgId: string; hospitalId: string }>();
  return <StaffPage orgId={orgId} hospitalId={hospitalId} />;
}
