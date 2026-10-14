"""J1 Author: create a questionnaire, add a Question Set, page and question, save, reload and see
it's kept. The preview shows the question."""

import re

from playwright.sync_api import Page, expect

from support import sign_in, unique


def test_author_builds_a_questionnaire_and_previews_it(page: Page):
    name = f"E2E J1 {unique()}"

    # Signed in (the dev stub locally, Cognito on QA) as a trust-wide super clinician, who picks
    # a hospital (S08).
    sign_in(page.context, "Alex Author", None)
    page.goto("/")
    page.get_by_role("region", name="Trust A").get_by_role("link", name="Hospital A1").click()
    expect(page.get_by_role("heading", name="Questionnaires")).to_be_visible()
    expect(page).to_have_url(re.compile(r"/o/[0-9a-f-]{36}/h/[0-9a-f-]{36}/questionnaires$"))

    # Create the questionnaire: the editor opens on the new draft.
    page.get_by_role("button", name="New Questionnaire").click()
    drawer = page.get_by_role("dialog", name="New Questionnaire")
    drawer.get_by_label("Name").fill(name)
    drawer.get_by_label("Description").fill("Made up for an end-to-end test.")
    drawer.get_by_role("button", name="Save").click()
    expect(page).to_have_url(re.compile(r"/questionnaires/[0-9a-f-]{36}$"))
    expect(drawer).to_be_hidden()
    header = page.get_by_role("banner")
    expect(header).to_contain_text(name)
    expect(header.get_by_text("v1 · draft")).to_be_visible()

    # A Question Set, a page on it, and a Text question.
    structure = page.get_by_role("complementary")
    structure.get_by_role("button", name="Add Question Set").click()
    expect(structure.get_by_text("Question Set 1")).to_be_visible()
    structure.get_by_role("button", name="Add Page").click()
    expect(structure.get_by_text("Page 1")).to_be_visible()
    page.get_by_role("main").get_by_role("button", name="Text", exact=True).click()

    # Its text is set in the Settings panel, which saves when the box loses focus.
    settings = page.get_by_role("tabpanel", name="Settings")
    settings.get_by_role("textbox", name="Question text").fill("What is your job?")
    settings.get_by_role("textbox", name="Question text").press("Tab")
    expect(page.get_by_role("textbox", name="Question text on the card")).to_have_value("What is your job?")
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")

    # Reload: the editor comes back to the same page, and the question is kept.
    expect(page).to_have_url(re.compile(r"\?set=.+&page=.+"))
    page.reload()
    expect(page.get_by_role("main").get_by_text("What is your job?")).to_be_visible()
    expect(page.get_by_role("complementary").get_by_text("Question Set 1")).to_be_visible()

    # The preview shows the question, as the patient would see it.
    page.get_by_role("link", name="Preview").click()
    expect(page.get_by_role("radio", name="Patient")).to_be_checked()
    expect(page.get_by_role("textbox", name="What is your job?")).to_be_visible()
