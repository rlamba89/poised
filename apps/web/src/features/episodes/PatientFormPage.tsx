// Staff filling in the patient's HQ on their behalf, e.g. on the phone (PAT-09, S14): the
// patient's own form, through …/episodes/<id>/patient-form. The answers are the patient's, marked
// "entered by" the staff member on the episode, the validation screens and the POA Summary. The
// form draws its own "← Episode" link, which saves the answer just typed before leaving.
import { Stack } from "@mantine/core";
import PatientHQ from "@/features/patient/PatientHQ";

export function PatientFormPage({ base, episodeId }: { base: string; episodeId: string }) {
  const back = `${base}/episodes/${episodeId}`;
  return (
    <Stack maw={960} mx="auto">
      <PatientHQ path={`${back}/patient-form`} onBehalf={{ back }} />
    </Stack>
  );
}
