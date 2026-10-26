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
to this episode only: the patient sees it; the editor's Question Set header shows the same estimate."""

from playwright.sync_api import BrowserContext, Page, expect

import re

from support import block_video_sites, choose, invite_path, sign_in, unique


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
