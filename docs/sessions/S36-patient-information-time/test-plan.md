# S36 manual test plan: the patient's Information, and the estimated time

For a tester who has never seen the code. Use only the browser. Report each case as Pass, Fail or Blocked, with what you saw.

**What's new:**
- **Information for patients.** Each procedure in the trust's list can have videos and leaflets from the content library. When an episode is made for that procedure, **the list is copied onto the episode** (a booked patient keeps their list; changing the procedure's list later changes only new episodes). Staff can add a video or leaflet to one episode, or remove any. The patient sees the list under an **Information** button on the questionnaire's card on their home page, with the hospital's phone number and email. A video or leaflet **retired** in the content library disappears from every patient's Information.
- **The estimated time.** Each of the patient's Question Sets says "About N minutes": 15 seconds for each question the patient may see, plus each video's length, rounded up to whole minutes (at least 1). Clinician-only questions, Clinical Summaries, Information blocks and calculations don't count. The editor shows the same figure on the Question Set's page. (15 seconds is the server's setting; you can't change it in the browser.)
- The patient's Question Set tiles no longer cut the status badge or button short on a phone.

## 1. Setup

1. In your own terminal (an automatic run does this for you):
   ```sh
   docker compose up -d db && make migrate seed && make dev
   ```
   Leave `make dev` running.
2. Open **http://localhost:3000** in Chrome. Sign in from the stub login page by clicking a user's name (sign out from the avatar menu to switch):
   - **Ada Admin** (admin, the whole of Trust A): Settings, Content library, Audit.
   - **Alex Author** (super clinician, the whole of Trust A): Procedures, questionnaires.
   - **Cara Clinician** (clinician, Hospital A1 only): episodes.
   - **Val Clinician** (clinician, Hospital A2 only).
   - **Bea Author** (super clinician, the whole of Trust B), **Max Multi** (admin of the whole of Trust B; clinician at Hospital A2).
3. **Captured messages:** http://localhost:3000/qa/messages lists the invite texts and emails (nothing is really sent). Each patient's invite link is in there (search by their email).
4. **Phone size:** for every patient case use Chrome's device toolbar at **390 × 844**.
5. **Videos:** the example videos all play the same public test film ("Big Buck Bunny", 10:35). Playing one is optional.
6. **Preparation (do once, in order):**
   1. As **Ada Admin**: **Settings** → Hospital A1's settings → set **Phone number for patients** to `0100 555 0199` and **Email address for patients** to `poa@hospital-a1.example` → Save Hospital A1's settings. Leave Hospital A2's empty.
   2. As **Ada Admin**: **Content library** → add two **leaflets** (Add item → Leaflet): `S36 Fasting leaflet` with link `https://example.org/s36-fasting.pdf`, and `S36 Parking leaflet` with link `https://example.org/s36-parking.pdf`.
   3. As **Max Multi** (choose Trust B's Hospital B1): **Content library** → add a leaflet `S36 Trust B leaflet`, link `https://example.org/s36-trust-b.pdf`.
   4. As **Alex Author** (Hospital A1): make a questionnaire **"S36 Time HQ"** with two Question Sets (you'll grow the first in T-05):
      - **About you** (Who is it for? **Patient**): one page with **four Yes / No** questions (any wording);
      - **Nurse checks** (Who is it for? **Clinician**): one Text question.
      Don't publish yet.

**Making an episode for a patient** (used below): as **Cara Clinician** at Hospital A1 → **New episode** → enter a **new patient** (first name `Sam`, a surname you'll recognise, date of birth `17/05/1980`, sex any, a made-up email such as `s36.a.<time>@example.org`) → **Procedure**: as the case says → **then choose the HQ "S36 Time HQ"** (picking a procedure picks its own default HQ first) → Create. Then open the captured email's invite link at 390 × 844, press **Continue**, enter the date of birth, Continue: you land on the questionnaire. Its **← Your questionnaires** link goes to the patient's home.

## 2. Test cases

### The procedure's Information

**T-01 Give a procedure a video and a leaflet** (whole-trust super clinician)
1. As **Alex Author**, open **Procedures**. The table has an **Information** column (— for every procedure).
2. **Edit** "Knee arthroscopy". Below the fields, **Information for patients** says "None yet." with an **Add a video or leaflet** picker.
3. Open the picker, type `anaesthetic`, choose **Example: Your anaesthetic explained (Video)**. Then choose **S36 Fasting leaflet (Leaflet)**.
4. **Save changes**.

Expected: each pick appears in the list in the order picked, with an icon and a **Remove** button, and the picker is empty again after each pick. After saving, "Knee arthroscopy is saved." and the Information column says **1 video, 1 leaflet**. Edit again: both are listed in the same order.

**T-02 What the picker offers**
1. Edit "Knee arthroscopy" again and open the picker.

Expected: it offers Preopsis's example videos and Trust A's own active leaflets (S36 Parking leaflet among them), each labelled (Video) or (Leaflet). It does **not** offer the two already in the list, nor **S36 Trust B leaflet**, nor any retired item. Typing narrows the list; nonsense says "Nothing matches".

**T-03 Remove from the list, and Cancel**
1. In the open form, **Remove** the video, then **Cancel**.
2. Edit again.

Expected: Cancel kept the saved list: both items are still there. (Don't save a removal yet.)

**T-04 Who may change it**
1. As **Cara Clinician**, open the Procedures page by its address (`…/procedures`, copy it from Alex's session).
2. As Cara, in DevTools → Console on any app page, run (Trust A's and Knee arthroscopy's ids are fixed by the seed):
   ```js
   fetch("/api/o/00000000-0000-4000-8000-0000000000fa/procedures/00000000-0000-4000-8000-000000000302", {method: "PATCH", headers: {"Content-Type": "application/json"}, body: JSON.stringify({infoContent: []})}).then(r => r.json().then(j => console.log(r.status, j)))
   ```

Expected: step 1 says only super clinicians and admins of the whole trust can manage procedures. Step 2 prints **403** with a plain message, and Knee arthroscopy's list is unchanged (Alex: still 1 video, 1 leaflet).

### The estimated time

**T-05 The editor's estimate**
1. As **Alex Author**, open **S36 Time HQ** in the editor. Click the Question Set **About you** in the structure (the set itself, not its page).
2. Read the line at the top: "Question Set · About 1 minute for the patient" (4 questions × 15 s = 60 s).
3. Add a **fifth** Yes / No question on its page; click the set again.
4. Add a **Text** question and turn its **Clinical** switch on (clinician only); add a **Clinical Summary**; add an **Information** block. Click the set again.
5. Add a **Video** and **Choose from library** → "Example: Your anaesthetic explained" (10:35). Click the set again.
6. Click the **Nurse checks** set.

Expected: step 3 → **About 2 minutes** (75 s). Step 4 → still **About 2 minutes** (none of those count). Step 5 → **About 12 minutes** (75 s + 635 s = 710 s, rounded up). Step 6 → no estimate (a clinician set). Now **Publish** the HQ.

**T-06 The patient's tiles, at 390 × 844**
1. Make an episode for patient **"Sam S36-One"** with procedure **Knee arthroscopy** and HQ **S36 Time HQ** (see Setup), and open it as the patient at 390 × 844.
2. Read the **About you** tile.
3. Start **About you**, answer every question, finish the section; you're back on the list.

Expected: the tile says **About 12 minutes** under its name. After finishing, the tile shows a whole **COMPLETE** badge and a whole **Review** button (not "COMPL…" or "Revie"); on a narrow screen they may sit on a line below the name. No sideways scrolling.

### The patient's Information

**T-07 The patient sees the procedure's video and leaflet, and the hospital's contacts** (390 × 844)
1. As Sam S36-One, go to **← Your questionnaires** (the home).
2. On the S36 Time HQ card, press **Information**.

Expected: below the card's own button row, the list opens: **Example: Your anaesthetic explained** with "Length 10:35" and the video player (it may play); **S36 Fasting leaflet** as a link (opens `https://example.org/s36-fasting.pdf` in a new tab); then **Questions? Contact Hospital A1** with **0100 555 0199** (a phone link) and **poa@hospital-a1.example** (an email link). S36 Parking leaflet is **not** there. Pressing Information again closes it. Nothing scrolls sideways; the video fits the screen width.

**T-08 Add a leaflet for one episode only** (clinician)
1. As **Cara Clinician**, open Sam S36-One's episode. Find the **Information for the patient** card.
2. Note its two items, each "From the procedure". Use **Add a video or leaflet for this patient** → **S36 Parking leaflet**.
3. Scroll to **General notes**.
4. As the patient (Sam S36-One, 390 × 844), reload the home and open Information.

Expected: step 2: the leaflet is added last, "Leaflet · Added for this patient by Cara Clinician"; the picker no longer offers it. Step 3: a note "Information added: S36 Parking leaflet" by Cara. Step 4: the patient sees S36 Parking leaflet too.

**T-09 Another patient with the same procedure doesn't get it**
1. Make a second episode for **"Sam S36-Two"** (another email) with **Knee arthroscopy** and **S36 Time HQ**; open it as that patient at 390 × 844, go to the home, open Information.

Expected: the video and S36 Fasting leaflet only, not S36 Parking leaflet. Cara's episode page for S36-Two shows the two "From the procedure" items only.

**T-10 Changing the procedure's list leaves booked patients alone** (S36 answers 2)
1. As **Alex Author**, edit Knee arthroscopy: **Remove** S36 Fasting leaflet, **Save changes** (Information column: "1 video").
2. As Sam S36-One and S36-Two, reload the home and open Information.
3. Make a third episode, **"Sam S36-Three"**, with Knee arthroscopy and S36 Time HQ; open its Information as the patient.

Expected: S36-One and S36-Two still see S36 Fasting leaflet (their lists were copied when they were booked). S36-Three sees only the video.

**T-11 Retiring a library item hides it from every patient**
1. As **Ada Admin**, Content library → **Retire** S36 Fasting leaflet.
2. As Sam S36-One and S36-Two, reload the home and open Information.
3. As Cara, open S36-One's episode page.
4. As Alex, edit **Total hip replacement** and open its picker.
5. As Ada, **Bring back** S36 Fasting leaflet; as S36-One, reload and open Information.

Expected: step 2: the leaflet is gone for both. Step 3: it's still listed, with a grey **RETIRED** badge (the patient doesn't see it). Step 4: it isn't offered. Step 5: S36-One sees it again.

**T-12 Remove an item from one episode**
1. As Cara, on S36-One's episode, **Remove** S36 Parking leaflet.
2. As S36-One, reload the home and open Information.

Expected: it's gone from the card and from the patient's Information; General notes say "Information removed: S36 Parking leaflet".

**T-13 Changing the episode's procedure**
1. As **Alex Author**, give **Laparoscopic cholecystectomy** the video **Example: Fasting before your operation**; save.
2. As Cara, on S36-Two's episode: add **S36 Parking leaflet** (for this patient), then **Edit procedure and dates** → Procedure **Laparoscopic cholecystectomy** → Save.
3. Look at the Information card; as S36-Two, reload the home and open Information.

Expected: the items that came from Knee arthroscopy are replaced by **Example: Fasting before your operation** ("From the procedure"); **S36 Parking leaflet** ("Added for this patient by Cara Clinician") stays. The patient sees the same.

**T-14 A hospital with no contact details, and an episode with nothing**
1. As **Val Clinician** (Hospital A2), make an episode with **no procedure** (any published HQ; a new patient with a new email); open it as the patient at 390 × 844, go home, press Information.

Expected: "There are no videos or leaflets for this yet." and no contact section (Hospital A2 has no phone or email set).

**T-15 Cancelled episodes**
1. As Cara, cancel S36-Three's episode (Episode status → Cancel, any reason).
2. Look at its Information card; as S36-Three, reload the home.

Expected: the card lists its items but has no Remove buttons and no picker. On the patient's home the cancelled card has **no Information button**.

### Refused paths and other trusts

**T-16 Trust B never sees Trust A's items, and the reverse**
1. As **Bea Author**, open Trust B's **Procedures**, edit **Inguinal hernia repair**, open the picker.
2. Give it **S36 Trust B leaflet**; save. Then, as Bea at **Hospital B1**, make an episode for a new patient (new email) with **Inguinal hernia repair** (any published HQ), and open its Information as the patient at 390 × 844.
3. As Alex (Trust A), open Knee arthroscopy's picker.

Expected: Bea's picker offers Preopsis's example videos and S36 Trust B leaflet, but **none** of Trust A's leaflets. The Trust B patient sees only S36 Trust B leaflet and Hospital B1's (empty) contacts. Alex's picker never offers S36 Trust B leaflet.

**T-17 A patient can't read another person's Information**
1. As Sam S36-One (patient, signed in), note S36-Two's episode id (from Cara's episode page address: the part after `/episodes/`).
2. In the patient's DevTools Console run:
   ```js
   fetch("/api/p/episodes/<S36-Two's id>/information").then(r => r.json().then(j => console.log(r.status, j)))
   ```
3. Repeat with `abc` as the id, then sign the patient out and run it with their own id.

Expected: **404** "Episode not found." for the other person's and for `abc`; **401** with a plain message once signed out.

**T-18 Staff at another hospital or trust**
1. As **Val Clinician** (Hospital A2), paste S36-One's episode page address (Hospital A1's).
2. As Val, in the Console, run `fetch("<the same path with /api in front of /o/…>/information").then(r => console.log(r.status))` (e.g. `/api/o/<trust>/h/<hospital A1>/episodes/<id>/information`).

Expected: the page says the hospital or episode isn't found; the request answers **404**.

**T-19 Bad input on the episode routes** (as Cara, in the Console on S36-One's episode page)
1. `fetch(location.pathname.replace(/^/, "/api") + "/information", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({contentId: "abc"})}).then(r => r.json().then(j => console.log(r.status, j)))`
2. The same with the id of an item already on the episode (the video's id: see the Network tab's `information` response).

Expected: step 1 → **400** "Choose a video or leaflet from the content library."; step 2 → **409** "That video or leaflet is already in this episode's Information."

### Reload, back and the audit log

**T-20 Reload and back**
1. On the patient's home with Information open, reload: the list is closed again (press Information to open it).
2. Open the questionnaire, then the browser's **Back**: the home is shown, and Information works.
3. On Cara's episode page, reload: the Information card shows the same items.

Expected: as described; no errors.

**T-21 The audit log**
1. As **Ada Admin**, open **Audit** and filter to today.

Expected: rows for "Added Information to an episode" and "Removed Information from an episode" (Cara), "Opened an episode's Information" (Cara), "Patient opened their Information" (the patient), and "Changed a procedure" (Alex). No patient names or item titles in the rows beyond what other rows show.

## 3. Regression

- **R-01** The patient sends their answers: finish About you for S36-One (tick the video's box if you added the tick; T-05 didn't), press **Send my answers**: "Thank you".
- **R-02** Cara validates S36-One's About you and opens the POA Summary: answers as before; the clinician-only question and the summary appear for the clinician.
- **R-03** A video inside a Question Set (T-05's) shows its player and length to the patient at 390 × 844 (S34).
- **R-04** An Information block still shows formatted text to the patient (S35): add one with `## Heading` and `- a list` in a new draft, Preview it.
- **R-05** Procedures: add a new procedure, retire it and bring it back (S19), with an empty Information list.
- **R-06** Content library (S33): as Ada, edit **S36 Parking leaflet**'s title to `S36 Parking map`; it shows with the new title on S36-Two's episode page and in S36-Two's Information (the list points at the library item, so edits reach it).

## 4. Out of scope

- Uploading leaflets (plan C8): a leaflet is a link.
- A patient's to-do list of required videos, or measuring real completion times.
- Changing the seconds per question (a server setting; Rahul checks it by hand).
- Patients choosing a language.
- Earlier notes already in `docs/pending.md` (e.g. a reload inside a set returns to the list).
