# S33 test plan: the content library

**Changed after verification 1 (for the last retest, test report 4):** new cases **T-16** (a capitalised link, `Https://…`, is saved), **T-17** (an old error doesn't come back after Edit → Cancel) and **T-18** (a new item can't be added already retired). Retest: T-16, T-17, T-18, then the full happy paths **T-02, T-03, T-05, T-07, T-13** and a quick **T-14** at 390px, because the link field and the form changed.

**Changed after test report 2:** T-14 also checks the Starter questions table's tags (e.g. MEASUREMENTS) and the checklist badges; T-15 step 1 also checks that leaving the Tags box with only spaces typed adds no empty tag.

**Changed after test report 1:** T-14 now checks the tables scroll sideways in their own box and the console's top bar scrolls; T-15 added (Enter in an empty Tags box, and switching Video → Leaflet).

For a tester who uses only the browser. Everything is made up; no real patient data.

## Setup

**In your own terminal** (the automatic run's conductor does this for you):

```sh
docker compose up -d db
make migrate seed
make dev        # leave it running
```

Open **http://localhost:3000**. Sign in on the stub login page by clicking a name. Sign out (top right) to change user, or use a second browser profile / incognito window to have two users at once.

| Who | Role | Use for |
| --- | --- | --- |
| **Sam Superadmin** | platform admin, in no trust | Preopsis's content library and the starter questions (`/admin/content`) |
| **Ada Admin** | admin, the whole of Trust A | Trust A's content library (add, edit, retire) |
| **Alex Author** | super clinician, the whole of Trust A | reads Trust A's library, can't change it; the questionnaire editor |
| **Cara Clinician** | clinician, Hospital A1 only | has no content library |
| **Sue Super** | super clinician, Hospital A1 only | has no content library (not the whole trust) |
| **Bea Author** | super clinician, the whole of Trust B | another trust: reads Trust B's library |
| **Max Multi** | clinician at Hospital A2; admin of the whole of Trust B | Trust B's admin |

After signing in, pick **Trust A → Hospital A1** (Bea and Max: Trust B → Hospital B1). Sam lands on the platform admin console.

**What the seed gives you:** three made-up **example videos** of Preopsis's, titled "Example: …" (Your anaesthetic explained; Fasting before your operation; Getting ready for a knee replacement), all pointing to the same public test film on YouTube, length 10:35. The seed puts them back as they were on each `make seed`.

**Links you can use:**
- YouTube: `https://www.youtube.com/watch?v=aqz-KE-bpKQ`, `https://youtu.be/aqz-KE-bpKQ`, `https://www.youtube.com/embed/aqz-KE-bpKQ`
- Vimeo: `https://vimeo.com/76979871`, a private link `https://vimeo.com/76979871/8272103f6e`, `https://player.vimeo.com/video/76979871`
- Not allowed for a video: `https://www.dailymotion.com/video/x7tgad0`, `https://example.org/film.mp4`, `http://www.youtube.com/watch?v=aqz-KE-bpKQ` (http, not https), `https://www.youtube.com/channel/UC123`
- A leaflet: any `https://` link, e.g. `https://www.trust-a.example/leaflets/fasting.pdf`. Not allowed: `http://…`, `leaflet.pdf`.

**Length** is typed as minutes and seconds (`3:20`), hours too (`1:02:05`), or whole minutes (`4`).

**Please use unique titles** (add the time, e.g. "Knee video 14:05"): items are never deleted, only retired, so earlier rounds' items stay in the list.

**Starter questions:** retiring one lasts across `make seed` (it's a real decision, not test data). **For T-13, use "Medicines you take regularly"**, and press **Offer again** at the end, because other tests use the other starter questions.

## Test cases

### Preopsis's library (Sam Superadmin)

**T-01 Open the page.**
1. Sign in as Sam Superadmin. In the top bar, click **Content library**.
2. Expected: the URL is `/admin/content`; the page is titled **Content library**; **Content library** is highlighted in the top bar (not **Trusts**). It shows **Add video** and **Add leaflet**, a **Checklist** card, a table **Preopsis's items** with the three "Example: …" videos (each Video, 10:35, with its tags), and a **Starter questions** section below.
3. Click **Trusts**: the trusts list opens, with **Trusts** highlighted. Use the browser's Back: the content library is back.
4. Reload `/admin/content` directly (deep link): the same page.

**T-02 Add a platform video with tags.**
1. Click **Add video**. Fill Title "Knee video <time>", link `https://youtu.be/aqz-KE-bpKQ`, Length `3:20`. In **Tags** type `Knee` and press Enter. Click **+ Recovery** under the tags (a ready-made checklist tag).
2. Click **Add video**. Expected: a green "Knee video … is added."; the row shows Video, **3:20**, tags Knee and Recovery, and the link (opens YouTube in a new tab when clicked).
3. Expected: the checklist's **Recovery** now says "covered" with a tick (it said "nothing yet" before, unless an earlier round left a Recovery item active).

**T-03 A link that isn't YouTube or Vimeo is refused with a plain message.**
1. Click **Add video**, Title "Bad video", Length `2:00`, and each of these links in turn, clicking **Add video** each time: `https://www.dailymotion.com/video/x7tgad0`; `https://example.org/film.mp4`; `http://www.youtube.com/watch?v=aqz-KE-bpKQ`; `https://www.youtube.com/channel/UC123`.
2. Expected each time: a red message under the form: "Only YouTube or Vimeo links can be used for a video, for example https://www.youtube.com/watch?v=… or https://vimeo.com/…". Nothing is added.
3. Try each accepted form from Setup's list (YouTube watch, youtu.be, embed; Vimeo, Vimeo private link, player.vimeo.com): each is added. (Retire the extras afterwards if you like.)

**T-04 Other refusals in the form.**
1. Add video with an empty title: "Enter a title."
2. Add video with no length, `0:00`, `3:75` or `soon`: "Enter the video's length."; `300` (300 minutes = 5 hours): "The video's length must be between 1 second and 4 hours."
3. Add leaflet with link `leaflet.pdf` or `http://example.org/a.pdf`: "Enter the leaflet's web address, starting https://"
4. Expected: nothing is added for any of these; **Cancel** closes the form.

**T-05 Add a leaflet; edit; retire and bring back.**
1. **Add leaflet**: Title "Fasting leaflet <time>", link `https://www.trust-a.example/leaflets/fasting.pdf`, click **+ Fasting**. Expected: added; Kind Leaflet; Length "—"; no Length field in the leaflet form.
2. On the video from T-02 click **Edit**. Expected: the form shows its values (length `3:20`). Change the length to `4:05`, remove the Knee tag (its ×), press Enter in the Length field. Expected: "… is saved."; the row shows 4:05 and only Recovery.
3. Click **Retire** on the video. Expected: "… is retired: it isn't offered any more."; the row moves to the bottom, greyed, with a **Retired** badge and a **Bring back** button. The checklist's Recovery goes back to "nothing yet" (if no other active Recovery item).
4. Click **Bring back**. Expected: "… is offered again."; it's active again.
5. Retire it again (it's used in T-11).

### A trust's library (Ada Admin, Trust A)

**T-06 Open Trust A's library.**
1. Sign in as Ada Admin (Trust A → Hospital A1). In the top bar, click **Content library**.
2. Expected: the URL is `/o/<trust>/h/<hospital>/content`. The page says it's Trust A's videos and leaflets; the **Checklist** card ("Preopsis's items count too"); a table **Trust A's items** (empty at first, or earlier rounds' items); and **From Preopsis (read-only)** with the three examples and Sam's active items, with no Edit or Retire buttons. Sam's retired video from T-05 is **not** listed.
3. Switch hospital to **Hospital A2** with the hospital picker: the content library of Trust A stays open. Reload: same page.

**T-07 Add a Trust A video and a leaflet.**
1. **Add video**: Title "A hip video <time>", link `https://vimeo.com/76979871/8272103f6e`, Length `12:05`, tags `Hip`. Expected: added, under **Trust A's items**, 12:05.
2. **Add leaflet**: Title "A who-to-call leaflet <time>", link `https://www.trust-a.example/leaflets/who-to-call.pdf`, click **+ Who to call**. Expected: added.
3. Expected: the checklist's **Who to call** is ticked "covered".
4. Edit the video: add **+ Recovery** and change the length to `12:30`; save. Expected: shown.
5. A bad link (`https://example.org/film.mp4`) on Edit is refused with the same plain message as T-03.

**T-08 The checklist shows topics with nothing yet; a tag ticks one.**
1. On Trust A's page, find a topic that says "nothing yet" (e.g. **Consent** or **What to bring**). Note the line "N of 11 have nothing yet."
2. Add a leaflet tagged with that topic using its **+ <topic>** button (or by typing it in Tags in another case, e.g. `consent`).
3. Expected: that topic now says "covered", and the count goes down by one.
4. Retire that leaflet. Expected: the topic says "nothing yet" again.

**T-09 Another trust never sees Trust A's items; both see Preopsis's.**
1. Sign in as Bea Author (Trust B → Hospital B1); open **Content library**.
2. Expected: **From Preopsis (read-only)** has the examples (and Sam's active items); Trust A's items from T-07 are nowhere on the page. No **Add video** / **Add leaflet** buttons for Bea (a super clinician reads only), and the line "Only the trust's admins can add or change items."
3. Sign in as Max Multi (Trust B → Hospital B1); open **Content library**; add a video "B video <time>". Then as Ada (Trust A) confirm "B video …" isn't on Trust A's page.

### Who may do what

**T-10 Authors read; clinicians have no library.**
1. As **Alex Author** (Trust A): **Content library** is in the top bar; the page lists Trust A's and Preopsis's items, with no Add, Edit or Retire buttons, and "Only the trust's admins can add or change items."
2. As **Cara Clinician**: no **Content library** in the top bar. Type `/o/<trust>/h/<hospital>/content` (copy the start of the URL from her Episodes page): the page says "Only super clinicians and admins of the whole trust can see the content library".
3. As **Sue Super** (Hospital A1 only): the same as Cara.
4. As **Ada Admin**, open `/admin/content`: "Only Preopsis's platform admins can open this page".
5. The API refuses too (from the browser console on any page while signed in as Cara, then Alex):
   ```js
   const o = location.pathname.split("/")[2];
   for (const [m, p, b] of [["GET", `/api/o/${o}/content`], ["POST", `/api/o/${o}/content`, {kind: "video", title: "X", url: "https://youtu.be/aqz-KE-bpKQ", lengthSeconds: 60, tags: []}], ["GET", `/api/o/${o}/content/search?tag=Hip`], ["GET", "/api/admin/content"]])
     console.log(m, p, (await fetch(p, {method: m, headers: {"Content-Type": "application/json"}, body: b && JSON.stringify(b)})).status);
   ```
   Expected: Cara gets **403** on all four. Alex gets **200** on the GET list and the search, **403** on the POST and on `/api/admin/content`. Signed out (after Sign out, on the login page), each answers **401**.
6. Trust B's people can't reach Trust A's library: as Bea, run `fetch("/api/o/00000000-0000-4000-8000-0000000000fa/content").then(r => r.status)`. Expected: **404** (Trust A's id is that one).

### The search (what S34's video picker will use)

**T-11 Search by tag returns Preopsis's and the trust's own, never retired ones or another trust's.**
There's no screen for it until S34. As **Alex Author** (Trust A), in the browser console:
```js
const o = location.pathname.split("/")[2];
const s = async (q) => (await (await fetch(`/api/o/${o}/content/search?${q}`)).json()).map(i => `${i.title} (${i.platform ? "Preopsis" : "trust"}, ${i.kind})`);
console.log(await s("tag=Hip"), await s("tag=fasting"), await s("q=example"), await s(""));
```
Expected:
- `tag=Hip`: Trust A's "A hip video …" (trust), never Max's Trust B video even if tagged Hip, never a retired item.
- `tag=fasting` (any case): "Example: Fasting before your operation" (Preopsis), plus any active Trust A or Preopsis leaflet tagged Fasting; not Sam's or Ada's retired ones.
- `q=example`: the three examples (words in the title or a tag).
- `""`: every active item of Preopsis's and Trust A's; none of Trust B's; nothing retired (e.g. Sam's T-05 video).
As Bea (Trust B), `tag=Hip` never lists Trust A's video.

### The platform's starter questions (Sam Superadmin)

**T-12 The list.**
1. On `/admin/content`, the **Starter questions** section lists the 11 starter questions with their text, ID and tags, each with **Retire**. Trusts' own library questions are not listed.

**T-13 Retire a starter question: Insert from library no longer offers it; questions already inserted stay.**
1. First, as **Alex Author**: open any **draft** questionnaire in Trust A (or create one: Questionnaires → New questionnaire), open a page, use **From library** to insert **"Medicines you take regularly"**, and wait for "All changes saved". Note it's there.
2. As **Sam**, on `/admin/content`, click **Retire** on "Medicines you take regularly". Expected: a green "“Medicines you take regularly” is retired: Insert from library no longer offers it. Questionnaires that already have it keep it."; the row moves last, greyed, **Retired**, with **Offer again**.
3. As **Alex**, reload the draft. Expected: the question inserted in step 1 is still there, unchanged (same text, same ID `regular_medicines`).
4. Open **From library** on any page and search "medicines". Expected: "Medicines you take regularly" isn't offered. Other starter questions are.
5. As **Sam**, click **Offer again**. Expected: "… is offered again."; as Alex, **From library** offers it again.
6. **Leave it offered** at the end (other tests use the library).

### Phone size

**T-14** The content library is staff-only (no patient page in this session). At a narrow window (390×844):
1. As **Ada** on Trust A's **Content library**: the form's fields stack; each table (Trust A's items, From Preopsis) scrolls sideways **inside its own box**, and every tag and **Retired** badge reads in full (no "H." or "R.."). The page itself doesn't scroll sideways.
2. As **Sam** on `/admin/content`: the top bar scrolls sideways on its own; **PLATFORM ADMIN**, **Trusts**, **Content library** and **Sign out** read in full when scrolled to. The items and **Starter questions** tables behave as in step 1. In the Starter questions table, scrolled right, **MEASUREMENTS** (How tall are you?, What do you weigh now?) and every other tag read in full. Every checklist badge reads in full on both pages.

**T-15 Enter in the Tags box, and switching the kind.**
1. As Ada, **Add leaflet**; fill Title and a https link. In Tags type `Hip`, press Enter (the tag is added), then press Enter again with the box empty, and once more after typing only spaces. Expected: nothing is sent; the form stays open with the Hip tag. Then click the Title field (leaving the Tags box with the spaces still typed): still only the Hip tag, no empty chip.
2. Switch the form to **Video** (the Video / Leaflet switch at its top), click **Add video**: the red "Only YouTube or Vimeo links…" message. Switch back to **Leaflet**: the message is gone.
3. Press Enter in the Title field: the leaflet is added (Enter in other fields still sends the form). Retire it afterwards.

### After verification 1

**T-16 A capital in the link (as a phone keyboard types it) is saved.**
1. As **Sam**, **Add leaflet**: Title "Capital leaflet <time>", link `Https://WWW.Trust-A.example/Leaflets/Capital.pdf`. Expected: added (no "Something went wrong"); the row's link reads `https://www.trust-a.example/Leaflets/Capital.pdf` (scheme and host in lower case; the path keeps its capitals).
2. **Add video** with link `HTTPS://WWW.YOUTUBE.COM/watch?v=aqz-KE-bpKQ`, length `1:00`. Expected: added, link `https://www.youtube.com/watch?v=aqz-KE-bpKQ`.
3. **Edit** that video and change its link to `Https://Vimeo.com/76979871`. Expected: saved, link `https://vimeo.com/76979871`.
4. As **Ada**, repeat step 1 on Trust A's page: added the same way.
5. `HTTP://www.youtube.com/watch?v=aqz-KE-bpKQ` (http, in capitals) on a video is still refused with the plain YouTube-or-Vimeo message; `HTTP://example.org/a.pdf` on a leaflet with the leaflet message.
6. Inspect the link field (right-click → Inspect, or in the console `document.querySelector('[data-testid=content-form] input[type=url]')`): it's `type="url"`, `autocapitalize="none"`.
7. Retire what you added.

**T-17 An old error doesn't come back.**
A Retire that fails is hard to cause by hand. In the browser console on Ada's Content library page, run this first: it makes the next Retire or Bring back answer an error, once.
```js
const real = window.fetch;
window.fetch = async (u, o) => { if (o?.method === "PATCH" && String(u).includes("/content/")) { window.fetch = real; return new Response(JSON.stringify({ error: "Test: this retire was refused." }), { status: 409, headers: { "Content-Type": "application/json" } }); } return real(u, o); };
```
1. Click **Retire** on one of Trust A's items. Expected: the red "Test: this retire was refused."; the item is unchanged.
2. Click **Edit** on any item: the form opens (the red message goes). Click **Cancel**. Expected: the red message does **not** come back.
3. The same with **Add video** → **Cancel**: no old message.
4. Click **Retire** again (the real one now): "… is retired".

**T-18 A new item can't be added already retired** (API only; the form never sends it). As Sam, in the console:
```js
fetch("/api/admin/content", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ kind: "video", title: "Retired at birth", url: "https://youtu.be/aqz-KE-bpKQ", lengthSeconds: 60, tags: [], status: "retired" }) }).then(async r => console.log(r.status, await r.text()))
```
Expected: **400** "A new item is always active."; nothing named "Retired at birth" appears in the list.

## Regression (earlier sessions)

- **R-01 (S10)** As Sam: **Trusts** lists the trusts; open Trust A; its hospitals show. **New trust** opens its form (no need to save).
- **R-02 (S19)** As Alex: **Procedures** lists Trust A's procedures; add one and retire it.
- **R-03 (S24)** As Alex, in a draft questionnaire: **From library** inserts "Do you smoke?" with ID `smoker`; **Save to library** on a question of your own works and it then appears under "Trust A" in **From library**.
- **R-04 (S24)** "Same as…" on a new single-choice question offers the starter question "Do you smoke?" (search "smoke").
- **R-05 (S32)** As Alex, a Clinical Summary can still be added to a page from the Add bar.

## Out of scope (don't report)

- The Video element and its picker in the editor (S34); the patient's Information on their home (S36).
- Uploading leaflets or images (C8): a leaflet is a link only.
- A trust hiding Preopsis's items it doesn't want.
- Adding new starter questions in the app (they come from the seed file).
- Deleting items: they're retired, never deleted.
- Mantine badges show tags in capitals: that's the house style.
