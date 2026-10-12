# S05: manual test plan

**What S05 changed:**
- The Go API is split into a **staff** part and a **patient** part, which will run as separate functions on AWS. Locally, `make dev` still runs both together.
- The **dev login** (the stub sign-in page) only works when it's switched on. `make dev` switches it on.
- The server reads its **per-country settings** (locale, time zone, web address, phone country) from its environment, with UK defaults for local dev. A new page, `GET /api/config`, tells the browser the locale and time zone.
- **Dates** in the web app are now formatted in the locale from `/api/config`, not a fixed UK format. With the local defaults they look exactly as before.

Nothing else should look different.

## Setup

**Rahul (or the conductor) runs, in his own terminal, from the repo root:**

```sh
docker compose up -d db && make migrate seed && make dev
```

Leave `make dev` running. **Open http://localhost:3000.**

**Sign in** on the stub login page at http://localhost:3000/login by clicking a name:

| User | Can do |
| --- | --- |
| Alex Author | author + publisher, Hospital A |
| Val Viewer | viewer, Hospital A |
| Cara Clinician | clinician, Hospital A (episodes, validating, POA Summary) |
| Bea Author | author, Hospital B |

**The extra servers (T-11 to T-16).** These cases ask you to start a second, short-lived server in a terminal, on a port that doesn't clash with `make dev` (3000 and 8080).
- Run each in the background from the repo root, wait until it logs `listening on`, test, then **stop it** with `lsof -ti :<port> | xargs kill` (`kill %1` can leave the compiled server behind `go run` running).
- Never stop `make dev` itself.
- The first start of a Go server takes a few seconds while it compiles.
- The commands that start with `(set -a; . ./.env; set +a; …)` load the repo's `.env` first; run them exactly as written.
- **Cookies are shared across ports on `localhost`.** Signing in on :3002 also signs you in on :3000 as the same user. That's expected, not a bug.

**Browser notes:** the Claude in Chrome network log shows a `204` as `503`. Ignore a 503 on logout or other 204 routes.

## Test cases

### The config route, and dates in the local default (en-GB)

**T-01: `GET /api/config` returns the public settings only**
1. Open http://localhost:3000/api/config in a tab (no sign-in needed; try it signed out).
2. **Expected:** exactly `{"locale":"en-GB","timeZone":"Europe/London"}`. No other keys: no web address, phone country, database URL or secret.
3. Sign in as Alex Author, then open the same URL again. **Expected:** the same answer.

**T-02: the app reads the config once, before the page shows**
1. Signed out, open http://localhost:3000/login and watch the network log.
2. Reload.
3. **Expected:** one `GET /api/config` (200) per page load, sent before or alongside `GET /api/dev/users`.
4. Click around a few pages (Questionnaires, Episodes). **Expected:** no further `/api/config` requests while moving between pages inside the app.

**T-03: the date of birth and "Created" on a new episode**
1. Sign in as **Cara Clinician**. Go to **Episodes** → **New episode** → **New patient**.
2. Fill in: First name `Sam`, Last name `S05 Test`, Date of birth `1980-05-17`, Sex `Female`, any Health questionnaire, Procedure `Knee replacement`. Click **Create episode**.
3. **Expected:** the episode page shows **Date of birth: 17 May 1980 (age 46)**, and **Created** as a day-first date and 24-hour time, e.g. `12/10/2026, 22:05` (today's date, your local time).
4. Go back to the **Episodes** list. **Expected:** the new episode's Created column has the same day-first format.

**T-04: dates on the questionnaire list**
1. As **Alex Author**, open **Questionnaires**.
2. **Expected:** each row's "updated" column reads like `12/10/2026, 21:40 by Alex Author` (day first, 24-hour time).

**T-05: the date on a saved option list**
1. As **Alex Author**, open a **draft** questionnaire (create one if there is none: **New questionnaire**, any name). Open a Question Set and add or select a **choice question** (e.g. Radio group) with options.
2. In its options, click **Save as a list**, name it `S05 list`, save.
3. On the same or another choice question, click **Use a saved list**.
4. **Expected:** `S05 list` shows "N options · Alex Author, 12/10/2026" (today, day first, no time).
5. Delete `S05 list` afterwards with its bin icon (optional).

**T-06: dates on the POA Summary and General notes**
1. As **Cara Clinician**, open an episode that has a submitted HQ (or use the episode from T-03 after the patient sends it in T-09). Open **POA Summary**.
2. **Expected:** Date of birth as `17 May 1980` style (day, month name, year). "Patient on …" and "Validated by … on …" show day-first dates with 24-hour times.
3. On the episode page, **General notes** entries ("Episode created" and so on) show day-first dates with times.

**T-07: deep link and reload keep the format**
1. On the T-03 episode page, copy the URL. Open it in a **new tab**, and also press **Reload** on it.
2. **Expected:** the page loads straight onto the episode; the date of birth and Created show in the T-03 format at once (no flash of another format, e.g. "May 17, 1980").
3. Use the browser **Back** button to the Episodes list. **Expected:** the dates are still day-first.

### The patient's link (phone size, 390×844)

**T-08: the patient page loads its config without signing in**
1. On the T-03 episode, click **Copy** next to the patient link.
2. Open the link in a **signed-out** browser window sized **390×844**.
3. **Expected:** the HQ opens ("Hello Sam."), and the network log shows `GET /api/config` 200 and `GET /api/p/<token>` 200. No console errors. Nothing is wider than the screen.

**T-09: the patient completes and sends the HQ (phone size)**
1. Still at 390×844, answer every patient Question Set and tap **Send my answers** → confirm.
2. **Expected:** "Thank you, Sam". Back as Cara, the episode is **Ready for review**.

### The dev login switch

**T-10: the dev login works under `make dev`**
1. Sign out. Open http://localhost:3000/login.
2. **Expected:** the stub login lists the seeded users, and clicking **Alex Author** signs in.

**T-11: with `DEV_LOGIN` off, the stub sign-in doesn't exist**
1. From the repo root, start the staff part alone **without** the dev login:
   ```sh
   (set -a; . ./.env; set +a; cd apps/api && API_ADDR=:8083 go run ./cmd/staff-api) &
   ```
2. **Expected:** it logs `staff-api listening on :8083 (dev login off, locale en-GB, time zone Europe/London)`.
3. Run:
   ```sh
   curl -s -o /dev/null -w '%{http_code}\n' localhost:8083/api/dev/users
   curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:8083/api/dev/login -d '{}'
   curl -s -o /dev/null -w '%{http_code}\n' localhost:8083/api/me
   ```
4. **Expected:** `404`, `404`, then `401` (the rest of the staff API is there, waiting for a sign-in).
5. Stop it.

### The router split: each part alone, as on AWS

**T-12: the patient part alone serves the patient's link, in the browser**
1. Get a patient link for an episode that **hasn't been sent yet**: as Cara, create another episode (as in T-03, Last name `S05 Split`) and **Copy** its link. Note the token, the part after `/p/`.
2. Start the patient part alone, and a second web app that talks only to it:
   ```sh
   make patient-api ADDR=:8081 &
   API_URL=http://localhost:8081 npm run dev -w apps/web -- --port 3001 &
   ```
   Wait for `patient-api listening on :8081` and Vite's `Local: http://localhost:3001/`.
3. At **390×844**, open `http://localhost:3001/p/<token>`.
4. **Expected:** the HQ opens and works as on :3000. Start a Question Set, answer one question, and go back to **All sections**: it shows **In progress**. Reload: the answer is kept.
5. Open `http://localhost:3001/login`. **Expected:** the login page can't list users and shows an error message. Only the patient's routes exist on this server.

**T-13: the patient part alone refuses every staff route and the dev login**
With T-12's `patient-api` still on :8081 (use a hospital ID from any `/h/<id>/…` URL on :3000):
```sh
for p in /api/health /api/config /api/p/<token> /api/me /api/codes?q=asthma /api/categories /api/dev/users /api/h/<hospital-id>/episodes /api/h/<hospital-id>/questionnaires; do echo "$p $(curl -s -o /dev/null -w '%{http_code}' "localhost:8081$p")"; done
curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:8081/api/dev/login -d '{}'
curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:8081/api/logout
```
**Expected:** `/api/health`, `/api/config` and `/api/p/<token>` are **200**. Every other path, the dev login POST and logout are **404**.
Then stop both T-12 servers.

**T-14: the staff part alone refuses the patient's link**
1. Start it: `make staff-api ADDR=:8082 &`, and wait for `staff-api listening on :8082 (dev login on, …)`.
2. Run, with T-12's token:
   ```sh
   curl -s -o /dev/null -w '%{http_code}\n' localhost:8082/api/p/<token>
   curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:8082/api/p/<token>/submit
   curl -s -o /dev/null -w '%{http_code}\n' localhost:8082/api/dev/users
   curl -s -o /dev/null -w '%{http_code}\n' localhost:8082/api/config
   ```
3. **Expected:** `404`, `404`, `200`, `200`.
4. As Cara on :3000, open the T-12 episode. **Expected:** still **HQ not complete** (the submit above did nothing).
5. Stop it.

### Per-country settings

**T-15: another country's locale changes the dates (browser)**
1. Start the staff part as if it were a US deployment, plus a web app that talks to it:
   ```sh
   DEFAULT_LOCALE=en-US DEFAULT_TIME_ZONE=America/New_York make staff-api ADDR=:8085 &
   API_URL=http://localhost:8085 npm run dev -w apps/web -- --port 3002 &
   ```
   Wait for `staff-api listening on :8085 (dev login on, locale en-US, time zone America/New_York)`.
2. Open http://localhost:3002/api/config. **Expected:** `{"locale":"en-US","timeZone":"America/New_York"}`.
3. Open http://localhost:3002/login, sign in as **Cara Clinician**, and open the T-03 episode.
4. **Expected:** **Date of birth: May 17, 1980 (age 46)**, and Created month-first with a 12-hour time, e.g. `10/12/26, 10:05 PM`. The Episodes list is month-first too.
5. As **Alex Author** on :3002, **Questionnaires** dates are month-first too.
6. Open the same episode on :3000. **Expected:** still `17 May 1980` and day-first (each deployment has its own locale).
7. Stop both servers.

**T-16: strict mode refuses to start without the country settings**
1. Run (it should exit by itself within seconds):
   ```sh
   (set -a; . ./.env; set +a; cd apps/api && API_ADDR=:8084 CONFIG_STRICT=true go run ./cmd/patient-api)
   ```
2. **Expected:** it **doesn't** start. It prints one error naming all four: `APP_URL, DEFAULT_TIME_ZONE, DEFAULT_LOCALE, PHONE_COUNTRY`, saying CONFIG_STRICT is on, and exits.
3. Run it again with a full set:
   ```sh
   (set -a; . ./.env; set +a; cd apps/api && API_ADDR=:8084 CONFIG_STRICT=true APP_URL=https://ca.example.test DEFAULT_TIME_ZONE=America/Toronto DEFAULT_LOCALE=en-CA PHONE_COUNTRY=CA go run ./cmd/patient-api) &
   ```
4. **Expected:** `patient-api listening on :8084 (dev login off, locale en-CA, time zone America/Toronto)`. `curl -s localhost:8084/api/config` gives `{"locale":"en-CA","timeZone":"America/Toronto"}`. Stop it.

**T-17: a bad setting stops start-up, even locally**
1. Run each; each should exit with an error naming the variable:
   ```sh
   (set -a; . ./.env; set +a; cd apps/api && API_ADDR=:8084 DEFAULT_TIME_ZONE=Mars/Olympus go run ./cmd/patient-api)
   (set -a; . ./.env; set +a; cd apps/api && API_ADDR=:8084 DEFAULT_LOCALE=english go run ./cmd/patient-api)
   (set -a; . ./.env; set +a; cd apps/api && API_ADDR=:8084 DEV_LOGIN=yes-please go run ./cmd/patient-api)
   ```
2. **Expected:** three refusals, naming `DEFAULT_TIME_ZONE`, `DEFAULT_LOCALE` and `DEV_LOGIN` in turn, in plain words. None of them starts listening.

## Regression (earlier sessions)

**R-01 (J1, S03): the author edits a Question Set.** As Alex Author, open a draft questionnaire, add a question, change its text and save. Reload: the change is kept.

**R-02 (J2): publish.** Publish a draft that passes the checks. It becomes read-only, with **New version** offered.

**R-03 (J4): the clinician validates.** As Cara, on a **Ready for review** episode (e.g. T-09's), change one answer in a Question Set (a "Patient answered…" note appears) and **Validate** each set. **Complete review**: the episode becomes **Ready for POA**.

**R-04 (J5): POA Summary.** Both tabs show; the corrected answer is marked "Changed by clinician" on the validated tab. Change the status: General notes records it. The **Print** button is enabled (don't print).

**R-05 (S04): the app is "Poised", and a deep link to a page works.** The tab title and header say Poised. Paste an episode's POA Summary URL into a new tab: it opens straight there.

## Out of scope (don't report)

- Anything on AWS: Lambda, SSM, `make build-lambdas` (S06).
- **Times are shown in your browser's time zone,** not the deployment's (T-15 shows a US date format but your local clock time). Hospital time zones come in S08.
- The plain-text `404 page not found` (rather than a JSON message) for a route a part doesn't serve. It's deliberate.
- The login page on :3001 (T-12) failing to list users: that's the point of the split.
- The known items in `docs/pending.md` §3 (e.g. the POA Summary listing a change to a question a later change hid).
- The real print dialog.
