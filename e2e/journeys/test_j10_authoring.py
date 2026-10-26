"""J10 Authoring additions (plan C5), built up session by session. S31 (capture blocks) was dropped,
so it starts here, in S32: Clinical Summaries; S34 completes it with a library video.

The author adds two Clinical Summaries in the editor, one inside the "Breathing" Group and one on the
page; each card says what it covers. A third, in the same Group, blocks Publish with the Question
Set named; deleted, the HQ publishes. The patient, on a phone, never sees a summary. The clinician,
validating, sees each summary box where the author placed it, listing only its own notes (each note
once, under the nearest summary), and types a comment in each. The POA Summary shows one block per
summary, with its own title, notes and comment.

S34: the author also adds a Video from the content library, found by its tag: its title and length
fill in, and its "I have watched this video" tick is made required. The patient, on a phone (WebKit),
sees the video (the player's address only: the real video site is blocked, so this runs offline) and
can't finish the set until it's ticked; the clinician sees that it was ticked, and can't change it.

S35: the author also adds an Information block with a heading, a list and a link, and a <script>
typed into it; the preview shows it formatted as they type. The patient, on a phone, sees it
formatted (the link opens in a new tab), and the <script> as text: it never runs.

S36: the patient's Information and the estimated time. Alex gives a procedure a library video and a
trust leaflet on the Procedures page; the hospital has a phone number and email for patients. A new
episode for that procedure: the patient, on a phone, sees each Question Set's estimated time ("About
4 minutes": five questions at the deployment's 15 seconds and a 150-second video; the clinician-only
question and the summary don't count), and under Information on their home the video's player, the
leaflet's link and the hospital's phone and email, with no sideways scroll. Cara adds another leaflet
to this episode only: the patient sees it; the editor's Question Set header shows the same estimate.

S37: a Clinical Summary can't have a condition. Its settings offer no Logic tab (a question's still
do). A draft whose summary has one (made by the API) can't be published: the Publish window names the
summary, and its Settings say why; the author removes the condition in the Code tab, and it publishes.

S38: a score across Question Sets. In the second set, Alex adds a Calculation adding up two questions
from the first set (listed "Sleep › Sleep habits") and one of its own, plus "Add 1 point when Age is
over 40" from the Patient menu, with Low and High bands, High carrying a disclosure. The set's preview
says the earlier answers count as unanswered (the score is Incomplete) until they're given there. The
patient (46, on a phone) answers both sets; Cara's Validate shows "3 (High)" and the disclosure in the
set's summary box, and the POA Summary lists it. Then a ready-made STOP-Bang: inserted with the BMI
from the first set, its questions locked as published; changing it isn't offered.

S39: the ASA guide. Alex adds Type 2 diabetes (E11.9) to a disclosure: the ASA guide offers its examples
(well controlled II, poorly controlled III, and the children's), fills in nothing, and Save asks for a
choice; "poorly controlled" gives ASA III with that reason, kept after a reload. COPD (one example)
fills in ASA III by itself; changing the grade by hand is kept. The clinician's Anaesthetic plan set
shows its condition as "Suggested ASA is at least III"; a patient set's condition doesn't offer it."""

from playwright.sync_api import BrowserContext, Page, expect

import re

from support import block_video_sites, choose, invite_path, pick, sign_in, unique


def test_two_clinical_summaries_from_the_editor_to_the_poa_summary(page: Page, phone: BrowserContext, as_user):
    alex = as_user("Alex Author")
    hq = alex.draft_hq("summaries-hq.json", f"E2E J10 {unique()}")
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/questionnaires/{hq['id']}")
    page.get_by_role("complementary").get_by_text("About you").click()
    page.get_by_role("complementary").get_by_text("Heart").click()
    main = page.get_by_role("main")
    # The page's old "Clinical summary" switch is gone: a summary is an element now.
    expect(main.get_by_role("switch", name="Clinical summary")).to_have_count(0)

    # A summary inside the Group: select the Group, then its own Add bar.
    group = main.get_by_test_id("card-g_breath")
    group.get_by_text("Breathing", exact=True).click()
    group.get_by_role("button", name="Clinical Summary").click()
    breathing = main.locator("[data-testid^=card-cs_]").filter(has_text="Breathing summary")
    expect(breathing).to_contain_text("Covers the Group “Breathing”")
    expect(breathing).to_contain_text("Clinician")

    # A summary on the page (the Group isn't selected now, so only the page's Add bar offers it).
    main.get_by_role("button", name="Clinical Summary").click()
    heart = main.locator("[data-testid^=card-cs_]").filter(has_text="Heart summary")
    expect(heart).to_contain_text("Covers the page “Heart”")
    # Its title is changed in Settings.
    settings = page.get_by_role("tabpanel", name="Settings")
    settings.get_by_role("textbox", name="Title").fill("Heart overview")
    settings.get_by_role("textbox", name="Title").press("Tab")
    expect(main.locator("[data-testid^=card-cs_]").filter(has_text="Heart overview")).to_contain_text("Covers the page “Heart”")
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")
    # No Logic on a summary (S37): no Logic tab and no "Displays when" on its card; a question still has Logic.
    expect(page.get_by_role("tab", name="Settings", exact=True)).to_be_visible()
    expect(page.get_by_role("tab", name="Logic", exact=True)).to_have_count(0)
    expect(main.locator("[data-testid^=card-cs_]").filter(has_text="Heart overview")).not_to_contain_text("Displays when")
    main.get_by_test_id("card-q_angina").click()
    expect(page.get_by_role("tab", name="Logic", exact=True)).to_be_visible()

    # A second summary in the same Group blocks Publish, naming the Question Set.
    group.get_by_text("Breathing", exact=True).click()
    group.get_by_role("button", name="Clinical Summary").click()
    expect(main.locator("[data-testid^=card-cs_]").filter(has_text="Breathing summary")).to_have_count(2)
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    expect(dialog.get_by_test_id("publish-problems")).to_contain_text(
        "About you · Two clinical summaries cover the same Group “Breathing”: keep one")
    expect(dialog.get_by_role("button", name="Publish")).to_be_disabled()
    dialog.get_by_role("button", name="Cancel").click()

    # Deleted, the HQ publishes.
    main.locator("[data-testid^=card-cs_]").filter(has_text="Breathing summary").last.get_by_role("button", name="Delete").click()
    page.get_by_role("dialog", name="Delete this?").get_by_role("button", name="Delete").click()
    expect(main.locator("[data-testid^=card-cs_]")).to_have_count(2)
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")

    # A library video (S34), found by its tag: its title and length fill in; its tick is made required.
    main.get_by_role("button", name="Video", exact=True).last.click()  # the page's Add bar, below every card
    settings = page.get_by_role("tabpanel", name="Settings")
    settings.get_by_role("button", name="Choose from library").click()
    picker = page.get_by_role("dialog", name="Choose a video from the library")
    picker.get_by_test_id("video-tags").get_by_role("button", name="The anaesthetic", exact=True).click()
    expect(picker.get_by_test_id("library-video-Example: Fasting before your operation")).to_have_count(0)
    picker.get_by_test_id("library-video-Example: Your anaesthetic explained").get_by_role("button", name="Use this").click()
    expect(picker).to_be_hidden()
    video = main.locator("[data-testid^=card-q_]").filter(has_text="Example: Your anaesthetic explained")
    expect(video).to_contain_text("YouTube · Length 10:35")
    expect(settings.get_by_role("textbox", name="Length")).to_have_value("10:35")
    settings.get_by_role("switch", name="“I have watched this video” tick").check()
    settings.get_by_role("switch", name="Required").check()
    expect(video).to_contain_text("I have watched this video")
    expect(video).not_to_contain_text("(optional)")
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")

    # An Information block (S35): a heading, a list and a link, and a <script> that must stay text.
    main.get_by_role("button", name="Information", exact=True).last.click()
    settings = page.get_by_role("tabpanel", name="Settings")
    settings.get_by_role("textbox", name="Text").fill(
        "## On the day\n- Bring your inhaler\n- Bring your glasses\n"
        "Read [the hospital's guide](https://example.org/guide).\n<script>window.__xss=1</script>")
    preview = settings.get_by_test_id("information-preview")
    expect(preview.get_by_role("heading", name="On the day")).to_be_visible()
    expect(preview.get_by_role("listitem")).to_have_text(["Bring your inhaler", "Bring your glasses"])
    expect(preview.get_by_role("link", name="the hospital's guide")).to_have_attribute("href", "https://example.org/guide")
    expect(preview).to_contain_text("<script>window.__xss=1</script>")
    settings.get_by_role("textbox", name="Text").press("Tab")
    expect(main.locator("[data-testid^=card-q_]").filter(has_text="On the day")).to_contain_text("Bring your glasses")
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    dialog.get_by_role("button", name="Publish").click()
    expect(page.get_by_role("banner").get_by_text("v1 · published")).to_be_visible()

    # The patient, on their phone, answers Yes to both: no summary, nor its comments box, anywhere.
    episode = as_user("Cara Clinician").episode(hq, f"J10 {unique()}")
    block_video_sites(phone)
    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_test_id("set-About you").get_by_role("button", name="Start").click()
    choose(patient, "Do you get chest pain (angina)?", "Yes")
    choose(patient, "Do you have asthma?", "Yes")
    expect(patient.get_by_text("summary")).to_have_count(0)
    expect(patient.get_by_text("Heart overview")).to_have_count(0)
    expect(patient.get_by_text("Clinical comments")).to_have_count(0)
    # The video (S34): the player through youtube-nocookie.com, and the required tick.
    player = patient.locator("iframe[title='Example: Your anaesthetic explained']")
    expect(player).to_have_attribute("src", "https://www.youtube-nocookie.com/embed/aqz-KE-bpKQ")
    expect(player).to_have_attribute("referrerpolicy", "strict-origin-when-cross-origin")
    expect(patient.get_by_text("Length 10:35")).to_be_visible()
    # The Information block (S35), formatted; the typed <script> is only text.
    expect(patient.get_by_role("heading", name="On the day")).to_be_visible()
    expect(patient.locator(".sj-info li")).to_have_text(["Bring your inhaler", "Bring your glasses"])
    guide = patient.get_by_role("link", name="the hospital's guide")
    expect(guide).to_have_attribute("href", "https://example.org/guide")
    expect(guide).to_have_attribute("target", "_blank")
    expect(patient.get_by_text("<script>window.__xss=1</script>")).to_be_visible()
    assert patient.evaluate("window.__xss") is None, "the typed <script> ran"
    patient.get_by_role("button", name="Finish this section").click()
    expect(patient.get_by_role("checkbox", name="I have watched this video")).to_be_visible()
    expect(patient.get_by_text("Please tick “I have watched this video” to go on.")).to_be_visible()
    patient.get_by_role("checkbox", name="I have watched this video").check()
    patient.get_by_role("button", name="Finish this section").click()
    patient.get_by_role("button", name="Send my answers").click()
    patient.get_by_role("dialog", name="Send your answers?").get_by_role("button", name="Send my answers").click()
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()

    # The clinician validates: each box where the author placed it, with only its own note.
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    page.get_by_role("row", name="About you").get_by_role("link", name="Validate").click()
    breathing_box = page.locator(".sj-summary").filter(has_text="Breathing summary")
    heart_box = page.locator(".sj-summary").filter(has_text="Heart overview")
    expect(breathing_box.get_by_role("listitem")).to_have_text(["Asthma"])
    expect(heart_box.get_by_role("listitem")).to_have_text(["Angina"])
    # Where the author placed them: the Group's box inside the Group, below its question (the page's
    # sits in the clinician column, beside the last row, as clinician items do).
    asthma = page.get_by_role("radiogroup", name="Do you have asthma?")
    assert asthma.bounding_box()["y"] < breathing_box.bounding_box()["y"]
    # The video, and that the patient ticked it: read-only for the clinician (S34).
    expect(page.get_by_role("checkbox", name="I have watched this video")).to_be_checked()
    expect(page.get_by_role("checkbox", name="I have watched this video")).to_be_disabled()
    expect(page.locator("[data-testid^=video-ticked-]")).to_have_text("The patient ticked this.")
    breathing_box.get_by_role("textbox").fill("Uses a blue inhaler.")
    heart_box.get_by_role("textbox").fill("Refer to cardiology.")
    heart_box.get_by_role("textbox").press("Tab")
    expect(page.get_by_test_id("validate-save-status")).to_have_text("Saved")
    page.get_by_role("button", name="Validate").click()
    expect(page.get_by_role("row", name="About you")).to_contain_text("Validated by Cara Clinician")

    # The POA Summary: one block per summary, with its own title, note and comment.
    page.get_by_role("link", name="POA Summary").click()
    about = page.get_by_role("region", name="About you")
    breathing_block = about.get_by_test_id("poa-summary-Breathing summary")
    heart_block = about.get_by_test_id("poa-summary-Heart overview")
    expect(breathing_block).to_contain_text("Asthma")
    expect(breathing_block).to_contain_text("Clinical comments: Uses a blue inhaler.")
    expect(breathing_block).not_to_contain_text("Angina")
    expect(heart_block).to_contain_text("Angina")
    expect(heart_block).to_contain_text("Clinical comments: Refer to cardiology.")
    expect(heart_block).not_to_contain_text("Asthma")
    # The POA Summary is unchanged by the video (S34): no block, and no "change" for its tick.
    expect(about).not_to_contain_text("Your anaesthetic explained")


def test_procedure_information_and_estimated_time(page: Page, phone: BrowserContext, as_user):
    u = unique()
    ada, alex, cara = as_user("Ada Admin"), as_user("Alex Author"), as_user("Cara Clinician")
    # The trust's own leaflets (the content library, S33), the hospital's contact details for patients
    # (Settings, S10), and a procedure offered at every hospital.
    leaflet = f"E2E Fasting leaflet {u}"
    extra = f"E2E Parking leaflet {u}"
    for title in (leaflet, extra):
        ada.call("POST", ada.o("/content"), {"kind": "leaflet", "title": title, "url": f"https://example.org/{u.replace(' ', '-')}/{title[4:11].lower()}.pdf", "tags": []})
    hospital_id = ada.base.split("/h/")[1]
    ada.call("PATCH", ada.o("/settings"), {"hospitalId": hospital_id, "values": {"contact.phone": "0100 555 0199", "contact.email": "poa@hospital-a1.example"}})
    procedure = f"E2E Bunion {u}"
    alex.call("POST", alex.o("/procedures"), {"name": procedure, "code": "", "specialty": "", "hospitalId": None})
    hq = alex.published_hq("information-time-hq.json", f"E2E Info {u}")

    # Alex gives the procedure a video and a leaflet from the library, on the Procedures page.
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/procedures")
    page.get_by_test_id(f"procedure-{procedure}").get_by_role("button", name="Edit").click()
    form = page.get_by_test_id("procedure-form")
    picker = form.get_by_role("combobox", name="Add a video or leaflet")
    picker.click()
    picker.fill("Your anaesthetic")
    page.get_by_role("option", name="Example: Your anaesthetic explained (Video)").click()
    picker.click()
    picker.fill(leaflet)
    page.get_by_role("option", name=f"{leaflet} (Leaflet)").click()
    expect(form.get_by_test_id("procedure-info").get_by_role("button", name=re.compile("^Remove"))).to_have_count(2)
    form.get_by_role("button", name="Save changes").click()
    expect(page.get_by_test_id(f"procedure-{procedure}").get_by_test_id("procedure-info-summary")).to_have_text("1 video, 1 leaflet")

    # A new episode for it gets the list (S36 answers 2).
    episode = cara.episode(hq, f"J10 Info {u}", procedure=procedure)

    # The patient, on a phone: each set's estimated time…
    block_video_sites(phone)
    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_test_id("set-Before your operation").get_by_test_id("set-estimate")).to_have_text("About 4 minutes")
    expect(patient.get_by_test_id("set-Your health").get_by_test_id("set-estimate")).to_have_text("About 1 minute")
    # …with the status badge and button whole at phone width (S36: the row wraps instead).
    patient.get_by_test_id("set-Your health").get_by_role("button", name="Start").click()
    choose(patient, "Do you have diabetes?", "No")
    patient.get_by_role("button", name="Finish this section").click()
    badge = patient.get_by_test_id("set-Your health").get_by_text("Complete", exact=True)
    expect(badge).to_be_visible()
    assert badge.evaluate("e => e.scrollWidth <= e.clientWidth + 1"), "the Complete badge is cut short"
    review = patient.get_by_test_id("set-Your health").get_by_role("button", name="Review")
    assert review.evaluate("e => e.scrollWidth <= e.clientWidth + 1"), "the Review button is cut short"

    # …and under Information on their home: the video, the leaflet and the hospital's phone and email.
    patient.goto("/home")
    card = patient.get_by_test_id("home-episode").filter(has_text=hq["name"])
    card.get_by_role("button", name="Information").click()
    info = card.get_by_test_id("home-information-list")
    player = info.locator("iframe[title='Example: Your anaesthetic explained']")
    expect(player).to_have_attribute("src", "https://www.youtube-nocookie.com/embed/aqz-KE-bpKQ")
    expect(info.get_by_text("Length 10:35")).to_be_visible()
    link = info.get_by_role("link", name=leaflet)
    expect(link).to_have_attribute("target", "_blank")
    expect(link).to_have_attribute("href", re.compile(r"^https://example\.org/"))
    expect(info.get_by_test_id("info-contact")).to_contain_text("Questions? Contact Hospital A1")
    expect(info.get_by_role("link", name="0100 555 0199")).to_have_attribute("href", "tel:01005550199")
    expect(info.get_by_role("link", name="poa@hospital-a1.example")).to_have_attribute("href", "mailto:poa@hospital-a1.example")
    expect(info.get_by_role("link", name=extra)).to_have_count(0)
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth"), "the home scrolls sideways"

    # Cara adds another leaflet to this episode only; the patient sees it.
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    panel = page.get_by_test_id("episode-information")
    expect(panel.get_by_test_id("episode-info-Example: Your anaesthetic explained")).to_contain_text("From the procedure")
    add = panel.get_by_role("combobox", name="Add a video or leaflet for this patient")
    add.click()
    add.fill(extra)
    page.get_by_role("option", name=f"{extra} (Leaflet)").click()
    expect(panel.get_by_test_id(f"episode-info-{extra}")).to_contain_text("Added for this patient by Cara Clinician")
    patient.reload()
    card = patient.get_by_test_id("home-episode").filter(has_text=hq["name"])
    card.get_by_role("button", name="Information").click()
    expect(card.get_by_test_id("home-information-list").get_by_role("link", name=extra)).to_be_visible()

    # The editor's Question Set header shows the patient's estimate (and none for the clinician's set).
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/questionnaires/{hq['id']}")
    page.get_by_role("complementary").get_by_text("Before your operation").click()
    expect(page.get_by_role("main").get_by_test_id("set-estimate")).to_have_text("· About 4 minutes for the patient")
    page.get_by_role("complementary").get_by_text("Nurse checks").click()
    expect(page.get_by_role("main").get_by_test_id("set-estimate")).to_have_count(0)


def test_a_summary_with_a_condition_cannot_be_published(page: Page, as_user):
    alex = as_user("Alex Author")
    hq = alex.draft_hq("summary-condition-hq.json", f"E2E J10 S37 {unique()}")
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/questionnaires/{hq['id']}")
    page.get_by_role("complementary").get_by_text("About you").click()
    page.get_by_role("complementary").get_by_text("Heart").click()
    main = page.get_by_role("main")

    # Publish is refused, naming the Question Set and the summary.
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    expect(dialog.get_by_test_id("publish-problems")).to_contain_text(
        "About you · The clinical summary “Heart checks” has a condition: remove it (a summary is always shown)")
    expect(dialog.get_by_role("button", name="Publish")).to_be_disabled()
    dialog.get_by_role("button", name="Cancel").click()

    # The summary's Settings say why; its Logic tab is there only to remove the condition.
    card = main.get_by_test_id("card-cs_heart")
    expect(card).to_contain_text("Displays when")
    card.click()
    expect(page.get_by_test_id("summary-condition")).to_contain_text("A Clinical Summary can't have a condition")
    page.get_by_role("tab", name="Logic", exact=True).click()
    logic = page.get_by_role("tabpanel", name="Logic")
    logic.get_by_role("tab", name="Code").click()
    logic.get_by_role("textbox", name="Visible if").fill("")
    logic.get_by_role("button", name="Save logic").click()
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")

    # Removed: no Logic tab, no warning, and the HQ publishes.
    expect(page.get_by_role("tab", name="Logic", exact=True)).to_have_count(0)
    expect(page.get_by_test_id("summary-condition")).to_have_count(0)
    expect(card).not_to_contain_text("Displays when")
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    expect(dialog.get_by_test_id("publish-problems")).to_have_count(0)
    dialog.get_by_role("button", name="Publish").click()
    expect(dialog).to_be_hidden()



def test_a_score_across_question_sets(page: Page, phone: BrowserContext, as_user):
    """S38: a score in the second Question Set adds up two questions from the first and one of its
    own, plus a point for the patient's age, with two bands, the higher carrying a disclosure."""
    u = unique()
    alex = as_user("Alex Author")
    hq = alex.draft_hq("scores-hq.json", f"E2E J10 scores {u}")
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/questionnaires/{hq['id']}")
    side = page.get_by_role("complementary")
    side.get_by_text("Neck", exact=True).click()
    side.get_by_text("Collar", exact=True).click()
    main = page.get_by_role("main")
    main.get_by_role("button", name="Calculation", exact=True).last.click()
    settings = page.get_by_role("tabpanel", name="Settings")
    settings.get_by_role("textbox", name="Question text").fill("Sleep risk")
    settings.get_by_role("textbox", name="Question text").press("Tab")

    # The questions to add up: the earlier set's grouped "Sleep › Sleep habits", this set's as "This Question Set".
    questions = settings.get_by_label("Add up the scores of")
    questions.click()
    listbox = page.get_by_role("listbox")
    expect(listbox.get_by_text("Sleep › Sleep habits")).to_be_visible()
    expect(listbox.get_by_text("This Question Set")).to_be_visible()
    for q in ("Do you snore loudly?", "Do you often feel tired in the day?", "Is your shirt collar large?"):
        page.get_by_role("option", name=q, exact=True).click()
    questions.press("Escape")

    # One point when the patient is over 40: from the Patient menu, never a typed formula.
    lines = settings.get_by_test_id("points-lines")
    lines.get_by_role("button", name="Add points line").click()
    lines.get_by_role("combobox", name="Question").click()
    expect(page.get_by_role("listbox").get_by_text("Measurements")).to_be_visible()  # Sleep's BMI question
    page.get_by_role("option", name="Age", exact=True).click()
    pick(page, "Comparison", ">")
    lines.get_by_role("textbox", name="Value").fill("40")
    lines.get_by_role("textbox", name="Value").press("Tab")
    card = main.locator("[data-testid^=card-q_]").filter(has_text="Total of the scores")  # its title is being edited in place
    expect(card).to_contain_text("Total of the scores of 3 questions (2 in earlier Question Sets), plus 1 points line")
    # A line removed while another is unfinished is saved without it (verification 1, F1).
    lines.get_by_role("button", name="Add points line").click()
    lines.get_by_role("combobox", name="Question").nth(1).click()
    page.get_by_role("option", name="Sex", exact=True).click()
    pick_in = lines.get_by_role("combobox", name="Value")
    pick_in.click()
    page.get_by_role("option", name="Male", exact=True).click()
    expect(card).to_contain_text("plus 2 points lines")
    lines.get_by_role("button", name="Add points line").click()  # left unfinished
    lines.get_by_role("button", name="Remove line").nth(1).click()  # the Sex line
    expect(card).to_contain_text("plus 1 points line")
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")
    page.reload()
    main.locator("[data-testid^=card-q_]").filter(has_text="Total of the scores").click()
    expect(main.locator("[data-testid^=card-q_]").filter(has_text="Total of the scores")).to_contain_text("plus 1 points line")

    # Two bands: Low up to 2, High above; High carries a disclosure.
    settings.get_by_role("button", name="Add band").click()
    settings.get_by_role("button", name="Add band").click()
    expect(settings.get_by_role("textbox", name="Band 2")).to_have_value("High")
    page.get_by_role("tab", name="Disclosures").click()
    page.get_by_role("tabpanel", name="Disclosures").get_by_role("button", name="Add").last.click()
    modal = page.get_by_role("dialog")
    modal.get_by_role("button", name="+ Add disclosure").click()
    modal.get_by_role("textbox", name="Note").fill("High risk of sleep apnoea")
    pick(page, "Category", "Respiratory")
    modal.get_by_role("button", name="Save output").click()
    modal.get_by_role("button", name="Done").click()
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")

    # This set's preview can't see Sleep's answers: it says so, and the score is Incomplete until they're given.
    page.goto(f"{base}/chapters/{hq['chapters']['Neck']}/preview")
    page.get_by_test_id("viewer-switch").get_by_text("Clinician").click()
    note = page.get_by_test_id("earlier-answers")
    expect(note).to_contain_text("Uses answers from earlier Question Sets, which count as unanswered here")
    scores = page.get_by_test_id("preview-scores")
    choose(page, "Is your shirt collar large?", "Yes")
    expect(scores).to_contain_text("Sleep risk: Incomplete")
    pick(page, "Sleep habits › Do you snore loudly?", "Yes (1 pt)")
    pick(page, "Sleep habits › Do you often feel tired in the day?", "No (0 pts)")
    expect(scores).to_contain_text("Sleep risk: 3 (High)")  # 1 + 0 + 1, and 1 for the sample patient's age (54)
    expect(page.get_by_test_id("outputs-panel")).to_contain_text("High risk of sleep apnoea")
    summary_box = page.locator(".sj-summary").filter(has_text="Sleep risk summary")
    expect(summary_box.get_by_role("listitem")).to_have_text(["High risk of sleep apnoea"])
    # Clear answers clears this set's answers only: the earlier answers given above still count (test report 1, F-1).
    page.get_by_role("button", name="Clear answers").click()
    expect(scores).to_contain_text("Sleep risk: Incomplete")
    expect(summary_box).not_to_contain_text("High risk of sleep apnoea")
    choose(page, "Is your shirt collar large?", "Yes")
    expect(scores).to_contain_text("Sleep risk: 3 (High)")
    expect(summary_box.get_by_role("listitem")).to_have_text(["High risk of sleep apnoea"])
    # Sleep is shown only to women: for a man its answers don't count, and the box says so (test report 2, F-1).
    pick(page, "Sex", "Male")
    expect(note.get_by_test_id("earlier-hidden")).to_have_text("The sample patient wouldn't see “Sleep”, so its answers don't count.")
    expect(scores).to_contain_text("Sleep risk: Incomplete")
    expect(page.get_by_test_id("outputs-panel")).not_to_contain_text("High risk of sleep apnoea")
    pick(page, "Sex", "Female")
    expect(note.get_by_test_id("earlier-hidden")).to_have_count(0)
    expect(scores).to_contain_text("Sleep risk: 3 (High)")

    # Published, the patient (46, on a phone) answers both sets: Yes, No, then Yes.
    alex.call("POST", alex.h(f"/questionnaires/{hq['id']}/publish"), {"versionId": hq["versionId"]})
    episode = as_user("Cara Clinician").episode(hq, f"J10 Scores {u}")
    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_test_id("set-Sleep").get_by_role("button", name="Start").click()
    choose(patient, "Do you snore loudly?", "Yes")
    choose(patient, "Do you often feel tired in the day?", "No")
    patient.get_by_role("button", name="Next").click()
    patient.get_by_label("What is your height?").fill("170")
    patient.get_by_label("What is your weight?").fill("70")
    patient.get_by_role("button", name="Finish this section").click()
    patient.get_by_test_id("set-Neck").get_by_role("button", name="Start").click()
    choose(patient, "Is your shirt collar large?", "Yes")
    expect(patient.get_by_text("Sleep risk")).to_have_count(0)  # clinicians' only
    patient.get_by_role("button", name="Finish this section").click()
    patient.get_by_role("button", name="Send my answers").click()
    patient.get_by_role("dialog", name="Send your answers?").get_by_role("button", name="Send my answers").click()
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()

    # Validate shows the score from both sets' answers, and the High band's disclosure in the summary box.
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    page.get_by_role("row", name="Neck").get_by_role("link", name="Validate").click()
    expect(page.get_by_text("3 (High)")).to_be_visible()
    expect(page.locator(".sj-summary").filter(has_text="Sleep risk summary").get_by_role("listitem")).to_have_text(["High risk of sleep apnoea"])
    page.get_by_role("button", name="Validate").click()
    expect(page.get_by_role("row", name="Neck")).to_contain_text("Validated by Cara Clinician")

    # The POA Summary lists it.
    page.get_by_role("link", name="POA Summary").click()
    expect(page.get_by_role("region", name="Neck")).to_contain_text("High risk of sleep apnoea")


def test_a_ready_made_stop_bang(page: Page, as_user):
    """S38: STOP-Bang inserted ready-made, with the BMI from an earlier Question Set, as published and locked."""
    alex = as_user("Alex Author")
    hq = alex.draft_hq("scores-hq.json", f"E2E J10 STOP-Bang {unique()}")
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/questionnaires/{hq['id']}")
    side = page.get_by_role("complementary")
    side.get_by_text("Neck", exact=True).click()
    side.get_by_text("Collar", exact=True).click()
    main = page.get_by_role("main")
    main.get_by_role("button", name="Ready-made score").last.click()
    window = page.get_by_role("dialog", name="Insert a ready-made score")
    expect(window).to_contain_text("University Health Network")
    # The only BMI is the earlier set's, so it's chosen already.
    expect(window.get_by_role("combobox", name="BMI from")).to_have_value("BMI Calculator (BMI)")
    window.get_by_role("button", name="Insert").click()
    group = main.get_by_test_id("card-stopbang")
    expect(group.get_by_test_id("ready-made").first).to_have_text("Ready-made: STOP-Bang")
    expect(group).to_contain_text("Do you snore loudly (loud enough to be heard through closed doors")
    # Its questions can't be changed, moved or deleted on their own; the Settings say why.
    snoring = main.get_by_test_id("card-stopbang_snoring")
    expect(snoring.get_by_role("button", name="Delete")).to_have_count(0)
    snoring.click()
    settings = page.get_by_role("tabpanel", name="Settings")
    expect(page.get_by_test_id("ready-score-note")).to_contain_text("can't be changed")
    expect(settings.get_by_role("textbox", name="Question text")).to_be_disabled()
    expect(page.get_by_role("tab", name="Logic", exact=True)).to_have_count(0)
    expect(page.get_by_role("tab", name="Disclosures", exact=True)).to_be_visible()
    # The score: its published bands, with the high-risk rule in words.
    main.get_by_test_id("card-stopbang_score").click()
    expect(page.get_by_role("tabpanel", name="Settings")).to_contain_text(
        "Also High risk when: Yes to 2 or more of the 4 STOP questions, and male, a BMI over 35 or a large neck")
    # A second one is refused; the HQ publishes.
    main.get_by_role("button", name="Ready-made score").last.click()
    window.get_by_role("button", name="Insert").click()
    expect(window).to_contain_text("This HQ already has STOP-Bang")
    window.get_by_role("button", name="Cancel").click()
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    dialog.get_by_role("button", name="Publish").click()
    expect(page.get_by_role("banner").get_by_text("v1 · published")).to_be_visible()


def test_a_repeated_key_does_not_break_the_score_editor(page: Page, as_user):
    """S38 verification 1, F2: a draft repeating a key in two Question Sets (Publish refuses it) still
    opens a score's settings, listing the key once."""
    alex = as_user("Alex Author")
    hq = alex.draft_hq("scores-hq.json", f"E2E J10 repeated key {unique()}")
    cid = hq["chapters"]["Neck"]
    chapter = alex.call("GET", alex.h(f"/chapters/{cid}"))
    elements = chapter["content"]["pages"][0]["elements"]
    elements.append({"type": "radiogroup", "name": "q_e2e_snore", "title": "Do you snore (asked again)?",
                     "choices": [{"value": "o_yes", "text": "Yes", "score": 1}, {"value": "o_no", "text": "No", "score": 0}]})
    elements.append({"type": "expression", "name": "q_e2e_risk", "title": "Repeat risk", "clinicianOnly": True, "calcKind": "score",
                     "expression": "iif(allAnswered('q_e2e_snore'), score('q_e2e_snore'), '')"})
    alex.call("PUT", alex.h(f"/chapters/{cid}/content"), {"content": chapter["content"], "revision": chapter["revision"]})
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/questionnaires/{hq['id']}")
    side = page.get_by_role("complementary")
    side.get_by_text("Neck", exact=True).click()
    side.get_by_text("Collar", exact=True).click()
    page.get_by_role("main").get_by_test_id("card-q_e2e_risk").click()
    settings = page.get_by_role("tabpanel", name="Settings")
    settings.get_by_label("Add up the scores of").click()
    expect(page.get_by_role("option", name="Do you snore (asked again)?", exact=True)).to_have_count(1)
    expect(page.get_by_role("option", name="Do you snore loudly?", exact=True)).to_have_count(0)
    page.keyboard.press("Escape")
    settings.get_by_test_id("points-lines").get_by_role("button", name="Add points line").click()
    settings.get_by_test_id("points-lines").get_by_role("combobox", name="Question").click()
    expect(page.get_by_role("option", name="Do you snore (asked again)?", exact=True)).to_have_count(1)
    expect(page.get_by_text("Something went wrong")).to_have_count(0)



def test_a_set_shown_by_a_score_follows_a_changed_earlier_answer(phone: BrowserContext, as_user, patient_link):
    """S38 verification 2, F4: "Sleep follow-up" is shown only when a score in Tiredness, worked out
    from Snoring's answer, is 1 or more. The patient answers Snoring Yes and Tiredness (saving the
    score as 1), then changes Snoring to No: the follow-up goes, though Tiredness still holds 1."""
    alex = as_user("Alex Author")
    hq = alex.published_hq("score-condition-hq.json", f"E2E J10 score condition {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J10 Score condition {unique()}")
    api = patient_link(episode)
    api.answer("Snoring", {"q_e2e_sa": "o_no"})
    api.answer("Tiredness", {"q_e2e_sb": "o_yes", "q_e2e_ss": 1})  # saved when Snoring was Yes
    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_test_id("set-Tiredness")).to_be_visible()
    expect(patient.get_by_test_id("set-Sleep follow-up")).to_have_count(0)
    # The other way: Snoring Yes now, Tiredness still saved with 0: the follow-up is shown, and must be answered.
    api.answer("Snoring", {"q_e2e_sa": "o_yes"})
    api.answer("Tiredness", {"q_e2e_sb": "o_yes", "q_e2e_ss": 0})
    patient.reload()
    expect(patient.get_by_test_id("set-Sleep follow-up")).to_be_visible()
    expect(patient.get_by_role("button", name="Send my answers")).to_be_disabled()


def test_the_asa_guide_fills_in_or_asks_for_the_grade(page: Page, as_user):
    alex = as_user("Alex Author")
    hq = alex.draft_hq("asa-hq.json", f"E2E J10 ASA {unique()}")
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/questionnaires/{hq['id']}")
    page.get_by_role("complementary").get_by_text("Your heart").click()
    page.get_by_role("complementary").get_by_text("Heart", exact=True).click()
    main = page.get_by_role("main")
    main.get_by_test_id("card-q_heart_failure").click()
    page.get_by_role("tab", name="Disclosures").click()
    page.get_by_role("tabpanel", name="Disclosures").get_by_role("button", name="Add").last.click()  # the "No" option
    modal = page.get_by_role("dialog")
    modal.get_by_role("button", name="+ Add disclosure").click()

    # Diabetes: the grade depends on control, so the guide asks and fills in nothing.
    modal.get_by_text("ICD10", exact=True).click()
    modal.get_by_placeholder("Search ICD10 by code or description").fill("E11.9")
    modal.get_by_role("button", name=re.compile(r"^E11\.9 Type 2 diabetes")).click()
    guide = modal.get_by_test_id("asa-guide")
    expect(guide.get_by_role("radio", name="ASA II: Well-controlled diabetes (Adults)")).to_be_visible()
    expect(guide.get_by_role("radio", name="ASA III: Poorly controlled diabetes (Adults)")).to_be_visible()
    expect(guide.get_by_role("radio", name="ASA II: Non-insulin-dependent diabetes (Children)")).to_be_visible()
    expect(modal.get_by_label("ASA grade", exact=True)).to_have_value("")
    modal.get_by_role("button", name="Save output").click()
    expect(modal.get_by_role("alert")).to_contain_text("Choose the ASA grade from the ASA guide, or No ASA grade.")
    guide.get_by_role("radio", name="ASA III: Poorly controlled diabetes (Adults)").check()
    expect(modal.get_by_label("ASA grade", exact=True)).to_have_value("III")
    expect(modal.get_by_test_id("asa-reason")).to_contain_text("“Poorly controlled diabetes”")
    modal.get_by_role("button", name="Save output").click()
    expect(modal).to_contain_text("ASA III")
    expect(modal).to_contain_text("(Poorly controlled diabetes)")

    # COPD: one example, so ASA III fills in by itself; a grade changed by hand is kept, without the reason.
    modal.get_by_role("button", name="+ Add disclosure").click()
    modal.get_by_placeholder("Search SNOMED by code or description").fill("13645005")
    modal.get_by_role("button", name=re.compile(r"^13645005 Chronic obstructive")).click()
    expect(modal.get_by_label("ASA grade", exact=True)).to_have_value("III")
    expect(modal.get_by_test_id("asa-reason")).to_contain_text("“COPD”")
    modal.get_by_label("ASA grade", exact=True).click()
    page.get_by_role("option", name="IV", exact=True).click()
    expect(modal.get_by_test_id("asa-reason")).to_have_count(0)
    modal.get_by_role("button", name="Save output").click()
    expect(modal).to_contain_text("ASA IV")
    modal.get_by_role("button", name="Done").click()
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")

    # Kept after a reload: the grade and the guide's words are copied into the HQ.
    page.reload()
    page.get_by_role("complementary").get_by_text("Heart", exact=True).click()
    main.get_by_test_id("card-q_heart_failure").click()
    page.get_by_role("tab", name="Disclosures").click()
    page.get_by_role("tabpanel", name="Disclosures").get_by_role("button", name="2").click()  # the "No" option's two
    expect(modal).to_contain_text("(Poorly controlled diabetes)")
    expect(modal).to_contain_text("ASA IV")

    # Editing it later still offers the guide's examples for its code, the chosen one selected (test report 1, F2).
    modal.get_by_role("button", name="Edit").first.click()
    guide = modal.get_by_test_id("asa-guide")
    expect(guide.get_by_role("radio", name="ASA III: Poorly controlled diabetes (Adults)")).to_be_checked()
    guide.get_by_role("radio", name="ASA II: Well-controlled diabetes (Adults)").check()
    expect(modal.get_by_test_id("asa-reason")).to_contain_text("“Well-controlled diabetes”")
    modal.get_by_role("button", name="Cancel").click()

    # Removing the code a grade came from takes the grade and its words with it (test report 1, F1).
    modal.get_by_role("button", name="+ Add disclosure").click()
    modal.get_by_placeholder("Search SNOMED by code or description").fill("13645005")
    modal.get_by_role("button", name=re.compile(r"^13645005 Chronic obstructive")).click()
    modal.get_by_placeholder("Search SNOMED by code or description").fill("77176002")
    modal.get_by_role("button", name=re.compile(r"^77176002 Smoker")).click()
    expect(modal.get_by_label("ASA grade", exact=True)).to_have_value("III")  # a grade already there is kept
    modal.get_by_role("button", name="Remove 13645005").click()
    expect(modal.get_by_label("ASA grade", exact=True)).to_have_value("II")  # Smoker's only example now
    expect(modal.get_by_test_id("asa-reason")).to_contain_text("“Current smoker”")
    modal.get_by_role("button", name="Cancel").click()

    # "No ASA grade" stands when another code is added: the declined grade doesn't come back (verification 1, F1).
    modal.get_by_role("button", name="+ Add disclosure").click()
    modal.get_by_placeholder("Search SNOMED by code or description").fill("13645005")
    modal.get_by_role("button", name=re.compile(r"^13645005 Chronic obstructive")).click()
    guide = modal.get_by_test_id("asa-guide")
    guide.get_by_role("radio", name="No ASA grade").check()
    modal.get_by_placeholder("Search SNOMED by code or description").fill("49727002")
    modal.get_by_role("button", name=re.compile(r"^49727002 Cough")).click()
    expect(modal.get_by_label("ASA grade", exact=True)).to_have_value("")
    expect(guide.get_by_role("radio", name="No ASA grade")).to_be_checked()
    expect(modal.get_by_test_id("asa-reason")).to_have_count(0)
    modal.get_by_role("button", name="Cancel").click()
    modal.get_by_role("button", name="Done").click()

    # The clinician set's condition, in words; a patient set's condition doesn't offer the suggested ASA.
    page.reload()  # nothing selected: the Question Sets are listed again
    page.get_by_role("complementary").get_by_text("Anaesthetic plan").click()
    expect(main.get_by_label("Question", exact=True)).to_have_value("Suggested ASA")
    expect(main.get_by_label("Comparison", exact=True)).to_have_value("is at least")
    expect(main.get_by_label("Value", exact=True)).to_have_value("III")
    page.get_by_role("complementary").get_by_text("Your heart").click()
    main.get_by_role("radio", name="Conditionally").check()
    main.get_by_label("Question", exact=True).click()
    expect(page.get_by_role("option", name="Patient's age")).to_be_visible()
    expect(page.get_by_role("option", name="Suggested ASA")).to_have_count(0)
