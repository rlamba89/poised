# S34 test report 1: the Video element

**24 passed, 0 failed, 0 blocked** (plus hand checks for Rahul, below).

- **Environment:** `git rev-parse --short HEAD` = `e890ab4`; working tree **not clean** (S34's uncommitted changes, as built). 25 Oct 2026, 20:42–21:25. http://localhost:3000 (`make dev`, run by the conductor; API :8080 healthy).
- **Browser:** Claude in Chrome ("trk" profile). Phone-size cases ran in a same-origin 390 px iframe (inner width checked at exactly 390 px).
- **Automatic mode.** Tester: Claude (manual-test agent).
- **Test data made:** questionnaires "MT S34 test 2042" (v1 to v4) and "MT S34 novideo 2042" (v1, v2); made-up patients Mtone, Mttwo, Mtthree and Mtfour Vidtest, and Mtfive Novideo (all born 17/05/1980, `mt.s34.*@example.com`); trust video "MT S34 2042 retire video" (tag MTS34TAG; retired again at the end).

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Add a Video | Pass | The card says "Video", PATIENT, and red "No link yet · No length yet". Settings has only the Settings and Logic tabs. It shows Title, Link (red hint), Choose from library, Length (red) and the tick switch (off), with no Required, Always ask fresh, Same as… or Save to library. Retitling worked and the bar said "All changes saved". |
| T-02 YouTube link | Pass | It said "Plays from YouTube (privacy-enhanced mode)" and the card "YouTube · Length 10:35" in grey. In Preview: title, "Length 10:35", a player and no tick. The iframe is exactly `https://www.youtube-nocookie.com/embed/aqz-KE-bpKQ` (referrerPolicy `strict-origin-when-cross-origin`, allowfullscreen). Back returned to the editor. Playback was proved on the patient's phone page (T-12): it played and showed CC and full-screen buttons. |
| T-03 youtu.be link | Pass | Same message. The Preview iframe was again `…youtube-nocookie.com/embed/aqz-KE-bpKQ`. |
| T-04 Vimeo links | Pass | "Plays from Vimeo (no tracking)" and "Vimeo · Length 10:35". The iframe was `https://player.vimeo.com/video/76979871?dnt=1`. With the private link it was `…/76979871?dnt=1&h=8272103f6e` (checked by parsing). |
| T-05 Refused links and Publish | Pass | All five refused links showed the red message, and the card read "Not a YouTube or Vimeo link · Length 10:35". Each Preview had **0 iframes** and showed "This video can't be shown." Publish listed "Question Set 1 · The video "Your anaesthetic" has a link that isn't a YouTube or Vimeo video" with Publish disabled. An empty link listed "…has no link: paste a YouTube or Vimeo link, or choose one from the library". |
| T-06 Length | Pass | `3:2` gave the red message and the card kept 10:35. `4` became 4:00 and `1:02:05` became 1:02:05. `5:00:00` was refused. Empty showed "No length yet" in red, and Publish listed "…has no length". |
| T-07 Choose from library | Pass | Window "Choose a video from the library". Tags (Example, Fasting, Hip, Knee replacement, Prehab, Recovery, The anaesthetic) showed whole. Videos only, each with its length and "Preopsis" or "Your trust". The tag The anaesthetic left one video, and pressing it again restored all. `fasting` left one. Use this filled the title, link and 10:35. Reopened (after a tag and a search), the window started afresh. |
| T-08 Tick and Required | Pass | Each step behaved as the plan says. Turning the tick off also turned Required off. "(optional)" shows on the library video's card. Published as "v1 · published": Settings greyed out, no Choose from library. |
| T-09 Copy kept after retiring | Pass (adapted) | See the test-plan note below. As Ada I added my own trust video and Alex picked it into a v2 draft. When Ada retired it, the draft still showed it, and **Choose from library no longer offered it** (nor its tag). After Bring back it was offered again. The published v1 kept "Example: Fasting…". |
| T-10 No video in a repeating group | Pass | Inside the repeating group, **Video** (and Clinical Summary) are greyed out. Before the group was made repeatable, "Move into" offered it. Afterwards the video card's More actions button isn't shown at all (no target is left). |
| T-11 Translations | Pass | German listed the video title (TEXT) and "I have watched this video" (TICK). In Preview (German): "Ihre Narkose *" and "Ich habe dieses Video gesehen". "Length" and "Complete" stay in English (app labels, not S34's). |
| T-12 Required tick holds the set (phone) | Pass | "Your anaesthetic *", Length 10:35, and the player at 290×163 (16:9) inside the card. `scrollWidth` 388/390, so no sideways scroll. Finish without the tick kept the patient on the page with the red "Please tick "I have watched this video" to go on." Tapping the words ticked it. The optional tick wasn't needed. The video **played in the page** (frames advanced; CC and full-screen buttons visible). Sent. GIF recorded. |
| T-13 Untick and reload | Pass | Ticked, then "Saved". After a reload the box was still ticked. Unticked + Finish gave the same message. The set list then showed "In progress" with Send disabled. Ticked again, finished and sent. |
| T-14 Optional tick never holds | Pass | Done on v2, answering No: the only videos shown had optional ticks, and the patient finished and sent without ticking. |
| T-15 Validate (read-only tick) | Pass | Both players are shown. The required tick is ticked and disabled with "The patient ticked this.", and the optional one is unticked and disabled with "The patient hasn't ticked this." Clicking the disabled box didn't change it. Validate worked. The POA Summary (both tabs) has no video, tick or video title. |
| T-16 Second HQ doesn't pre-fill the tick | Pass | Same patient, new episode: the "Please check" card said "1 answer". "Do you smoke?" = No with "From your answers on 25 Oct 2026". **Both ticks unticked.** |
| T-17 Fill in for the patient | Pass | Videos and ticks show and the ticks are enabled for staff. Finish without the tick was refused with the same message. Staff ticked it, finished, sent: "Sent for Mttwo Vidtest". |
| T-18 Who can change a video | Pass | For Cara (published v1) and for Sue (v2 draft, "READ ONLY"), every video setting is disabled and there's no Choose from library. |
| T-19 Video behind a condition | Pass | Logic: shown when "Do you smoke? is Yes" (videos aren't offered as condition sources). Answering No hid the video, and the patient finished and sent without the tick. Answering Yes showed it with *, Finish was refused until ticked, then the patient sent. |
| R-01 Clinical Summary (S32) | Pass | It adds inside a Group and on the page. Two in one Group block Publish: "Two clinical summaries cover the same Group "Group": keep one". |
| R-02 Content library (S33) | Pass | As Ada, a Dailymotion link is refused ("Only YouTube or Vimeo links can be used for a video…"). A youtu.be link was added, length "10:35". |
| R-03 Second HQ pre-fill (S25) | Pass | Covered by T-16. |
| R-04 Always ask fresh / Save to library | Pass | A Text question (and Yes / No) still offers Same as…, Save to library and Always ask fresh. |
| R-05 No-video HQ at phone size | Pass | "MT S34 novideo 2042": the patient answered, finished and sent at 390 px as before. |

**Test-plan note (T-09 step 1):** the plan asks Ada to retire "Example: Fasting before your operation". That item is Preopsis's, and the library lists it under "From Preopsis (read-only)" with no Retire button, so step 1 can't be done as written. I used my own trust video instead, and every expected result was checked on it.

## Failures

None.

## Hand checks for Rahul

- **Full screen on a phone:** the full-screen button is shown, but my 390 px harness iframe has no `allowfullscreen`, so it can't be tested here. Please check on a real iPhone (Safari) and Android.
- **Vimeo playback:** the Vimeo player loaded in Preview (frame and controls showed), but I only pressed play on YouTube. YouTube played in the page.
- **QA's `Referrer-Policy: no-referrer`:** a real video plays on QA (from the builder's notes; it can't be checked locally).

## Found while exploring

1. **Patient set list cut off at 390 px. This predates S34, but patients see it.** On the patient's questionnaire home, the status badge is cut short: "PLEASE CHE…" (the pre-fill "Please check") and "IN PROGR…". The **Continue** button's label is also clipped ("Continu"). This happens because the set name "Question Set 1" wraps to two lines. It was measured at exactly 390 px (badge `scrollWidth` 89 > `clientWidth` 86, with `text-overflow: ellipsis`). S34 changed `PatientHQ.tsx` by one import only, so this isn't caused by S34. Minor/cosmetic. Earlier sessions have failed on cut-off text, so the builder may want to look.
2. **The required tick has no marker of its own.** The red asterisk sits on the video title. Before the patient taps Finish, nothing on the tick says it's needed, and the patient side shows no "(optional)" on optional ticks (the editor card does). This is a suggestion, not a fail.
3. **Start times are dropped.** `…watch?v=…&t=42s` (and `m.youtube.com`, and leading/trailing spaces) is accepted and trimmed, but the player starts at 0:00 because the address is built from the ID only. This is by design. An author might expect the time to be kept.
4. **YouTube Shorts links** (`youtube.com/shorts/<id>`) aren't in the agreed link list, so they'd be refused. This is a product question.
5. **Length parsing:** `0:00` and `1:60` are refused, and ` 75:00 ` becomes 1:15:00. Good.
6. **Server check confirmed (an in-page API call as the patient, to check what the UI can't reach).** I saved answers without the required tick, and `POST …/submit` returned **409** "Please tick "I have watched this video" for "Your anaesthetic" before sending your answers." The tick was put back afterwards.
7. **Role and tenancy (an in-page API call).** Sue gets **403** from `GET /api/o/<Trust A>/content/search`. Bea (Trust B) gets **404** for Trust A, and Trust B's own search contains no Trust A items.
8. **Small text mismatches, not S34's:** the delete confirmation for a Clinical Summary titled "Group summary" says "“Clinical summary” will be deleted". The page header in a patient Question Set reads "Clinical Page". Both cosmetic.
9. A patient's set shows COMPLETE after a reload once every required answer is in, without tapping Finish. This is existing behaviour, noted only.

## Console and network errors

- No console errors or warnings in the staff tabs or the preview.
- The extension recorded every `POST …/publish` as **503**, but the page's own `fetch` got **204** each time (checked by wrapping `fetch`; a direct call also gave 204 in 116 ms), and each publish succeeded. This is a quirk of how the extension records requests, not an app error.
- No other unexpected 4xx or 5xx.

## GIFs

- `~/Downloads/S34-T12-happy-path.gif`: the patient at 390 px is refused at Finish until the required tick, then ticks it, plays the video, finishes and sends.
