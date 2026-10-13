"""Staff invites, roles and removing access (S09; STF-01, STF-02, STF-04): only a trust's
whole-trust admins see Staff. Invited people appear in the dev sign-in list at once. A person can
hold several memberships in a trust (the whole trust and/or some hospitals), each changed or
removed on its own.

Each run invites made-up people with unique emails; their names start with "Zoe E2E" so they sort
to the end of the sign-in list. The expired invite (7 days) is tested in the API tests, which can
move the clock."""

import re
import secrets

from playwright.sync_api import Browser, Page, expect

from support import pick, sign_in, unique


def fresh_email(who: str) -> str:
    return f"{who}.{secrets.token_hex(4)}@e2e.example"


def open_staff(page: Page, base: str) -> None:
    page.goto(f"{base}/episodes")
    page.get_by_role("banner").get_by_role("link", name="Staff").click()
    expect(page.get_by_role("heading", name="Staff")).to_be_visible()


def test_an_admin_invites_a_clinician_who_sees_only_their_hospital_then_loses_access(page: Page, browser: Browser, base_url):
    name, email = f"Zoe E2E {unique()}", fresh_email("zoe")
    open_staff(page, sign_in(page.context, "Ada Admin"))

    page.get_by_role("button", name="Invite", exact=True).click()
    dialog = page.get_by_role("dialog")
    dialog.get_by_label("Email").fill(email)
    dialog.get_by_label("Name").fill(name)
    pick(page, "Access", "Hospital A1")
    pick(page, "Role", "Clinician")
    dialog.get_by_role("button", name="Send invite").click()
    expect(page.get_by_role("status")).to_contain_text(f"Invitation sent to {email}")
    row = page.get_by_role("row", name=name)
    expect(row).to_contain_text("Hospital A1")
    expect(row).to_contain_text("Clinician")
    expect(row).to_contain_text("Invited · expires")

    # The invitee, in their own browser: they're in the sign-in list, and see only Hospital A1.
    theirs = browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
    zoe = theirs.new_page()
    try:
        zoe.goto("/login")
        zoe.get_by_role("button", name=name).click()
        expect(zoe).to_have_url(re.compile(r"/o/[0-9a-f-]{36}/h/[0-9a-f-]{36}/episodes$"))
        header = zoe.get_by_role("banner")
        expect(header.get_by_label("Trust")).to_have_text("Trust A")
        expect(header.get_by_role("combobox", name="Hospital")).to_have_value("Hospital A1")
        expect(header.get_by_label("Your role here")).to_have_text("Clinician")
        expect(header.get_by_role("link", name="Staff")).to_have_count(0)
        header.get_by_role("combobox", name="Hospital").click()
        expect(zoe.get_by_role("option")).to_have_count(1)  # only A1
        zoe.keyboard.press("Escape")

        # Ada sees the invite accepted, then removes her.
        page.reload()
        expect(page.get_by_role("row", name=name)).to_contain_text("Active")
        page.get_by_role("button", name=f"Remove {name} from Trust A").click()
        page.get_by_role("dialog").get_by_role("button", name="Remove").click()
        expect(page.get_by_role("status")).to_contain_text(f"{name} no longer has access to Trust A")
        expect(page.get_by_role("row", name=name)).to_have_count(0)

        # Her next action is refused (STF-04), and a reload says she has no access.
        header.get_by_role("link", name="Questionnaires").click()
        expect(zoe.get_by_text("Hospital not found.")).to_be_visible()
        zoe.reload()
        expect(zoe.get_by_text("You don't have access to this hospital")).to_be_visible()
    finally:
        theirs.close()


def test_changing_a_role_changes_what_they_can_do(page: Page, browser: Browser, base_url, as_user):
    name, email = f"Zoe E2E {unique()}", fresh_email("zoe")
    ada = as_user("Ada Admin")
    a1 = next(h for h in ada.call("GET", "/me")["trusts"][0]["hospitals"] if h["name"] == "Hospital A1")
    zoe_id = ada.invite(email, name, a1["id"], "clinician")["userId"]

    theirs = browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
    zoe = theirs.new_page()
    try:
        zoe_base = sign_in(theirs, name)
        zoe.goto(f"{zoe_base}/questionnaires")
        expect(zoe.get_by_role("heading", name="Questionnaires")).to_be_visible()
        expect(zoe.get_by_role("button", name="New Questionnaire")).to_have_count(0)

        open_staff(page, sign_in(page.context, "Ada Admin"))
        page.get_by_role("button", name=f"Change {name}'s access at Hospital A1").click()
        pick(page, "Access", "Whole trust")
        pick(page, "Role", "Super clinician")
        page.get_by_role("dialog").get_by_role("button", name="Save").click()
        expect(page.get_by_role("status")).to_contain_text(f"{name} is now Super clinician, whole trust")

        zoe.reload()
        expect(zoe.get_by_role("button", name="New Questionnaire")).to_be_visible()
        expect(zoe.get_by_role("banner").get_by_label("Your role here")).to_have_text("Super clinician")
    finally:
        theirs.close()
        ada.call("DELETE", ada.o(f"/staff/{zoe_id}"))  # so runs don't leave super clinicians in Trust A


def test_an_existing_account_only_gets_a_membership(page: Page, as_user):
    """STF-02: Max (Trust B's admin) invites someone new to Trust B; Ada then invites the same email
    to Trust A, which adds a membership to the same account."""
    name, email = f"Zoe E2E {unique()}", fresh_email("shared")
    max_ = as_user("Max Multi", "Hospital B1")
    zoe_id = max_.invite(email, name, None, "clinician")["userId"]

    open_staff(page, sign_in(page.context, "Ada Admin"))
    page.get_by_role("button", name="Invite", exact=True).click()
    page.get_by_role("dialog").get_by_label("Email").fill(email.upper())
    pick(page, "Access", "Hospital A2")
    page.get_by_role("dialog").get_by_role("button", name="Send invite").click()
    expect(page.get_by_role("status")).to_contain_text(f"{name} already had an account")
    expect(page.get_by_role("row", name=name)).to_contain_text("Hospital A2")

    page.goto("/login")
    expect(page.get_by_role("button", name=name)).to_have_count(1)  # one account, not two

    ada = as_user("Ada Admin")  # so runs don't leave people in the trusts
    ada.call("DELETE", ada.o(f"/staff/{zoe_id}"))
    max_.call("DELETE", max_.o(f"/staff/{zoe_id}"))


def test_only_whole_trust_admins_see_staff(page: Page):
    for who in ["Cara Clinician", "Alex Author"]:  # a clinician; a whole-trust super clinician
        base = sign_in(page.context, who)
        page.goto(f"{base}/episodes")
        expect(page.get_by_role("heading", name="Episodes")).to_be_visible()
        expect(page.get_by_role("banner").get_by_role("link", name="Staff")).to_have_count(0)
        page.goto(f"{base}/staff")
        expect(page.get_by_text("Only the trust's admins can manage staff")).to_be_visible()


def test_an_invalid_email_is_refused(page: Page):
    open_staff(page, sign_in(page.context, "Ada Admin"))
    page.get_by_role("button", name="Invite", exact=True).click()
    dialog = page.get_by_role("dialog")
    dialog.get_by_label("Email").fill("not-an-email")
    dialog.get_by_role("button", name="Send invite").click()
    expect(dialog.get_by_text("Enter a valid email address")).to_be_visible()


def a1_id(ada) -> str:
    return next(h for t in ada.call("GET", "/me")["trusts"] for h in t["hospitals"] if h["name"] == "Hospital A1")["id"]


def test_someone_can_work_at_several_hospitals_of_a_trust(page: Page, browser: Browser, base_url, as_user):
    """STF-02 (Rahul, 13 Oct): a role at each of several hospitals, each changed or removed alone."""
    name, email = f"Zoe E2E {unique()}", fresh_email("multi")
    ada = as_user("Ada Admin")
    zoe_id = ada.invite(email, name, a1_id(ada), "clinician")["userId"]
    try:
        open_staff(page, sign_in(page.context, "Ada Admin"))
        page.get_by_role("button", name=f"Give {name} more access").click()
        dialog = page.get_by_role("dialog")
        expect(dialog).to_contain_text(email)
        pick(page, "Access", "Hospital A2")
        pick(page, "Role", "Super clinician")
        dialog.get_by_role("button", name="Add access").click()
        expect(page.get_by_role("status")).to_contain_text(f"{name} now also has access to Hospital A2 as Super clinician")
        row = page.get_by_role("row", name=name)
        expect(row).to_contain_text("Hospital A1 · Clinician")
        expect(row).to_contain_text("Hospital A2 · Super clinician")

        # Remove only the A1 access: A2 stays.
        page.get_by_role("button", name=f"Remove {name}'s access at Hospital A1").click()
        page.get_by_role("dialog").get_by_role("button", name="Remove").click()
        expect(page.get_by_role("status")).to_contain_text(f"{name} no longer has access to Hospital A1")
        expect(row).not_to_contain_text("Hospital A1")
        expect(row).to_contain_text("Hospital A2 · Super clinician")

        # Signing in, they go straight into their one hospital, A2.
        theirs = browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
        try:
            zoe = theirs.new_page()
            zoe.goto("/login")
            zoe.get_by_role("button", name=name).click()
            expect(zoe.get_by_role("banner").get_by_role("combobox", name="Hospital")).to_have_value("Hospital A2")
        finally:
            theirs.close()
    finally:
        ada.call("DELETE", ada.o(f"/staff/{zoe_id}"))


def test_the_invite_dialog_starts_afresh_each_time(page: Page, as_user):
    """S09 test report 1, F-1: it kept the last email, access and role."""
    open_staff(page, sign_in(page.context, "Ada Admin"))
    dialog = page.get_by_role("dialog")

    def opened_fresh():
        page.get_by_role("button", name="Invite", exact=True).click()
        expect(dialog.get_by_label("Email")).to_have_value("")
        expect(dialog.get_by_label("Name")).to_have_value("")
        expect(dialog.get_by_role("combobox", name="Access")).to_have_value("Hospital A1")
        expect(dialog.get_by_role("combobox", name="Role")).to_have_value("Clinician")

    opened_fresh()
    dialog.get_by_label("Email").fill("cancelled@e2e.example")
    pick(page, "Access", "Whole trust")
    pick(page, "Role", "Admin")
    dialog.get_by_role("button", name="Cancel").click()
    expect(dialog).to_have_count(0)

    opened_fresh()  # after Cancel
    name, email = f"Zoe E2E {unique()}", fresh_email("fresh")
    dialog.get_by_label("Email").fill(email)
    dialog.get_by_label("Name").fill(name)
    pick(page, "Access", "Whole trust")
    pick(page, "Role", "Super clinician")
    dialog.get_by_role("button", name="Send invite").click()
    expect(page.get_by_role("status")).to_contain_text(f"Invitation sent to {email}")

    opened_fresh()  # after an invite
    dialog.get_by_role("button", name="Cancel").click()
    ada = as_user("Ada Admin")
    zoe = next(r for r in ada.call("GET", ada.o("/staff")) if r["email"] == email)
    ada.call("DELETE", ada.o(f"/staff/{zoe['userId']}"))
