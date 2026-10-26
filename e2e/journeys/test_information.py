"""The Information element (S35, INF-03). It was the Statement.

The author types simple marks (a heading, bold, a list, a link, a picture link) and sees a live
preview; anything else they type (a <script>, a javascript: link) shows as text, never as a link or
something that runs. Old statements keep their text, and open in the editor as it. The patient, on a
phone, sees the block formatted, with the picture no wider than the screen and links that open in a
new tab; HTML that nobody could type in the editor (a hand-made or tampered HQ) is cleaned before it's
shown. The block's German translation shows in the Preview's German (patients can't choose a language
yet). Pictures come from https://pictures.example.org, answered offline by the test (support)."""

import re

from playwright.sync_api import BrowserContext, Page, expect

from support import choose, invite_path, pick, serve_test_pictures, sign_in, unique

def px(locator, prop: str = "font-size") -> float:
    """A computed CSS size in pixels, e.g. a heading's font size."""
    return float(locator.evaluate(f"e => getComputedStyle(e).getPropertyValue('{prop}')").removesuffix("px"))


EVIL_ANSWER = "[Sign in again](https://evil.example/login) ![](https://evil.example/p.png)"

TYPED = "\n".join([
    "# Fasting",
    "Do **not** eat after *midnight*.",
    "- Water is fine",
    "- Black coffee is fine",
    "1. Shower in the morning",
    "[Our fasting page](https://example.org/fasting)",
    "[Click](javascript:window.__xss=5)",
    "<script>window.__xss=6</script>",
    "<img src=x onerror=window.__xss=7>",
])


def test_the_author_formats_information_and_old_statements_keep_their_text(page: Page, as_user):
    serve_test_pictures(page.context)
    hq = as_user("Alex Author").draft_hq("information-hq.json", f"E2E Information {unique()}")
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/questionnaires/{hq['id']}")
    page.get_by_role("complementary").get_by_text("Before you come in").click()
    page.get_by_role("complementary").get_by_text("Your visit").click()
    main = page.get_by_role("main")

    # An old statement shows as before, and opens as its plain text; edited, it keeps that text.
    old = main.get_by_test_id("card-i_old")
    expect(old).to_contain_text("Please arrive 30 minutes early.")
    expect(old).to_contain_text("Bring a dressing gown & slippers.")
    old.get_by_text("Please arrive 30 minutes early.").click()
    settings = page.get_by_role("tabpanel", name="Settings")
    text = settings.get_by_role("textbox", name="Text")
    expect(text).to_have_value("Please arrive 30 minutes early.\nBring a dressing gown & slippers.")
    text.fill("Please arrive 30 minutes early.\nBring a dressing gown & slippers.\n**Thank you**")
    expect(settings.get_by_test_id("information-preview").locator("strong")).to_have_text("Thank you")
    text.press("Tab")
    expect(old.locator("strong")).to_have_text("Thank you")
    expect(old).to_contain_text("Bring a dressing gown & slippers.")
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")

    # HTML nobody could type here is cleaned on its card too: no picture, no javascript: link.
    bad = main.get_by_test_id("card-i_bad")
    expect(bad).to_contain_text("Imported text")
    expect(bad).to_contain_text("bad link")
    expect(bad.locator("img")).to_have_count(0)
    expect(bad.locator("a")).to_have_count(0)

    # A new Information block: the hint, the live preview, and typed HTML shown as text.
    main.get_by_role("button", name="Information", exact=True).last.click()
    settings = page.get_by_role("tabpanel", name="Settings")
    expect(settings.get_by_test_id("information-hints")).to_contain_text("How to format")
    text = settings.get_by_role("textbox", name="Text")
    text.fill(TYPED)
    preview = settings.get_by_test_id("information-preview")
    expect(preview.get_by_role("heading", name="Fasting", level=1)).to_have_count(0)
    expect(preview.get_by_role("heading", name="Fasting", level=2)).to_be_visible()
    expect(preview.locator("strong")).to_have_text("not")
    expect(preview.locator("em")).to_have_text("midnight")
    expect(preview.locator("ul > li")).to_have_text(["Water is fine", "Black coffee is fine"])
    expect(preview.locator("ol > li")).to_have_text(["Shower in the morning"])
    link = preview.get_by_role("link", name="Our fasting page")
    expect(link).to_have_attribute("href", "https://example.org/fasting")
    expect(link).to_have_attribute("target", "_blank")
    expect(preview.get_by_role("link")).to_have_count(1)
    expect(preview).to_contain_text("[Click](javascript:window.__xss=5)")
    expect(preview).to_contain_text("<script>window.__xss=6</script>")
    expect(preview).to_contain_text("<img src=x onerror=window.__xss=7>")
    expect(settings.get_by_test_id("information-refused")).to_contain_text("javascript:window.__xss=5")
    text.press("Tab")
    card = main.locator("[data-testid^=card-q_]").filter(has_text="Fasting")
    expect(card.get_by_role("heading", name="Fasting")).to_be_visible()
    fasting_size = (px(preview.get_by_role("heading", name="Fasting")), px(preview.get_by_role("heading", name="Fasting"), "line-height"))
    expect(card).to_contain_text("<script>window.__xss=6</script>")
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")
    assert page.evaluate("window.__xss") is None, "something the author typed ran"

    # The settings' preview of the Parking block: its heading's size, to compare with the Preview (test report 1, F1).
    main.get_by_test_id("card-i_parking").get_by_text("Pay at the machine").click()
    in_editor = page.get_by_role("tabpanel", name="Settings").get_by_test_id("information-preview").get_by_role("heading", name="Parking")
    editor_size = (px(in_editor), px(in_editor, "line-height"))

    # The Preview shows each block formatted; in German, the block's translation.
    page.get_by_role("link", name="Preview").click()
    expect(page).to_have_url(re.compile(r"/preview$"))
    expect(page.get_by_role("heading", name="Parking")).to_be_visible()
    # The same heading size as the author's preview, not SurveyJS's own (much bigger) one.
    assert (px(page.get_by_role("heading", name="Parking")), px(page.get_by_role("heading", name="Parking"), "line-height")) == editor_size
    fasting = page.get_by_role("heading", name="Fasting")
    assert (px(fasting), px(fasting, "line-height")) == fasting_size
    expect(page.get_by_role("img", name="The car park entrance")).to_be_visible()
    expect(page.get_by_role("heading", name="Fasting")).to_be_visible()
    expect(page.get_by_text("<script>window.__xss=6</script>")).to_be_visible()
    pick(page, "Language", "German")
    expect(page.get_by_role("heading", name="Parken")).to_be_visible()
    expect(page.locator("strong").filter(has_text="Parkhaus B")).to_be_visible()
    # Not translated: the English text.
    expect(page.get_by_role("heading", name="Fasting")).to_be_visible()
    assert page.evaluate("window.__xss") is None, "something in the HQ ran"
    page.go_back()

    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    page.get_by_role("dialog", name="Publish version 1?").get_by_role("button", name="Publish").click()
    expect(page.get_by_role("banner").get_by_text("v1 · published")).to_be_visible()


def test_the_patient_sees_information_formatted_on_a_phone(page: Page, phone: BrowserContext, as_user):
    hq = as_user("Alex Author").published_hq("information-hq.json", f"E2E Information {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"Information {unique()}")
    serve_test_pictures(phone)
    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_test_id("set-Before you come in").get_by_role("button", name="Start").click()

    # Formatted: the heading, bold, the list, links that open in a new tab.
    expect(patient.get_by_role("heading", name="Parking")).to_be_visible()
    expect(patient.locator("strong").filter(has_text="car park B")).to_be_visible()
    expect(patient.locator(".sj-info ul > li")).to_have_text(["Pay at the machine", "Keep your ticket"])
    the_map = patient.get_by_role("link", name="the map")
    expect(the_map).to_have_attribute("href", "https://example.org/map")
    # Headings are the author's preview's size (h3 18px, as in the editor), never bigger than the set's title (F1).
    parking = patient.get_by_role("heading", name="Parking")
    assert px(parking) == 18 and px(parking) <= px(patient.get_by_role("heading", name="Before you come in")), f"Parking is {px(parking)}px"
    expect(the_map).to_have_attribute("target", "_blank")
    expect(the_map).to_have_attribute("rel", "noopener noreferrer")
    expect(patient.get_by_role("link", name="call us")).to_have_attribute("href", "tel:+10000000000")

    # The picture loads and fits the phone's screen; nothing on the page is wider than it.
    picture = patient.get_by_role("img", name="The car park entrance")
    expect(picture).to_be_visible()
    expect(picture).to_have_attribute("referrerpolicy", "no-referrer")
    patient.wait_for_function("document.querySelector('img[alt=\"The car park entrance\"]').naturalWidth > 0")
    box = picture.bounding_box()
    width = patient.viewport_size["width"]
    assert box and box["x"] >= 0 and box["x"] + box["width"] <= width, f"the picture {box} is wider than the screen ({width})"
    assert patient.evaluate("document.documentElement.scrollWidth") <= width, "the page scrolls sideways"

    # The old statement as before; the tampered one cleaned: its words only, nothing that runs.
    expect(patient.get_by_text("Please arrive 30 minutes early.")).to_be_visible()
    expect(patient.get_by_text("Bring a dressing gown & slippers.")).to_be_visible()
    expect(patient.get_by_text("Imported text")).to_be_visible()
    expect(patient.get_by_text("bad link")).to_be_visible()
    expect(patient.get_by_role("link", name="bad link")).to_have_count(0)
    expect(patient.locator("img[src=x]")).to_have_count(0)
    patient.get_by_text("Click me").click()
    assert patient.evaluate("window.__xss") is None, "something in the HQ ran"

    # An answer shown back in a block ({q_note}) is plain text, never a link or picture (verification 1, F1).
    note = patient.get_by_role("textbox", name="Anything else we should know?")
    note.fill(EVIL_ANSWER)
    note.press("Tab")
    expect(patient.get_by_text(f"You told us: {EVIL_ANSWER}")).to_be_visible()
    expect(patient.get_by_role("link", name="Sign in again")).to_have_count(0)
    expect(patient.locator("img[src*='evil.example']")).to_have_count(0)

    choose(patient, "Will someone drive you home?", "Yes")
    patient.get_by_role("button", name="Finish this section").click()
    patient.get_by_role("button", name="Send my answers").click()
    patient.get_by_role("dialog", name="Send your answers?").get_by_role("button", name="Send my answers").click()
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()

    # The clinician sees the same blocks, cleaned the same way.
    serve_test_pictures(page.context)
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    page.get_by_role("row", name="Before you come in").get_by_role("link", name="Validate").click()
    expect(page.get_by_role("heading", name="Parking")).to_be_visible()
    expect(page.get_by_text("Please arrive 30 minutes early.")).to_be_visible()
    expect(page.get_by_role("link", name="bad link")).to_have_count(0)
    assert page.evaluate("window.__xss") is None, "something in the HQ ran"
    # The patient's answer, shown back to the clinician: words only, no working link or picture (F1).
    expect(page.get_by_text(f"You told us: {EVIL_ANSWER}")).to_be_visible()
    expect(page.get_by_role("link", name="Sign in again")).to_have_count(0)
    expect(page.locator("img[src*='evil.example']")).to_have_count(0)
