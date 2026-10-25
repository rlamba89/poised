"""J10 Authoring additions (plan C5), built up session by session. S31 (capture blocks) was dropped,
so it starts here, in S32: Clinical Summaries.

The author adds two Clinical Summaries in the editor, one inside the "Breathing" Group and one on the
page; each card says what it covers. A third, in the same Group, blocks Publish with the Question
Set named; deleted, the HQ publishes. The patient, on a phone, never sees a summary. The clinician,
validating, sees each summary box where the author placed it, listing only its own notes (each note
once, under the nearest summary), and types a comment in each. The POA Summary shows one block per
summary, with its own title, notes and comment."""

from playwright.sync_api import BrowserContext, Page, expect

from support import choose, invite_path, sign_in, unique


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
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    dialog.get_by_role("button", name="Publish").click()
    expect(page.get_by_role("banner").get_by_text("v1 · published")).to_be_visible()

    # The patient, on their phone, answers Yes to both: no summary, nor its comments box, anywhere.
    episode = as_user("Cara Clinician").episode(hq, f"J10 {unique()}")
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
