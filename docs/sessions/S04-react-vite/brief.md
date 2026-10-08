# S04: Frontend on React + Vite + React Router

**Plan:** F2 ([f2-react-vite.md](../../plans/f2-react-vite.md)), all of it.
**Depends on:** S03 verified. J1–J5 are the proof.

**Ask Rahul first:**
- Rename the UI branding from "Lifebox" to "Poised" in this session?
  - The places are listed in decisions §5.
  - **Recommended: yes,** because the layouts are being rewritten anyway. It's the only visible change allowed.

## Build

- F2 Steps 1–4:
  - the Vite scaffold;
  - the same routes in `src/routes.tsx`;
  - the Next.js imports replaced;
  - Next.js removed;
  - `apps/web/CLAUDE.md` and `AGENTS.md` updated, and the root README.
- **No other behaviour or URL changes.**

## Not in this session

- new pages;
- Amplify (S06);
- the `/o/…` URLs (S08).

## Done when

- `make lint test` pass.
- **J1–J5 pass unchanged, 3 runs in a row.** Any selector change needs a reason.
- After `vite build`, `vite preview` reloads a deep link correctly.
- The bundle size is noted in the F2 Status section.

## Manual test focus

This is the biggest regression of all, because every screen was rewritten. Test at desktop size unless a screen is for patients:
- the login page;
- the questionnaires list;
- **the editor:** Structure tree, Settings, Logic builder, Disclosures and test cases;
- preview;
- publish and versions;
- the episodes list and an episode;
- **the patient link at 390×844;**
- the validate screens;
- the POA Summary, both tabs, and the Print button;
- **reloading each deep link;**
- the back button;
- the not-found page;
- console errors on every screen.
