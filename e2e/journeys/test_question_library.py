"""The question library and "Same as…" (S24): inserting a library question keeps its key, and a
second insert into the same HQ is refused; Save to library and Remove for the trust's own questions
(never seen by another trust); "Same as…" gives a new question a published key and its options,
keeping its own text, with incompatible candidates greyed with their reason; and S23's lock then
refuses removing one of the adopted options at Publish."""

import re

from playwright.sync_api import Page, expect

from support import sign_in, unique


def open_page(page: Page, base: str, hq: dict, set_name: str, page_title: str) -> None:
    page.goto(f"{base}/questionnaires/{hq['id']}")
    structure = page.get_by_role("complementary")
    structure.get_by_text(set_name, exact=True).click()
    structure.get_by_text(page_title, exact=True).click()


def saved(page: Page) -> None:
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")


def add_question(page: Page, kind: str, title: str) -> str:
    """Adds a question of a kind at the end of the page, gives it its text, and returns its key."""
    page.get_by_role("main").get_by_role("button", name=kind, exact=True).last.click()
    settings = page.get_by_role("tabpanel", name="Settings")
    text = settings.get_by_label("Question text")
    text.fill(title)
    text.press("Tab")
    saved(page)
    return page.get_by_role("complementary").get_by_text(re.compile(r"^ID q_")).inner_text().removeprefix("ID ")


def test_library_and_same_as(page: Page, as_user):
    alex = as_user("Alex Author")
    hq = alex.draft_hq("workflow-hq.json", f"E2E Library {unique()}")
    base = sign_in(page.context, "Alex Author")
    open_page(page, base, hq, "About you", "Smoking")
    main = page.get_by_role("main")

    # Insert "Do you smoke?" from the platform's library: it keeps its key, smoker.
    main.get_by_role("button", name="From library").last.click()
    library = page.get_by_role("dialog", name="Insert from the question library")
    library.get_by_label("Search").fill("smoke")
    row = library.get_by_test_id("library-smoker")
    expect(row).to_contain_text("Preopsis")
    row.get_by_role("button", name="Insert").click()
    expect(library).to_be_hidden()
    expect(page.get_by_test_id("card-smoker")).to_contain_text("No, I have never smoked")
    expect(page.get_by_role("complementary").get_by_text("ID smoker")).to_be_visible()
    saved(page)

    # Inserting it again into this HQ is refused.
    main.get_by_role("button", name="From library").last.click()
    library.get_by_label("Search").fill("smoke")
    library.get_by_test_id("library-smoker").get_by_role("button", name="Insert").click()
    expect(library.get_by_test_id("library-smoker")).to_contain_text("This HQ already has a question with the ID smoker")
    page.keyboard.press("Escape")
    expect(library).to_be_hidden()
    expect(page.get_by_test_id("card-smoker")).to_have_count(1)

    # Save a question of the trust's own to the library, with a tag; Trust B's authors never see it.
    title = f"E2E library question {unique()}"
    key = add_question(page, "Text", title)
    page.get_by_role("complementary").get_by_role("button", name="Save to library").click()
    save = page.get_by_role("dialog", name="Save to the question library")
    save.get_by_label("Tags (optional)").fill("E2E")
    save.get_by_label("Tags (optional)").press("Enter")
    save.get_by_role("button", name="Save to library").click()
    expect(save.get_by_text("Saved.")).to_be_visible()
    save.get_by_role("button", name="Done").click()
    assert alex.call("GET", alex.o(f"/library-questions?q={title}"))[0]["key"] == key
    bea = as_user("Bea Author", hospital="Hospital B1")
    assert bea.call("GET", bea.o(f"/library-questions?q={title}")) == []

    # It's in the library as the trust's own; Remove takes it out (the HQ keeps its question).
    main.get_by_role("button", name="From library").last.click()
    library.get_by_label("Search").fill(title)
    mine = library.get_by_test_id(f"library-{key}")
    expect(mine).to_contain_text("This trust")
    expect(mine).to_contain_text("E2E")
    mine.get_by_role("button", name="Remove").click()
    mine.get_by_role("alert").get_by_role("button", name="Remove").click()
    expect(library.get_by_text("No library questions match.")).to_be_visible()
    page.keyboard.press("Escape")
    expect(page.get_by_test_id(f"card-{key}")).to_be_visible()

    # Each opening starts with an empty search (test report 1, found 2). A repeating list from the
    # library: its row questions can't be saved to the library on their own (found 1).
    main.get_by_role("button", name="From library").last.click()
    expect(library.get_by_label("Search")).to_have_value("")
    library.get_by_label("Search").fill("allergies")
    library.get_by_test_id("library-allergies").get_by_role("button", name="Insert").click()
    saved(page)
    page.get_by_test_id("card-allergies_what").click()
    expect(page.get_by_role("complementary").get_by_text("ID allergies_what")).to_be_visible()
    expect(page.get_by_role("complementary").get_by_role("button", name="Save to library")).to_have_count(0)
    page.get_by_test_id("card-allergies").get_by_text(re.compile(r"^Repeats")).first.click()
    expect(page.get_by_role("complementary").get_by_text("ID allergies", exact=True)).to_be_visible()
    expect(page.get_by_role("complementary").get_by_role("button", name="Save to library")).to_be_visible()

    # "Same as…": a new choice question takes the published key "pregnant" (the seeded Demo pre-op
    # HQ asks it) and its options, and keeps its own text. "pregnant" is Always ask fresh, so the new
    # question is too (Rahul, 23 Oct); switched off, it pre-fills from that HQ, by link.
    mine_title = "Could you be pregnant at the moment?"
    add_question(page, "Select One", mine_title)
    page.get_by_role("complementary").get_by_role("button", name="Same as…").click()
    same = page.get_by_role("dialog", name="Same as…")
    same.get_by_label("Search").fill("pregnant")
    candidate = same.get_by_test_id("candidate-pregnant")
    expect(candidate).to_contain_text("In Demo pre-op HQ")
    expect(candidate.get_by_test_id("candidate-fresh")).to_have_text("Always ask fresh, like the original")
    candidate.get_by_role("button", name="Use this").click()
    expect(same).to_be_hidden()
    card = page.get_by_test_id("card-pregnant")
    expect(card).to_contain_text("I'm not sure")
    settings = page.get_by_role("complementary")
    expect(settings.get_by_label("Question text")).to_have_value(mine_title)
    expect(settings.get_by_text("ID pregnant")).to_be_visible()
    expect(settings.get_by_test_id("linked-from")).to_contain_text("Linked to “Is there any chance you could be pregnant?”")
    saved(page)
    expect(card.get_by_test_id("always-fresh")).to_be_visible()
    settings.get_by_role("switch", name="Always ask fresh").uncheck()
    saved(page)
    label = card.get_by_test_id("prefill-label")
    expect(label).to_contain_text(re.compile(r"Pre-fills from \d+ other HQs?"))
    label.click()
    expect(page.get_by_test_id("prefill-sources").get_by_role("listitem").filter(has_text="Demo pre-op HQ")).to_contain_text("by link")
    page.keyboard.press("Escape")

    # An incompatible candidate is greyed, with its reason, and can't be used.
    add_question(page, "Text", "How loud is your snoring?")
    page.get_by_role("complementary").get_by_role("button", name="Same as…").click()
    same.get_by_label("Search").fill("snore")
    snoring = same.get_by_test_id("candidate-snoring")
    expect(snoring.get_by_test_id("candidate-reason")).to_have_text("Its answer is single choice; this question's is text.")
    expect(snoring.get_by_role("button", name="Use this")).to_be_disabled()
    page.keyboard.press("Escape")
    expect(same).to_be_hidden()
    page.get_by_role("complementary").get_by_role("button", name="Same as…").click()
    expect(same.get_by_label("Search")).to_have_value("")
    expect(same.get_by_test_id("candidate-snoring")).to_be_visible()
    page.keyboard.press("Escape")
    expect(same).to_be_hidden()

    # Deleting one of the adopted options is refused at Publish: the trust published "pregnant" with it.
    page.get_by_test_id("card-pregnant").click()
    settings.get_by_role("textbox", name="Option 3").locator("xpath=ancestor::*[.//button[@aria-label='Delete option']][1]").get_by_role(
        "button", name="Delete option").click()
    saved(page)
    expect(page.get_by_test_id("card-pregnant")).not_to_contain_text("I'm not sure")
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    expect(dialog.get_by_test_id("publish-problems")).to_contain_text(
        f"“{mine_title}”: the option “I'm not sure” was removed. A published question's answers can't change")
    expect(dialog.get_by_role("button", name="Publish")).to_be_disabled()


def test_same_as_and_a_later_question_sets_condition(page: Page, as_user):
    """S24 verification 2: a later Question Set shown only for an answer the new options don't have.
    Same as… warns first; after "Use this anyway" the Publish window refuses the set's condition, so
    the set can't be hidden from patients without anyone knowing."""
    alex = as_user("Alex Author")
    hq = alex.draft_hq("workflow-hq.json", f"E2E Same as sets {unique()}")
    # "Your health" is shown only to non-smokers: {q_smokes} = 'o_no' ("No").
    health = hq["chapters"]["Your health"]
    chapter = alex.call("GET", alex.h(f"/chapters/{health}"))
    content = {**chapter["content"], "chapterVisibleIf": "{q_smokes} = 'o_no'"}
    alex.call("PUT", alex.h(f"/chapters/{health}/content"), {"content": content, "revision": chapter["revision"]})

    base = sign_in(page.context, "Alex Author")
    open_page(page, base, hq, "About you", "Smoking")
    page.get_by_test_id("card-q_smokes").click()
    page.get_by_role("complementary").get_by_role("button", name="Same as…").click()
    same = page.get_by_role("dialog", name="Same as…")
    same.get_by_label("Search").fill("do you smoke")
    smoker = same.get_by_test_id("candidate-smoker")
    smoker.get_by_role("button", name="Use this").click()
    # "No" has no option with the same text among smoker's, and its options had disclosures.
    losses = smoker.get_by_test_id("candidate-losses")
    expect(losses).to_contain_text("Its own disclosures on 2 options will be removed.")
    expect(losses).to_contain_text("1 condition tests its old answers")
    losses.get_by_role("button", name="Use this anyway").click()
    expect(same).to_be_hidden()
    saved(page)

    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    expect(dialog.get_by_test_id("publish-problems")).to_contain_text(
        "Question Set condition: it tests an answer that no longer exists (“Do you smoke?”: o_no)")
    expect(dialog.get_by_role("button", name="Publish")).to_be_disabled()
