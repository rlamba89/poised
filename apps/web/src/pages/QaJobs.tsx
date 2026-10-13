import { ScheduledJobs } from "@/features/qa/ScheduledJobs";

// Run the scheduled jobs as of a later time, or move the local timer's clock (MESSAGING=capture:
// local and QA only, S16).
export function QaJobs() {
  return <ScheduledJobs />;
}
