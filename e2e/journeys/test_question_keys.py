"""Question keys (S23): the editor's pre-fill label (PRE-05), "Always ask fresh", and Publish
refusing a published question whose options changed, in the browser and in Go."""

import re

from playwright.sync_api import Page, expect

from support import sign_in, unique


def open_page(page: Page, base: str, hq: dict, set_name: str, page_title: str) -> None:
    page.goto(f"{base}/questionnaires/{hq['id']}")
    structure = page.get_by_role("complementary")
    structure.get_by_text(set_name, exact=True).click()
    structure.get_by_text(page_title, exact=True).click()


def test_label_always_fresh_and_the_option_lock(page: Page, as_user):
    alex = as_user("Alex Author")
    earlier = alex.published_hq("workflow-hq.json", f"E2E Keys earlier {unique()}")
    hq = alex.draft_hq("workflow-hq.json", f"E2E Keys {unique()}")
    base = sign_in(page.context, "Alex Author")

    # "Do you smoke?" has the same key as in the earlier HQ: it pre-fills from it, by link.
    open_page(page, base, hq, "About you", "Smoking")
    card = page.get_by_test_id("card-q_smokes")
    label = card.get_by_test_id("prefill-label")
    expect(label).to_contain_text(re.compile(r"Pre-fills from \d+ other HQs?"))
    label.click()
    sources = page.get_by_test_id("prefill-sources")
    expect(sources.get_by_role("listitem").filter(has_text=earlier["name"])).to_contain_text("by link")
    page.keyboard.press("Escape")
    expect(sources).to_be_hidden()

    # From the keyboard too (test report 3): the badge is a button; Enter opens the list, Escape closes it.
    expect(label).to_have_role("button")
    label.focus()
    page.keyboard.press("Enter")
    expect(sources).to_be_visible()
    page.keyboard.press("Escape")
    expect(sources).to_be_hidden()

    # A new question pre-fills from nothing.
    page.get_by_role("main").get_by_role("button", name="Text", exact=True).click()
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")
    new_card = page.locator("[data-testid^=card-q_]").last
    expect(new_card.get_by_test_id("prefill-label")).to_have_text("New question: won't pre-fill")

    # Always ask fresh: a switch beside Clinical, a badge on the card instead of the label, kept on reload.
    card.click()
    settings = page.get_by_role("tabpanel", name="Settings")
    settings.get_by_role("switch", name="Always ask fresh").check()
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")
    expect(card.get_by_test_id("always-fresh")).to_be_visible()
    expect(card.get_by_test_id("prefill-label")).to_be_hidden()
    page.reload()
    expect(page.get_by_test_id("card-q_smokes").get_by_test_id("always-fresh")).to_be_visible()

    # Reword the published option "Yes": the Publish window refuses, naming the question and the option.
    page.get_by_test_id("card-q_smokes").click()
    option = page.get_by_role("textbox", name="Option 1")
    option.fill("Yes, every day")
    option.press("Tab")
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    expect(dialog.get_by_test_id("publish-problems")).to_contain_text(
        "“Do you smoke?”: the option “Yes” was reworded to “Yes, every day”. A published question's answers can't change")
    expect(dialog.get_by_role("button", name="Publish")).to_be_disabled()
    dialog.get_by_role("button", name="Cancel").click()

    # Put it back: it publishes, and "Always ask fresh" is kept in the published version.
    option.fill("Yes")
    option.press("Tab")
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    dialog.get_by_role("button", name="Publish").click()
    expect(dialog).to_be_hidden()
    expect(page.get_by_role("banner").get_by_text("v1 · published")).to_be_visible()
    expect(page.get_by_test_id("card-q_smokes").get_by_test_id("always-fresh")).to_be_visible()
