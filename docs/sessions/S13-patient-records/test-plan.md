# S13 manual test plan: patient records

For a tester who hasn't seen the code. Use only the browser.

**Changed after test report 2 (all passed):** new case **T-18** (Change patient forgets the patient in the address; the patient page highlights **Episodes**; "Patient not found." has **← Episodes**). For the retest, run **T-18** (round 3's numbers and `R3` surnames), plus a quick T-05 and T-13.

**Changed after test report 1:** F-1 is fixed (no questionnaire → "Choose a published HQ."). **Change patient** now empties the Hospital number (T-05), and **New episode** from a patient's page fills it in (T-13). T-05's "first 20" hint now uses surnames. For the retest, run **T-04, T-05 and T-13** (round 2's numbers and `R2` surnames), plus a quick T-02 and T-07.

**What S13 built:** a patient record now belongs to the **trust**, identified by its **NHS number** when it has one (checked with the NHS number's check digit). Staff create episodes on a new **New episode page** (it replaces the old drawer with its Existing/New patient tabs):
1. **Find the patient:** a search of **this hospital's patients only** (anyone with at least one episode here), by NHS number, hospital number or surname. It needs a precise term and shows at most 20 results.
2. **Or type a new patient's details** from the referral. If the trust already has that patient (same NHS number, date of birth and surname; or, without an NHS number, the same name, date of birth **and** the same mobile or email), the episode is **linked silently** to the trust's record. **Nothing about the other hospital is shown.**
3. **Refusals and warnings:** a wrong NHS number is refused; an NHS number the trust has with a different date of birth or surname is refused **without showing the stored details**; the same name and date of birth with other (or no) contact details gives a **"Possible duplicate"** warning with **Create anyway**.

The **hospital number** is now on the **episode** (each hospital's own label for the patient), not the patient. Each patient has a **patient page** showing the trust's record and their episodes **at this hospital only**.

## Setup

**Rahul, in your own terminal** (the automatic run does this itself):

```sh
docker compose up -d db
make migrate seed
make dev        # leave it running
```

**Open:** http://localhost:3000. It goes to the development sign-in page: click a person to sign in as them. **Sign out** is in the page header.

**A second person at the same time:** open **http://tester.localhost:3000** in another tab. It's the same app but keeps its own sign-in, so you can be two people at once (e.g. Cara in one tab, Val in the other).

**Seeded people:**

| Person | What they are |
| --- | --- |
| **Cara Clinician** | Clinician at **Hospital A1** only (Trust A) |
| **Val Clinician** | Clinician at **Hospital A2** only (Trust A) |
| **Ada Admin** | Admin of the whole of Trust A (both hospitals; sees the Audit screen) |
| **Max Multi** | Clinician at Hospital A2 **and** admin of Trust B (Hospital B1) |
| **Bea Author** | Super clinician of the whole of Trust B |

**Published questionnaires:** the local database has **"Trust A demo HQ"** and **"Trust B demo HQ"** (v1), plus many "E2E …" ones from automated runs. Any published one will do. *If Trust B has none:* sign in as Bea Author, **Questionnaires → New**, add a Question Set with one question, and **Publish**.

**Made-up data only.** Use the NHS numbers below (all made up, with valid check digits). **Use the set for your round**, and **add the round to every surname you type** (e.g. `Linkman R1`), so a retest never meets an earlier round's records.

| Round | N1 | N2 | N3 | N4 | N5 | N6 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 9990002118 | 9990002126 | 9990002134 | 9990002142 | 9990002150 | 9990002169 |
| 2 | 9990003106 | 9990003114 | 9990003122 | 9990003130 | 9990003149 | 9990003157 |
| 3 | 9990004102 | 9990004110 | 9990004129 | 9990004137 | 9990004145 | 9990004153 |
| 4 | 9990005109 | 9990005117 | 9990005125 | 9990005133 | 9990005141 | 9990005168 |

- **A wrong check digit:** take any of them and change the last digit, e.g. `9990002119`.
- **Mobiles:** use Ofcom's drama range, `07700 900000` to `07700 900999`. They're stored in the international format: `07700 900123` becomes `+447700900123`.
- **Telling whether two episodes share one record:** open each episode, click the **patient's name** (the heading) to reach the patient page, and compare the address: `…/patients/<record id>`. The same id means the same trust record.

## Test cases

### T-01 The New episode page replaces the drawer

1. Sign in as **Cara Clinician**. You land on Hospital A1's **Episodes**.
2. Click **New episode**.
3. Reload the page; then use the browser's **Back** button; then **Forward**.

**Expected:** a full page titled **New episode** at `…/h/<A1>/episodes/new` (not a side drawer), with three parts: **Find the patient** (a Search box and an optional date of birth), **New patient** (first name, surname, date of birth, sex, NHS number, mobile, email) and **Episode** (hospital number, health questionnaire, procedure, anaesthetic, consultant). There are **no "Existing patient / New patient" tabs**. Reload keeps you on the page; Back returns to Episodes; Forward comes back. **← Episodes** and **Cancel** go back to the list.

### T-02 Create a patient with a valid NHS number

1. As **Cara**, on the New episode page, type in **Search**: your round's **N1** with spaces, e.g. `999 000 2118`.
2. Fill **New patient**: First name `Nora`, Surname `Valid R<round>`, Date of birth `1985-03-04`, Sex Female, **NHS number** `999 000 2118` (N1, with spaces), Mobile `07700 900101`, Email `Nora.Valid@Example.TEST`.
3. **Episode:** Hospital number `A1-N1`, a published Health questionnaire, Procedure `Knee replacement`. Click **Create episode**.

**Expected:** step 1 says **"No patients at this hospital match. Enter their details below."** Step 3 opens the new episode. Its card shows **Hospital number A1-N1**, **NHS number 9990002118** (digits only), **Mobile +447700900101**, **Email nora.valid@example.test** (lower case), and the date of birth.

### T-03 A wrong NHS number is refused, clearly

1. As **Cara**, open New episode. Type a new patient (Surname `Wrong R<round>`, any date of birth and sex) with the NHS number **N1 with its last digit changed** (e.g. `9990002119`). Choose a questionnaire. Click **Create episode**.
2. Try again with only 9 digits (`999000211`).
3. Go back to **Episodes**.

**Expected:** each time a red message: **"That isn't a valid NHS number. Check it, or leave it empty if the patient doesn't have one."** You stay on the page with what you typed. No episode for "Wrong R…" appears in the Episodes list.

### T-04 Other details are checked

On the New episode page as **Cara**, try each of these with an otherwise valid new patient, clicking **Create episode** each time:
1. Mobile `12345`.
2. Email `nora@`.
3. No questionnaire chosen.
4. No first name (the browser may stop you; clear it and click Create).

**Expected:** a plain message for each: "Enter a valid mobile number, or leave it empty.", "Enter a valid email address, or leave it empty.", "Choose a published HQ.", and a message asking for the name. Nothing is created.

### T-05 Search this hospital's patients

As **Cara**, on the New episode page, using the patient from T-02:
1. Type `Va` in Search. Wait a second.
2. Type `Val` (3 letters).
3. Type `Valid R<round>` (the full surname), then the NHS number N1 **with spaces**, then the hospital number in lower case `a1-n1`.
4. Type `Va` again and set **Date of birth (optional)** to `1985-03-04`.
5. Click **Choose** on Nora's row. Then click **Change patient**.

**Expected:**
- Step 1: nothing is searched (no table, no "No patients" message).
- Steps 2–4: a **Patients found** table with Nora's row: **Name, Date of birth, NHS number, Hospital number (A1-N1)** and a **Choose** button. It shows no mobile or email. Partial hospital numbers (e.g. `a1-n`) find nothing.
- Step 5: Choose replaces the search and the New patient form with a card for **Nora Valid R…** (born …, NHS number …, Hospital number A1-N1) and fills the Episode's **Hospital number** with A1-N1. **Change patient** brings the search and form back **and empties the Hospital number** (it was Nora's).
- A search matching more than 20 patients shows 20 and the line **"Showing the first 20. Type more, or add the date of birth, to narrow the search."** (The search matches **surnames**: try `Aud`, or `J` + date of birth `1980-05-17`. Automated runs have made many "Sam Audit …" and "Sam J3 …" patients at A1.) With a date of birth set, the line says only "Type more to narrow the search."

### T-06 A patient known only at the other hospital is never found

1. In the second tab (**tester.localhost:3000**), sign in as **Val Clinician** (Hospital A2). New episode: type a new patient: First name `Greta`, Surname `Elsewhere R<round>`, Date of birth `1970-02-03`, Sex Female, NHS number **N2**, Mobile `07700 900202`, Email `greta@example.test`. Hospital number **`A2-SECRET`**. Choose a questionnaire. **Create episode**.
2. Back in Cara's tab (Hospital A1), on New episode, search for: **N2**; then `A2-SECRET`; then `Elsewhere R<round>`; then `Els` + date of birth `1970-02-03`.

**Expected:** step 1 creates the episode at A2. In step 2 **every** search says **"No patients at this hospital match."** Nothing about Greta or Hospital A2 appears.

### T-07 Typing a known patient's details links silently, showing nothing about the other hospital

1. As **Cara** (A1), on New episode, type: First name **`G`**, Surname `elsewhere r<round>` (lower case), Date of birth `1970-02-03`, Sex Female, NHS number **N2**, Mobile `07700 900999`. Hospital number **`A1-G`**. Choose a questionnaire. **Create episode**.
2. Look over the episode page. Click the patient's name (the heading) to open the **patient page**. Note the record id at the end of the address.
3. In Val's tab (A2), open Greta's A2 episode from Episodes, click her name, and note that patient page's record id.
4. In each tab, look at **Episodes** (the hospital's list).

**Expected:**
- Step 1 creates the episode **with no warning or question**: it was linked to the trust's existing record.
- The episode page shows **Hospital number A1-G**. It **never** shows `A2-SECRET`, the words "Hospital A2", or A2's episode.
- The patient details shown are the **trust's record** (first name **Greta**, mobile **+447700900202**, email greta@example.test), not what Cara typed. *This is expected:* once the patient has an episode at A1, A1's staff see the trust's patient record (Rahul, S08 answers). Nothing was shown before the episode existed, and nothing about A2's episodes ever is.
- Both patient pages have the **same record id**. Cara's patient page lists **"Episodes at Hospital A1"** with **one** episode (A1-G); Val's lists one at A2 (A2-SECRET).
- Each hospital's Episodes list shows only its own episode.
- Now Cara's search for **N2** finds Greta, with hospital number **A1-G** (never A2-SECRET).

### T-08 A mismatched NHS number and date of birth is refused, without the stored details

1. As **Cara** (A1), on New episode, type: First name `Greta`, Surname `Elsewhere R<round>`, Date of birth **`1971-02-03`** (one year off), Sex Female, NHS number **N2**. Choose a questionnaire. **Create episode**.
2. Change the date of birth back to `1970-02-03` but the surname to `Other R<round>`. **Create episode**.

**Expected:** each time a red box **"The details don't match"**: "These details don't match the trust's record for this NHS number. Check the NHS number, date of birth and surname. Nothing was created." It shows **none** of the stored details (no 1970, no "Greta", no phone or email, no hospital). You stay on the page, and no new episode appears in the Episodes list.

### T-09 The same NHS number: one record per trust

1. As **Val** (A2), on New episode, type Greta again exactly as in T-06 (NHS number N2, same surname and date of birth), Hospital number `A2-AGAIN`. **Create episode**. Open the patient page.
2. Sign in (either tab) as **Max Multi** and switch to **Hospital B1** (Trust B) with the header's hospital picker. On New episode, search for **N2**. Then type a new patient: First name `Bert`, Surname `Trustb R<round>`, Date of birth `1990-09-09`, Sex Male, NHS number **N2**. Choose **Trust B demo HQ** (or any Trust B HQ). **Create episode**.

**Expected:** step 1 makes **no second record**: the patient page (same record id as before) now lists **two** episodes at A2 (two open at once is allowed). Step 2: Trust B's search finds nothing, and the create **succeeds** with a **new** record for Bert in Trust B (the same NHS number may exist once per trust). Nothing about Trust A is shown.

### T-10 No NHS number: a possible-duplicate warning that names nobody, and Create anyway

1. As **Val** (A2), create a new patient: First name `Pat`, Surname `Nonhs R<round>`, Date of birth `1960-06-06`, Sex Female, **no NHS number**, Mobile `07700 900303`, Email `pat.nonhs@example.test`. Any questionnaire. Create.
2. As **Cara** (A1), on New episode, type: First name `pat` , Surname `NONHS R<round>`, Date of birth `1960-06-06`, Sex Female, no NHS number, Mobile **`07700 900404`** (a different one). Choose a questionnaire. **Create episode**.
3. Click **Create anyway**.

**Expected:** step 2 shows an orange **"Possible duplicate"** box: "This patient may already be registered in the trust. Check their NHS number or contact details, or create a new record anyway." It names **nobody**: no record, no stored mobile or email, no hospital, and **no "Use this record" button**. Step 3 creates the episode; its patient details are only what Cara typed (mobile **+447700900404**, no email), and its patient page has a **different** record id from Val's.

### T-11 No NHS number: the same name, date of birth and mobile (or email) link silently

1. As **Val** (A2), create: First name `Mo`, Surname `Contact R<round>`, Date of birth `1955-05-05`, Sex Male, no NHS number, Mobile `07700 900505`, Email `mo.contact@example.test`. Create. Note the patient page's record id.
2. As **Cara** (A1), on New episode, type: First name `Mo`, Surname `Contact R<round>`, Date of birth `1955-05-05`, Sex Male, no NHS number, Mobile **`+44 7700 900505`** (the same number, another format). Create.
3. Repeat step 2 at A1 with **no mobile** but Email **`MO.CONTACT@example.test`**.

**Expected:** steps 2 and 3 create episodes **with no warning**. Their patient pages have **the same record id** as Val's (linked), and show the email Val typed (mo.contact@example.test), even in step 2 where Cara didn't type it. Nothing about Hospital A2 is shown. Cara's patient page lists two A1 episodes.

### T-12 A match among this hospital's own patients is shown, to use or not

1. As **Cara** (A1), create: First name `Ola`, Surname `Ownhere R<round>`, Date of birth `1999-09-09`, Sex Female, no NHS number, no contact. Hospital number `A1-OLA`. Create.
2. On New episode again, type the same name and date of birth with Mobile `07700 900606`. **Create episode**.
3. In the orange box, click **Use this record**. Then **Create episode**.

**Expected:** step 2's **"Possible duplicate"** box says "This hospital already has a patient with this name and date of birth. Use that record, or create a new one anyway." and **shows Ola** (name, date of birth, NHS number —, Hospital number A1-OLA) with **Use this record**, plus **Create anyway**. Step 3 chooses Ola (her card replaces the form; Hospital number fills with A1-OLA) and creates the episode; Ola's patient page lists **two** episodes.

### T-13 The patient page

1. As **Cara**, open any A1 episode from T-02 to T-12 and click the patient's name.
2. Check the page; click an episode row; go **Back**; **reload**.
3. Click **New episode** on the patient page.
4. In Val's tab, open the patient page of **Pat Nonhs R<round>** from T-10 step 1 (Val's record, which has no A1 episode) and copy its address. In Cara's tab, paste it, then replace the hospital id after `/h/` with A1's id (copy it from Cara's own address bar). Do the same with **Greta**'s A2 patient page (T-07).

**Expected:**
- The page shows the patient's name, **NHS number, date of birth (with age), sex, mobile, email**, then **"Episodes at Hospital A1"**: only A1's episodes (created, HQ, procedure, hospital number, status). Clicking a row opens that episode; Back and reload work.
- **New episode** opens the New episode page with this patient already chosen (the patient card, with **Change patient**), and the card and the Episode's **Hospital number** show their newest hospital number here (e.g. A1-N1), as choosing them from the search does.
- Step 4: **Pat Nonhs (Val's record) shows "Patient not found."**, even though the trust holds her. Greta's address works at A1 (she has an A1 episode now) and lists only A1's episode.

### T-14 The hospital number is on the episode

1. Look at Cara's **Episodes** list, an episode page, and its **POA Summary** for the T-02 episode.

**Expected:** the hospital number (A1-N1) shows under the patient's name in the list, in the episode's card, and on the POA Summary (beside the NHS number and mobile). The POA Summary also shows **NHS number** and **Mobile**.

### T-15 Another hospital's and another trust's pages are refused

1. As **Cara** (A1 only), paste Val's A2 New episode address (`…/h/<A2>/episodes/new`) and an A2 patient page address.
2. As **Bea Author** (Trust B), on Hospital B1's New episode, search for **N1** (Nora, Trust A) and for `Valid R<round>`.

**Expected:** step 1 shows **"You don't have access to this hospital"**, not A2's data. Step 2 finds nothing: Trust A's patients are never searched.

### T-16 The audit log records the new screens

1. Sign in as **Ada Admin**. Open **Audit**. Choose **Who: Cara Clinician**, click **Show**.

**Expected:** rows for **"Searched patients"**, **"Opened a patient"** and **"Created an episode"** from today, each naming the patients involved (a search names those it found). Typing `Valid R<round>` (or N1) in **Patient** and Show finds Nora's rows.

### T-17 The patient's link still works on a phone (390×844)

1. As **Cara**, on the T-02 episode, click **Copy** beside the patient link.
2. Open the link in a window sized **390×844** (e.g. Chrome's device toolbar, iPhone 12 Pro).

**Expected:** the patient's questionnaire opens ("Hello Nora."), fits the screen with no sideways scrolling, and can be filled in and sent as before.

### T-18 Change patient forgets the patient; the patient page's header and "not found" page

1. As **Cara**, open any A1 patient's page (e.g. Nora's from T-02). Check the header.
2. Click **New episode**. The address ends in `?patient=<id>` and the patient's card shows.
3. Click **Change patient**. Check the address. **Reload** the page.
4. Use **Back**, then **Forward**.
5. Open a patient page that doesn't exist at A1 (e.g. change the last character of the record id in the address).

**Expected:**
- Step 1: **Episodes** is highlighted in the header on the patient page, as on the episode pages.
- Step 3: the address becomes `…/episodes/new` (no `?patient=`); after the reload the page shows the search and the New patient form, **with no patient chosen**.
- Step 4: Back returns to the patient's page; Forward returns to New episode **with no patient chosen**.
- Step 5: "Patient not found." with a **← Episodes** link above it, which goes to the Episodes list.
- Switching hospital from the header while on a patient's page goes to the other hospital's **Episodes**.

## Regression

- **R-01** Cara's **Episodes** list and its status filter work; Ada sees **All hospitals** with each episode's hospital.
- **R-02** On a submitted episode, Cara can **Validate** each Question Set and **Complete HQ review**; the POA Summary opens.
- **R-03** Ada's **Staff** and **Settings** pages open.
- **R-04** The Audit screen's **Export CSV** downloads.
- **R-05** Alex Author can open a questionnaire in the editor and its Preview.

## Out of scope (don't report)

- Sending invites by SMS or email, the patient signing in, and patient accounts (S14–S15). Episodes still show the copyable patient link.
- Editing a patient record, or an episode's hospital number, after it's created; merging duplicate records.
- NHS numbers shown as 10 digits without spaces.
- The demo patients Jo Bloggs, Pat Example and Robin Sample: their records may already have episodes in this database from earlier use, so they aren't used in the cases above.
- Seeing "Greta"'s first name and contacts as the trust holds them once A1 has an episode for her (T-07): that's expected.
