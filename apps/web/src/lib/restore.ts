// A page the browser restores from its back/forward cache (Back after leaving it) shows exactly
// what it showed before, without asking the API: e.g. the patient list of a clinician who has
// since signed out, on a shared workstation (S12 test report 1, finding 1). Reloading it asks the
// API again, which sends a signed-out browser to the sign-in page. main.tsx installs it.
type Win = {
  addEventListener: (type: "pageshow", handler: (e: { persisted: boolean }) => void) => void;
  location: { reload: () => void };
};

export function reloadWhenRestored(win: Win) {
  win.addEventListener("pageshow", (e) => {
    if (e.persisted) win.location.reload();
  });
}
