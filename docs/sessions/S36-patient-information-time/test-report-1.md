# S36 test report 1

**28 passed, 0 failed, 0 blocked** (21 cases + 6 regression checks + setup), automatic run.

- Commit: `896d0c7` (S36 work uncommitted in the tree: the builder's changes, `git status` not clean)
- Date: Mon 26 Oct 2026, 00:48–01:15 GMT
- URL: http://localhost:3000 (web) / :8080 (API), `make dev` run by the conductor
- Browser: Chrome ("trk" profile), desktop window 1512×771. Patient screens in a same-origin **390 × 844 iframe** (the window can't be resized). Staff users were switched with the dev login (the same call the stub login page makes).
- Test data: HQ **S36 Time HQ** (v1 published), leaflets **S36 Fasting leaflet**, **S36 Parking leaflet** (renamed **S36 Parking map** in R-06), **S36 Trust B leaflet**; patients Sam S36-One / S36-Two / S36-Three (Hospital A1), Sam S36-Val (Hospital A2), Sam S36-TrustB (Hospital B1); procedure **MT S36 0115 Bunion**; draft **MT S36 R04 info**.

## Results

| Case | Result | Note |
|---|---|---|
| Setup 1–4 | Pass | Hospital A1 phone and email saved; three leaflets added; S36 Time HQ built (4 Yes/No in About you; Nurse checks = Clinician, one Text). |
| T-01 Procedure gets a video and a leaflet | Pass | Picks listed in order with icon and Remove; picker empties after each pick; "Knee arthroscopy is saved."; column "1 video, 1 leaflet"; order kept on re-edit. |
| T-02 What the picker offers | Pass | 28 options: Preopsis videos and Trust A's active items, each "(Video)"/"(Leaflet)"; not the two in the list, not S36 Trust B leaflet, no retired item. "parking" narrows; "parkingzzqx" says "Nothing matches". |
| T-03 Remove then Cancel | Pass | After Cancel and Edit, both items are still listed. |
| T-04 Who may change it | Pass | Cara's page: "Only super clinicians and admins of the whole trust can manage its procedures". PATCH: **403** "Only super clinicians and admins of the whole trust can change the trust's procedures."; Alex still sees both items. |
| T-05 Editor's estimate | Pass | 4 questions: "About 1 minute"; 5: "About 2 minutes"; plus a clinical Text, a Clinical Summary and an Information block: still 2; plus the 10:35 library video: "About 12 minutes"; Nurse checks: no estimate. Published. The published version shows 12 too. |
| T-06 Patient tiles at 390 | Pass | "About 12 minutes" under About you. After finishing: whole **COMPLETE** badge and whole **Review** button, on a line below the name; scrollWidth 390 = clientWidth. |
| T-07 Patient's Information at 390 | Pass | The video (Length 10:35, `youtube-nocookie.com` player 324px wide), S36 Fasting leaflet link (`https://example.org/s36-fasting.pdf`, `target=_blank`, `rel=noopener noreferrer`), "Questions? Contact Hospital A1", `tel:01005550199`, `mailto:poa@hospital-a1.example`. No Parking leaflet. Pressing again closes it. No sideways scroll. |
| T-08 Add a leaflet to one episode | Pass | Added last, "Leaflet · Added for this patient by Cara Clinician"; the picker then says "Nothing matches" for "S36"; General notes "Information added: S36 Parking leaflet"; patient sees it after reload. |
| T-09 Another patient doesn't get it | Pass | S36-Two: video and Fasting leaflet only; Cara's card shows the two "From the procedure" items. |
| T-10 Procedure change leaves booked patients alone | Pass | Knee arthroscopy saved as "1 video". S36-One and S36-Two still see the Fasting leaflet; S36-Three sees only the video. |
| T-11 Retiring hides it from every patient | Pass | Retired: gone for S36-One and S36-Two; Cara's card shows it with a grey **RETIRED** badge; Total hip replacement's picker doesn't offer it; brought back: S36-One sees it again. |
| T-12 Remove from one episode | Pass | Gone from the card and from the patient's Information; General notes "Information removed: S36 Parking leaflet". |
| T-13 Changing the episode's procedure | Pass | After Edit procedure and dates → Laparoscopic cholecystectomy: "Example: Fasting before your operation" (From the procedure) plus S36 Parking leaflet (Added for this patient by Cara Clinician). The card updated without a reload. The patient sees the same. |
| T-14 No contacts, no items | Pass | Val's Hospital A2 episode with no procedure: "There are no videos or leaflets for this yet." and no contact section. |
| T-15 Cancelled episode | Pass | Card lists its item with no Remove and no picker; POST and DELETE answer 409 "This episode is cancelled, so it can't be changed."; patient's cancelled card has no Information button. |
| T-16 Trusts kept apart | Pass | Bea's picker: Preopsis items, B video 1840, S36 Trust B leaflet; no Trust A leaflet. The Trust B patient sees only S36 Trust B leaflet and no contact section. Alex's picker (T-02) never offered S36 Trust B leaflet. |
| T-17 Another person's Information | Pass | S36-One asking for S36-Two's: **404** "Episode not found."; `abc`: **404**; signed out: **401** "For your security, you've been signed out. Please open the link in your text message or email again." |
| T-18 Another hospital's staff | Pass | Val on Hospital A1's episode page: "You don't have access to this hospital"; `/api/o/…/h/<A1>/…/information` **404** "Hospital not found."; through Hospital A2's path GET, POST and DELETE all **404** "Episode not found." |
| T-19 Bad input | Pass | `abc`: **400** "Choose a video or leaflet from the content library."; the video already there: **409** "That video or leaflet is already in this episode's Information." |
| T-20 Reload and back | Pass | Reload closes the list; Continue, then browser Back: the home, and Information opens; Cara's page reloaded shows the same items. |
| T-21 Audit log | Pass | Rows for "Added Information to an episode", "Removed Information from an episode", "Opened an episode's Information" (Cara), "Patient opened their Information", "Changed a procedure" (Alex; Cara's attempt as REFUSED (403)). Refused adds/removes on the cancelled episode show REFUSED (409). No item titles in any row. |
| R-01 Send my answers | Pass | In-app "Send your answers?" dialog, then "Thank you, Sam". The home card says SENT and still has Information. |
| R-02 Validate and POA Summary | Pass | Patient answers shown in the validation view with the clinician-only Text question and the Page 1 summary; validated; POA Summary shows "Clinical comments: MT S36 summary comment". |
| R-03 Video in a set at 390 | Pass | Player and "Length 10:35" inside About you at 390px. |
| R-04 Information block formatting | Pass | `## Heading` and `- a list` show as a heading and bullets in the editor and in Preview. |
| R-05 Procedure add, retire, bring back | Pass | MT S36 0115 Bunion added, retired, brought back; Information "—". |
| R-06 Library edit reaches episodes | Pass | Renamed to "S36 Parking map": shown on S36-Two's episode page and in S36-Two's Information. |

## Failures

None.

## Found while exploring

Nothing broke. Notes for the builder, smallest first:

1. **Minor (wording, staff):** the Content library's intro still says "A retired item isn't offered any more, but anything that already uses it keeps its copy." For Information that's no longer true: retiring an item now **hides it from every patient's Information** (T-11, by design). An admin reading this line would expect booked patients to keep it. Suggest the line mention Information.
2. **Cosmetic:** a Question Set with **no questions** (empty, or only an Information block) still says "About 1 minute" in the editor and would on the patient's tile. This matches the "at least 1" rule in the build notes, but "About 1 minute" for a set with nothing to answer may read oddly.
3. **Cosmetic:** the cancelled card on the patient's home says "Please contact them if you have any questions", but the hospital's phone and email are only under Information, which a cancelled card doesn't offer. So the patient has no way to see the contact details there.
4. **Cosmetic:** on a cancelled episode, the Tasks card says "This episode is cancelled, so its tasks can't be changed." The Information card just loses its controls without saying why, and still says "changes here are for this patient only".
5. Checked and fine:
   - **Three parallel POSTs** of the same item gave one 204 and two 409s, with no duplicate row.
   - **Removing the same item twice** gave 404 "That video or leaflet isn't in this episode's Information."
   - **An empty body** gave 400. **An unknown uuid** gave 400. **Malformed JSON** gave 400 "The request could not be read."
   - **Sue Super** (a hospital-scoped super clinician) changing a procedure's list: 403.
   - **Trust B with Trust A's items:**
     - As Bea, adding Trust A's leaflet to a Trust B episode gave 400.
     - As Bea, putting Trust A's leaflet in a Trust B procedure's list gave 400 "Information: choose videos and leaflets from the trust's content library."
     - Trust B's episode `library` lists no Trust A item.
   - **A retired item kept in a procedure's list:**
     - The form shows it with a RETIRED badge.
     - The procedure still saves with another item added.
     - I put it there with an API PATCH as Alex. Then I cleared the list and brought the leaflet back.
   - **The pickers work by keyboard:** type, Down, Enter.
6. **The patient's link stops working once the episode is cancelled** ("This link doesn't work"). To check the cancelled card I signed S36-Three in through `/signin`, with an email sign-in link. That's expected (the cancel panel says the patient's link stops working); I note it only so it's clear how I reached T-15 step 2.

## Console and network errors

None seen. I checked the console with `error|Error|warn` on the staff tab and the patient tab, and saw no unexpected 4xx/5xx on the episode page. Every 4xx was one a case or a probe asked for.

## GIFs

- `/Users/rahul/Downloads/S36-T01.gif`: the procedure gets a video and a leaflet, then saves.
- `/Users/rahul/Downloads/S36-T07.gif`: the patient's Information at 390px, opened and closed.

## Hand checks for Rahul

None from this round. The build notes' own hand checks still apply:
- a real video plays on a real phone;
- the tel and mailto links open the phone and mail apps;
- `SECONDS_PER_QUESTION` set to another value changes every estimate.
