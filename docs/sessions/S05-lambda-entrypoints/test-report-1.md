# S05: test report 1

**22 passed, 0 failed, 0 blocked** (17 plan cases T-01 to T-17, 5 regression cases R-01 to R-05).

**Environment**
- Commit `6563286`. The working tree is not clean: the conductor deliberately left the S05 code and planning docs uncommitted, and this round tested that code.
- Mon 12 Oct 2026, 22:04–22:20 BST. Chrome (Claude in Chrome), automatic mode.
- `make dev` was already running (API on :8080, web on :3000), freshly migrated and seeded. `/api/health` returned `{"status":"ok"}`.
- I started the extra servers for T-11 to T-17 myself and stopped every one afterwards. Nothing I started is still listening on 8081–8085 or 3002–3003.

**Deviations from the plan (none affect the results)**
- **Port 3001 was taken** by an unrelated `next-server` that had been running since 6 Oct (not part of this repo's dev setup). I left it alone and ran T-12's second web app on **:3003** (`--port 3003 --strictPort`) instead.
- **Phone size (T-08, T-09, T-12):** `resize_window` reported success, but the window stayed 2560 px wide (it looks maximised). Instead I loaded the patient page in a same-origin **390×844 iframe**, so the app's own layout and media queries ran at 390 px. The page's `scrollWidth` was 390, so nothing was wider than the screen.
- **T-05:** I skipped the optional delete of the saved list, because the bin icon may open a confirm dialog. The list `S05 list 2210` is still there.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | Signed out (`/api/me` gave 401) and signed in as Alex, both returned exactly `{"locale":"en-GB","timeZone":"Europe/London"}` (`application/json`). There were no other keys. |
| T-02 | Pass | One `GET /api/config` (200) per page load, sent before `/api/dev/users`. Moving around inside the app (Questionnaires → editor → back; Episodes → episode → Episodes) sent no further `/api/config`. |
| T-03 | Pass | Showed "17 May 1980 (age 46)" and Created "12/10/2026, 22:08". The Episodes list showed "12/10/2026, 22:08". |
| T-04 | Pass | Rows read like "12/10/2026, 21:59 by Alex Author". |
| T-05 | Pass | "S05 list 2210": "3 options · Alex Author, 12/10/2026" (day first, no time). |
| T-06 | Pass | POA Summary showed Date of birth "17 May 1980", "Patient on 12/10/2026, 22:10" and "Validated by Cara Clinician on 12/10/2026, 22:11". General notes entries were day first with times. |
| T-07 | Pass | The deep link in a new tab, Reload and Back all kept the format. To look for a flash, I loaded the page in an iframe and checked its text every 2 ms: it only ever showed "17 May 1980 (age 46)" and "12/10/2026, 22:08". |
| T-08 | Pass | Signed out, at 390 px: "Hello Sam." appeared, with `/api/config` 200 and `/api/p/<token>` 200. No console errors and no horizontal overflow. |
| T-09 | Pass | At 390 px, both sets done → Send my answers → confirm (an in-page modal) showed "Thank you, Sam". As Cara, the episode was **Ready for review**. |
| T-10 | Pass | The stub login listed the 4 seeded users, and clicking Alex Author signed in. |
| T-11 | Pass | The server logged `staff-api listening on :8083 (dev login off, locale en-GB, time zone Europe/London)`. The three requests returned `404`, `404`, `401`. |
| T-12 | Pass | Ran `patient-api` on :8081 with the web app on **:3003**. At 390 px the HQ opened. I answered one question, and All sections showed **In progress**. After a reload the answer was kept. `/login` showed "Something went wrong. Please try again." |
| T-13 | Pass | `/api/health`, `/api/config` and `/api/p/<token>` returned 200. Every staff route, the dev login and logout returned 404. |
| T-14 | Pass | The four requests returned `404`, `404`, `200`, `200` (an extra check, PUT `/api/p/<token>/answers/x`, returned 404 too). On :3000 the S05 Split episode still showed **HQ not complete**. |
| T-15 | Pass | `/api/config` returned `{"locale":"en-US","timeZone":"America/New_York"}`. The episode showed "May 17, 1980 (age 46)" and Created "10/12/26, 10:08 PM". The Episodes list, POA Summary and Questionnaires were all month first. The same episode on :3000 stayed "17 May 1980" and "12/10/2026, 22:08". |
| T-16 | Pass | Without the settings: `config: not set: APP_URL, DEFAULT_TIME_ZONE, DEFAULT_LOCALE, PHONE_COUNTRY (CONFIG_STRICT is on, so there are no local defaults)`, exit 1. With a full set: `listening on :8084 (dev login off, locale en-CA, time zone America/Toronto)`, and `/api/config` returned the en-CA values. |
| T-17 | Pass | Three refusals, each naming its variable in plain words, for example `DEFAULT_TIME_ZONE must be an IANA time zone such as America/Toronto, not "Mars/Olympus"`. None of them started listening. |
| R-01 | Pass | Added a Select One question, changed its text to "MT S05 favourite colour" and reloaded: the text was kept. |
| R-02 | Pass | Published "MT S05 2210 Draft". It became read-only, with **Create new version** offered. |
| R-03 | Pass | Changed "Do you smoke?" from No to Yes and the "Patient answered: No" note appeared. I validated all 3 sets and clicked Complete HQ review: the episode became **Ready for POA**. |
| R-04 | Pass | Both tabs were shown. The Validated tab marked the change as "Changed by clinician" (Patient answered "No", now "Yes"). Setting the status to POA complete added a General notes entry. Print was enabled (I didn't print). |
| R-05 | Pass | The tab title and header say "Poised". The POA Summary URL opened straight there in a new tab. |

## Failures

None.

## Found while exploring

Nothing here is a failure. For information:

1. **A well-formed but unknown locale is accepted.** `DEFAULT_LOCALE=zz-ZZ` starts the server (`listening … locale zz-ZZ`), because it is a syntactically valid language tag. A browser given `zz-ZZ` silently falls back to its own default locale, so a typo of this kind wouldn't be caught at start-up. This is low risk, because each country's value is set once in infra config. The builder may want to decide whether that is enough.
2. **An empty `DEFAULT_LOCALE=` (strict mode off) falls back to the en-GB default** and starts. This looks intended for local dev. Strict mode would catch it on AWS (T-16).
3. **Other bad values are refused clearly:** `PHONE_COUNTRY=gbr`, `APP_URL=not-a-url` and `DEFAULT_TIME_ZONE=Local` each stop start-up with a plain message naming the variable.
4. **The config route by method:** POST, PUT and DELETE on `/api/config` return 405. `/api/config/` (trailing slash) returns 404. The response has no `Cache-Control` header. That's fine for now; just noting it, since the browser fetches config on every page load.
5. **If the API is down when the page loads** (web app on :3003 pointed at a dead port), `/api/config` returns 502. The app still renders: the login page shows "Something went wrong. Please try again.", and the patient page shows the same message. There is no blank screen or crash.
6. **A bad patient token** (`/p/not-a-real-token`) shows "This link doesn't work. Please check it, or contact the hospital." after `/api/config` 200 and `/api/p/…` 404.
7. **Keyboard:** I could Tab to the stub-login buttons and sign in with Enter (as Bea Author).
8. **Another hospital's URL:** Bea Author opening a Hospital A questionnaire URL sees "You don't have access to this hospital."
9. **Already signed in:** `/login` shows the stub login page without redirecting. This was already the behaviour before S05 and isn't in scope.
10. **A tool quirk, not an app bug:** a click sent straight after a page navigation sometimes didn't reach the page (no `POST /api/dev/login` was sent). Taking a screenshot first, then clicking, always worked.

## Console and network errors not tied to a case

- None. I saw no console errors or warnings on the patient pages, and none in the main tab once tracking started.
- Every 503 in the network log was on a 204 route (logout, dev login, publish), as the plan says.
- `/api/dev/users`, `/api/me` and other GETs appear twice per load. That is React StrictMode in dev, not a bug.

## GIFs

- `~/Downloads/S05-T03.gif`: the main happy path (create the episode → episode page → back to the list, with en-GB dates).
- `~/Downloads/S05-T15.gif`: the US-locale deployment on :3002 (sign in, month-first dates).

## Hand checks for Rahul

- None required.
- If you want a real-window phone check, open the patient link at 390×844 yourself. I used a 390×844 iframe because the browser window wouldn't resize.

## Test data created

- Questionnaire "MT S05 2210 Draft" (published v1).
- Option list "S05 list 2210".
- Episodes "Sam S05 Test" (POA complete) and "Sam S05 Split" (HQ not complete, one answer in progress).
