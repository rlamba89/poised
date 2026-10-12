"""J2 Publish: publishing is blocked by a logic problem; fix it, publish, then "Create new version"."""

from playwright.sync_api import Page, expect

from support import pick, sign_in, unique


def test_publish_is_blocked_until_the_logic_problem_is_fixed(page: Page, as_user):
    # A draft whose "How many a day?" shows only if a question that no longer exists is Yes.
    hq = as_user("Alex Author").draft_hq("logic-problem-hq.json", f"E2E J2 {unique()}")
    hospital = sign_in(page.context, "Alex Author")
    page.goto(f"/h/{hospital}/questionnaires/{hq['id']}")

    # Publish is refused, and says why.
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    expect(dialog.get_by_role("alert", name="1 problem to fix first")).to_contain_text(
        '"How many a day?": it tests something that no longer exists (q_gone)')
    expect(dialog.get_by_role("button", name="Publish")).to_be_disabled()
    dialog.get_by_role("button", name="Cancel").click()

    # The editor's logic-problem list leads to the broken condition.
    page.get_by_role("complementary").get_by_text("About you").click()
    page.get_by_role("button", name="1 logic problem").click()
    page.get_by_role("dialog", name="1 logic problem").get_by_text("How many a day?").click()
    expect(page.get_by_role("tab", name="Logic")).to_have_attribute("aria-selected", "true")
    expect(page.get_by_text("ID q_perday")).to_be_visible()

    # Fix it: show the question when "Do you smoke?" is Yes.
    pick(page, "Question", "Do you smoke?")
    pick(page, "Value", "Yes")
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")
    expect(page.get_by_role("button", name="1 logic problem")).to_be_hidden()

    # Now it publishes, and the version is read-only.
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    expect(dialog).to_contain_text("It becomes read-only")
    dialog.get_by_role("button", name="Publish").click()
    expect(dialog).to_be_hidden()
    header = page.get_by_role("banner")
    expect(header.get_by_text("v1 · published")).to_be_visible()
    expect(header.get_by_text("Read only")).to_be_visible()

    # "Create new version" gives a draft version 2, which can be published again.
    header.get_by_role("button", name="Create new version").click()
    expect(header.get_by_text("v2 · draft")).to_be_visible()
    expect(header.get_by_role("button", name="Publish")).to_be_visible()

    # The fixed condition was carried into version 2.
    page.get_by_role("complementary").get_by_text("About you").click()
    page.get_by_role("complementary").get_by_text("Smoking").click()
    page.get_by_role("main").get_by_text("How many a day?").click()
    page.get_by_role("tab", name="Logic").click()
    expect(page.get_by_role("combobox", name="Question")).to_have_value("Do you smoke?")
    expect(page.get_by_role("combobox", name="Value")).to_have_value("Yes")
