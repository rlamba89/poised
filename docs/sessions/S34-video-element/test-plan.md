# S34 test plan: the Video element

**Changed after test report 1:** T-09 now has Ada add and retire a video of **her trust's own**: Preopsis's items (the "Example: …" videos) are read-only on a trust's library page, with no Retire button.

For a tester who uses only the browser. Everything is made up; no real patient data.

## Setup

**In your own terminal** (the automatic run's conductor does this for you):

```sh
docker compose up -d db
make migrate seed
make dev        # leave it running
```

Open **http://localhost:3000**. Sign in on the stub login page by clicking a name, then pick **Trust A → Hospital A1**. Sign out (top right) to change user, or use a second browser profile / incognito window to have two users at once.

| Who | Role | Use for |
| --- | --- | --- |
| **Alex Author** | super clinician, the whole of Trust A | the questionnaire editor: adds videos, publishes |
| **Ada Admin** | admin, the whole of Trust A | the trust's Content library (retire / bring back a video, T-09) |
| **Cara Clinician** | clinician, Hospital A1 only | makes episodes, validates, sees the POA Summary; can't edit questionnaires |
| **Sue Super** | super clinician, Hospital A1 only | views the trust's questionnaires but can't change them |

**The content library** (S33) has three made-up example videos of Preopsis's, "Example: Your anaesthetic explained" (tags The anaesthetic, Example), "Example: Fasting before your operation" (Fasting, Example) and "Example: Getting ready for a knee replacement" (Knee replacement, Prehab, Example), each 10:35, all the same public test film on YouTube. Earlier rounds may have left other videos (e.g. "A hip video …"): that's fine.

**Links you can use:**
- YouTube: `https://www.youtube.com/watch?v=aqz-KE-bpKQ`, `https://youtu.be/aqz-KE-bpKQ`
- Vimeo: `https://vimeo.com/76979871`, a private link `https://vimeo.com/76979871/8272103f6e`
- Not allowed: `https://www.dailymotion.com/video/x7tgad0`, `https://example.org/film.mp4`, `http://www.youtube.com/watch?v=aqz-KE-bpKQ` (http), `https://www.youtube.com/channel/UC123`, `not a link`

**Checking a player's address:** right-click the video → Inspect, and read the `<iframe src="…">`; or run `[...document.querySelectorAll('iframe')].map(f => f.src)` in the console.

**A patient's link:** make an episode as Cara (**Episodes → New episode**, a made-up patient with an email, born **17/05/1980**), then open **http://localhost:3000/qa/messages** and the invite link in that email; **Continue**, then the date of birth 17 / 5 / 1980. For the phone size, use a window sized **390×844** (DevTools device toolbar).

**Make a questionnaire for the cases:** as Alex, **Questionnaires → New questionnaire** "S34 test <time>" → add a Question Set (patient), a page, and a **Yes / No** question "Do you smoke?" on it. Use this draft in T-01…T-08.

## Test cases

### The editor (Alex Author)

**T-01 · Add a Video**
1. On your draft's page, in **Add content**, click **Video**.
2. Expected: a card "Video" with the PATIENT badge and, in red, "No link yet · No length yet". The Settings panel on the left says **Video**, with Title "Video", **Link** (with the red "Paste the video's link, or choose one from the library."), **Choose from library**, **Length** (red "Add the video's length."), the **“I have watched this video” tick** switch (off), and no Required switch yet. The Settings panel has only **Settings** and **Logic** tabs (no Disclosures), and no "Always ask fresh", "Same as…" or "Save to library".
3. Change the Title to "Your anaesthetic" and press Tab: the card's title changes; the top bar says "All changes saved".

**T-02 · Paste a YouTube link**
1. In **Link**, paste `https://www.youtube.com/watch?v=aqz-KE-bpKQ`, press Tab.
2. Expected: under it, "Plays from YouTube (privacy-enhanced mode)". Type `10:35` in **Length**, Tab: the card reads "YouTube · Length 10:35" (grey, not red).
3. Click **Preview** (top right). Expected: as the patient sees it: the title, "Length 10:35", a player, and no tick. The player's address is exactly `https://www.youtube-nocookie.com/embed/aqz-KE-bpKQ`. Press play: the film plays (needs the internet), with YouTube's own captions button. Browser Back returns to the editor.

**T-03 · A `youtu.be` link**
1. Change the Link to `https://youtu.be/aqz-KE-bpKQ`, Tab.
2. Expected: still "Plays from YouTube (privacy-enhanced mode)"; in Preview the address is again `https://www.youtube-nocookie.com/embed/aqz-KE-bpKQ`.

**T-04 · Vimeo links**
1. Change the Link to `https://vimeo.com/76979871`, Tab. Expected: "Plays from Vimeo (no tracking)"; the card says "Vimeo · Length 10:35". In Preview the address is `https://player.vimeo.com/video/76979871?dnt=1`.
2. Change it to the private link `https://vimeo.com/76979871/8272103f6e`. Expected in Preview: `https://player.vimeo.com/video/76979871?dnt=1&h=8272103f6e`.

**T-05 · Any other link is refused, and Publish is held back**
1. Paste each "Not allowed" link above in turn (Tab after each).
2. Expected for each: the red "Only YouTube or Vimeo links can be used." under Link, and the card says "Not a YouTube or Vimeo link · Length 10:35" in red. In Preview, the video shows "This video can't be shown." and **no player** (no iframe at all).
3. With a refused link still there, click **Publish**. Expected: the window lists "<your Question Set> · The video “Your anaesthetic” has a link that isn't a YouTube or Vimeo video", and its Publish button is disabled. Cancel.
4. Empty the Link and Tab: Publish lists "… has no link: paste a YouTube or Vimeo link, or choose one from the library". Cancel. Put the YouTube link back.

**T-06 · The length**
1. Type `3:2` in Length, Tab. Expected: red "Type the length as minutes and seconds, for example 3:20, or 1:02:05 (4 hours at most).", and the saved length doesn't change (the card keeps its old length).
2. Type `4`, Tab: the card says "Length 4:00". Type `1:02:05`: "Length 1:02:05". Type `5:00:00`: refused (over 4 hours).
3. Empty it, Tab: the card says "No length yet" in red, and Publish lists "The video “Your anaesthetic” has no length". Cancel, then set `10:35` again.

**T-07 · Choose from the library, by tag and by words**
1. Add a second **Video** on the page. In its Settings click **Choose from library**.
2. Expected: a window "Choose a video from the library": a search box, a row of **Tags** buttons (e.g. Example, Fasting, Knee replacement, Prehab, The anaesthetic, …; each whole, not cut short) and the videos with their length, "Preopsis" or "Your trust", and their tags. No leaflets.
3. Click the tag **The anaesthetic**: only videos with that tag stay (e.g. "Example: Your anaesthetic explained"); "Example: Fasting before your operation" is gone. Click it again: all are back.
4. Type `fasting` in the search: only matching videos. Clear it.
5. Click **Use this** on "Example: Fasting before your operation". Expected: the window closes; the Title is "Example: Fasting before your operation", the Link is `https://www.youtube.com/watch?v=aqz-KE-bpKQ`, the Length `10:35`; the card says "YouTube · Length 10:35". "All changes saved".
6. Close and reopen the window: it starts afresh (no tag pressed, empty search).

**T-08 · The tick and Required**
1. On the first video, switch on **“I have watched this video” tick**. Expected: the card shows a box "I have watched this video (optional)"; a **Required** switch appears.
2. Switch on **Required**: "(optional)" goes from the card.
3. Switch the tick off: the box goes from the card, and the Required switch disappears. Switch the tick on again: Required is off (it was turned off with the tick). Switch Required on.
4. On the library video (T-07), switch on the tick only (optional).
5. **Publish** the questionnaire (no problems listed). Expected: "v1 · published"; the video's Settings are greyed out and **Choose from library** isn't shown.

**T-09 · The questionnaire keeps its own copy (library item retired)**
1. As **Ada Admin**, open **Content library** (top bar) and **Add video**: Title "S34 retire video <time>", link `https://youtu.be/aqz-KE-bpKQ`, Length `2:30`, a tag of your own (e.g. "S34 <time>"). (Preopsis's "Example: …" videos are read-only for a trust, so use the trust's own.)
2. As Alex, on a draft (e.g. **New version** of the S34 questionnaire), add a **Video** and **Choose from library** → your new video. Expected: its title, link and "Length 2:30" fill in.
3. As Ada, **Retire** that video.
4. As Alex, reload the draft. Expected: the video still shows your title and "YouTube · Length 2:30"; in Preview it still plays from `youtube-nocookie.com`. **Choose from library** no longer offers it (nor its tag).
5. As Ada, **Bring back** the video: Choose from library offers it again.

**T-10 · No video in a repeating group**
1. On a draft page, add a **Group** and make it repeatable (its Settings). Select the Group: its own **Add content** has **Video** greyed out.
2. On a video's card, open **More actions (…)**: "Move into" doesn't offer the repeating group (an ordinary Group is offered).

**T-11 · Translations**
1. On a draft with a ticked video, click **Translate** (top bar), add **German**.
2. Expected: the video's title is listed (Text) and its tick's words, "I have watched this video" (Tick). Type German for both (e.g. "Ihre Narkose", "Ich habe dieses Video gesehen").
3. Open **Preview**, pick **German** in the language box: the video's title and the tick are in German.

### The patient (phone size, 390×844)

**T-12 · A required tick holds the set until it's ticked**
1. As Cara, make an episode on "S34 test … v1" for a made-up patient with an email (born 17/05/1980). Open their invite from **/qa/messages** in a 390×844 window; Continue; date of birth.
2. **Start** the Question Set. Expected: "Your anaesthetic" with a red asterisk, "Length 10:35", the player the width of the card (no part off the screen; the page doesn't scroll sideways), then the box "I have watched this video". The library video below has its own (optional) tick.
3. Answer "Do you smoke?", leave the required box empty, tap **Finish this section**. Expected: you stay on the screen; the video says "Please tick “I have watched this video” to go on." in red.
4. Tick it (tap the words or the box), then **Finish this section**: the set is complete. Expected: the optional library video's tick wasn't needed.
5. Play a video (needs the internet): it plays in the page; full screen works.
6. **Send my answers** and confirm.

**T-13 · Unticking, and reload**
1. On another new episode's link (phone size), start the set, tick the required box, wait for "Saved", then **reload** the page and reopen the set: the box is still ticked.
2. Untick it and tap **Finish this section**: refused again with the same message. Tick it and finish.

**T-14 · An optional tick never holds the patient up**
1. On a questionnaire where the only video's tick is optional (e.g. a new version with Required off), the patient finishes and sends without ticking.

### Staff

**T-15 · Validate shows the tick, read-only**
1. As Cara, open the T-12 episode → **Validate** on the Question Set.
2. Expected: each video shows its length and player; the required one's box is ticked and greyed out (it can't be changed) with "The patient ticked this."; the optional one unticked and greyed, with "The patient hasn't ticked this.". Validate works.
3. Open the **POA Summary**: no video appears there, and nothing about the ticks (it's unchanged by videos).

**T-16 · A second HQ doesn't pre-fill the tick**
1. As Cara, **New episode** for the **same patient** as T-12 (search them by name and date of birth), on the same questionnaire. Open their new invite (phone size).
2. Expected: the set opens on the pre-fill card ("Please check"), and "Do you smoke?" holds the earlier answer, but **"I have watched this video" is not ticked**, on either video.

**T-17 · Fill in for the patient**
1. On a new episode's page, **Fill in for the patient**: the videos and ticks show as for the patient; the required tick must be ticked before Finish; staff can tick it.

**T-18 · Who can change a video**
1. As **Sue Super** (Hospital A1 only) or **Cara**, open the S34 draft or published questionnaire: the video's settings are read-only and there's no **Choose from library** button.

**T-19 · A video shown only when a condition is met**
1. On a draft, give a required-ticked video a **Logic → Display: Conditionally** rule, e.g. when "Do you smoke?" is Yes. Publish, and open a patient's link at phone size.
2. Answer **No**: the video is hidden, and the set finishes and sends without the tick. On another episode, answer **Yes**: the video shows and its tick is required.

## Regression

- **R-01 (S32):** a Clinical Summary still adds inside a Group and on the page, and two in one Group still block Publish.
- **R-02 (S33):** the Content library page as Ada still adds a video with a YouTube link and refuses a Dailymotion one; its lengths read "10:35".
- **R-03 (S25):** a patient's second HQ still opens on the pre-fill card with their earlier answers (T-16 covers it with a video present).
- **R-04 (S23/S24):** on an ordinary Text question, "Always ask fresh" and "Save to library" are still offered.
- **R-05 (J3):** a questionnaire with no videos is filled in and sent by a patient at phone size as before.

## Out of scope

- Information blocks with formatted text (S35); the patient's Information list and the estimated time (S36).
- Whether the patient really played the video: not tracked, by design.
- The questionnaire editor's own layout at phone width (it's a desktop tool; the editor page scrolls sideways at 390px as before S34).
- A video failing to play because the computer is offline, or YouTube/Vimeo refusing the test film: report it, but it isn't the app's fault if the player's address is right.
