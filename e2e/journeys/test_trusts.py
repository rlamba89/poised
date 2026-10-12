"""Trusts and hospitals (S08): the hospital picker, switching hospitals and trusts from the header,
"All hospitals" for trust-wide staff, and a hospital-scoped clinician kept to their hospital."""

import re

from playwright.sync_api import Page, expect

from support import choose, pick, sign_in, unique


def test_a_member_of_two_trusts_picks_and_switches_hospitals(page: Page):
    # Max is a clinician at Hospital A2 (Trust A) and the admin of all of Trust B.
    page.goto("/login")
    page.get_by_role("button", name="Max Multi").click()
    expect(page.get_by_role("heading", name="Choose a hospital")).to_be_visible()
    trust_a = page.get_by_role("region", name="Trust A")
    trust_b = page.get_by_role("region", name="Trust B")
    expect(trust_a.get_by_role("link")).to_have_count(1)  # only A2, not A1
    expect(trust_a.get_by_role("link", name="Hospital A2")).to_contain_text("Clinician")
    expect(trust_b).to_contain_text("Admin, whole trust")

    # Into Trust B: an admin starts on Questionnaires.
    trust_b.get_by_role("link", name="Hospital B1").click()
    expect(page.get_by_role("heading", name="Questionnaires")).to_be_visible()
    header = page.get_by_role("banner")
    expect(header.get_by_label("Trust")).to_have_text("Trust B")
    expect(header.get_by_label("Your role here")).to_have_text("Admin")

    # On Episodes, switch to Trust A's hospital from the header: it stays on Episodes, with no
    # "All hospitals", because Max has only A2 in that trust.
    header.get_by_role("link", name="Episodes").click()
    expect(page).to_have_url(re.compile(r"/episodes$"))
    pick(page, "Hospital", "Hospital A2")
    expect(page).to_have_url(re.compile(r"/o/[0-9a-f-]{36}/h/[0-9a-f-]{36}/episodes$"))
    expect(page.get_by_role("heading", name="Episodes")).to_be_visible()
    expect(header.get_by_label("Trust")).to_have_text("Trust A")
    expect(header.get_by_label("Your role here")).to_have_text("Clinician")
    expect(page.get_by_text("All hospitals")).to_have_count(0)


def test_only_whole_trust_staff_change_the_trusts_questionnaires(page: Page, as_user):
    """S08 answers, 5: Sue, a super clinician of Hospital A1 only, sees Trust A's questionnaires
    but can't change them; Alex, a super clinician of the whole trust, can."""
    hq = as_user("Alex Author").draft_hq("workflow-hq.json", f"E2E trust-wide {unique()}")
    base = sign_in(page.context, "Sue Super")
    page.goto(f"{base}/questionnaires")
    expect(page.get_by_role("heading", name="Questionnaires")).to_be_visible()
    expect(page.get_by_role("button", name="New Questionnaire")).to_have_count(0)
    page.get_by_role("button", name=f"{hq['name']} options").click()
    expect(page.get_by_role("menuitem", name="View")).to_be_visible()
    expect(page.get_by_role("menuitem", name="Delete")).to_have_count(0)
    page.goto(f"{base}/questionnaires/{hq['id']}")
    header = page.get_by_role("banner")
    expect(header.get_by_text("v1 · draft")).to_be_visible()
    expect(header.get_by_text("Read only")).to_be_visible()
    expect(header.get_by_role("button", name="Publish")).to_have_count(0)

    # The Preview lets her try the answers and run test cases, but not save or delete them
    # (S08 test report 2, F-1).
    cid = hq["chapters"]["About you"]
    page.goto(f"{base}/chapters/{cid}/preview")
    cases = page.get_by_test_id("test-cases")
    expect(cases).to_contain_text("Read only: only super clinicians and admins of the whole trust can change test cases.")
    expect(cases.get_by_role("button", name="Save answers")).to_have_count(0)
    expect(cases.get_by_role("textbox", name="Test case name")).to_have_count(0)
    expect(page.get_by_role("link", name="Open in editor")).to_be_visible()


def test_the_staff_header_fits_a_narrow_window(page: Page):
    """S08 test report 2: below about 768px the header pushed Episodes and Sign out off screen."""
    page.set_viewport_size({"width": 390, "height": 844})
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes")
    header = page.get_by_role("banner")
    for name in ("Questionnaires", "Episodes"):
        header.get_by_role("link", name=name).scroll_into_view_if_needed()
        expect(header.get_by_role("link", name=name)).to_be_in_viewport()
    header.get_by_role("button", name="Sign out").scroll_into_view_if_needed()
    expect(header.get_by_role("button", name="Sign out")).to_be_in_viewport()
    # The header scrolls inside itself; it never widens the page.
    assert header.bounding_box()["width"] <= 390, "the header is wider than the window"


def test_trust_wide_staff_see_all_hospitals_and_others_only_their_own(page: Page, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E trusts {unique()}")
    at_a1 = as_user("Cara Clinician").episode(hq, f"A1 {unique()}")
    at_a2 = as_user("Val Clinician", "Hospital A2").episode(hq, f"A2 {unique()}")

    # Ada, Trust A's admin, sees one hospital's episodes, then all of them.
    base = sign_in(page.context, "Ada Admin")
    page.goto(f"{base}/episodes")
    expect(page.get_by_role("row", name=at_a1["name"])).to_be_visible()
    expect(page.get_by_role("row", name=at_a2["name"])).to_have_count(0)
    page.get_by_text("All hospitals").click()
    expect(page).to_have_url(re.compile(r"\?hospitals=all$"))
    expect(page.get_by_role("columnheader", name="Hospital")).to_be_visible()
    expect(page.get_by_role("row", name=at_a1["name"])).to_contain_text("Hospital A1")
    expect(page.get_by_role("row", name=at_a2["name"])).to_contain_text("Hospital A2")

    # The choice survives a reload, and an episode opens at its own hospital.
    page.reload()
    page.get_by_role("row", name=at_a2["name"]).click()
    expect(page.get_by_role("heading", name=at_a2["name"])).to_be_visible()
    expect(page.get_by_role("banner").get_by_role("combobox", name="Hospital")).to_have_value("Hospital A2")

    # Cara, a clinician at A1 only, can't open A2's episode by its URL.
    a2_episode_url = page.url
    page.context.clear_cookies()
    sign_in(page.context, "Cara Clinician")
    page.goto(a2_episode_url)
    expect(page.get_by_text("You don't have access to this hospital")).to_be_visible()
    expect(page.get_by_text(at_a2["name"])).to_have_count(0)


def test_a_whole_trust_super_clinician_saves_a_test_case_on_a_note_only_output(page: Page, as_user):
    """S08 test report 3: Save answers did nothing when an output had no codes or note category,
    as in this fixture ({id, note: {text}})."""
    hq = as_user("Alex Author").draft_hq("workflow-hq.json", f"E2E test case {unique()}")
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/chapters/{hq['chapters']['About you']}/preview")
    choose(page, "Do you smoke?", "Yes")
    expect(page.get_by_test_id("outputs-panel")).to_contain_text("Smoker")
    cases = page.get_by_test_id("test-cases")
    cases.get_by_role("textbox", name="Test case name").fill("Smoker")
    cases.get_by_role("button", name="Save answers").click()
    expect(cases.get_by_text("Smoker", exact=True)).to_be_visible()
    cases.get_by_role("button", name="Run all").click()
    expect(cases).to_contain_text("All 1 passed")
