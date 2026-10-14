"""J6, onboarding a trust (S10; ORG-01…05, A-21): the platform admin creates a trust and a hospital
and invites the trust's first admin; that admin invites a clinician; the clinician signs in. Also:
suspending a trust stops its staff signing in; only platform admins reach /admin; a trust admin
gives a hospital settings of its own.

Each run makes a new trust with a unique code, and its people's names start with "Zoe E2E" so they
sort to the end of the sign-in list. Trusts can't be deleted, so the platform admin's list grows;
reset the local database when it gets long."""

import re
import secrets

import pytest
from playwright.sync_api import Browser, Page, expect

from support import pick, sign_in, unique


# Signs in as people it has just invited: the dev login only (local). On QA their temporary
# password is in Cognito's email, and Cognito sends about 50 a day (S12; conftest.py).
pytestmark = pytest.mark.dev_login


def fresh_email(who: str) -> str:
    return f"{who}.{secrets.token_hex(4)}@e2e.example"


def fresh_code() -> str:
    return f"E2E-{secrets.token_hex(3).upper()}"


def sign_in_as(page: Page, name: str) -> None:
    """Signs in from the dev sign-in page, as a person would."""
    page.goto("/login")
    page.get_by_role("button", name=name).click()


def test_j6_the_platform_admin_onboards_a_trust_whose_admin_invites_a_clinician(page: Page, browser: Browser, base_url):
    trust, code = f"E2E Trust {unique()}", fresh_code()
    admin, admin_email = f"Zoe E2E Admin {unique()}", fresh_email("admin")
    hospital = f"E2E General {unique()}"
    clinician, clinician_email = f"Zoe E2E Clinician {unique()}", fresh_email("clinician")

    # The platform admin signs in and lands on the console.
    sign_in_as(page, "Sam Superadmin")
    expect(page).to_have_url(re.compile(r"/admin$"))
    expect(page.get_by_role("heading", name="Trusts")).to_be_visible()

    page.get_by_role("link", name="New trust").click()
    page.get_by_label("Trust name").fill(trust)
    page.get_by_label("Code").fill(code.lower())  # kept in capitals
    page.get_by_label("Contact name").fill("Pat Contact")
    page.get_by_label("Contact email").fill("pat.contact@e2e.example")
    page.get_by_label("Admin's email").fill(admin_email)
    page.get_by_label("Admin's name").fill(admin)
    page.get_by_label("PROMs").check()
    page.get_by_role("button", name="Create trust").click()

    expect(page.get_by_role("status")).to_contain_text(f"{trust} was created. An invitation was sent to {admin_email}")
    expect(page.get_by_role("heading", name=trust)).to_be_visible()
    expect(page.get_by_text(f"Code {code}")).to_be_visible()
    expect(page.get_by_label(f"Admin {admin}")).to_contain_text("Invited")
    expect(page.get_by_label("PROMs: trust default")).to_be_checked()

    page.get_by_role("button", name="Add hospital").click()
    dialog = page.get_by_role("dialog")
    dialog.get_by_label("Hospital name").fill(hospital)
    dialog.get_by_role("textbox", name="Address", exact=True).fill("1 E2E Road\nTestville")
    dialog.get_by_label("Phone for patients").fill("0100 000 0000")
    dialog.get_by_role("button", name="Add hospital").click()
    expect(page.get_by_role("status")).to_contain_text(f"{hospital} was added to {trust}")
    row = page.get_by_role("row", name=hospital, exact=True)
    expect(row).to_contain_text("0100 000 0000")
    expect(row).to_contain_text("(default)")  # the deployment's time zone

    # The new admin, in their own browser: straight into the hospital, with Staff and Settings.
    theirs = browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
    try:
        ann = theirs.new_page()
        sign_in_as(ann, admin)
        expect(ann).to_have_url(re.compile(r"/o/[0-9a-f-]{36}/h/[0-9a-f-]{36}/questionnaires$"))
        header = ann.get_by_role("banner")
        expect(header.get_by_label("Trust")).to_have_text(trust)
        expect(header.get_by_role("combobox", name="Hospital")).to_have_value(hospital)
        expect(header.get_by_label("Your role here")).to_have_text("Admin")
        expect(header.get_by_role("link", name="Settings")).to_be_visible()
        expect(header.get_by_role("link", name="Platform admin")).to_have_count(0)

        header.get_by_role("link", name="Staff").click()
        ann.get_by_role("button", name="Invite", exact=True).click()
        invite = ann.get_by_role("dialog")
        invite.get_by_label("Email").fill(clinician_email)
        invite.get_by_label("Name").fill(clinician)
        pick(ann, "Access", hospital)
        pick(ann, "Role", "Clinician")
        invite.get_by_role("button", name="Send invite").click()
        expect(ann.get_by_role("status")).to_contain_text(f"Invitation sent to {clinician_email}")

        # The clinician signs in and works at the hospital.
        cara = theirs.browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
        try:
            them = cara.new_page()
            sign_in_as(them, clinician)
            expect(them).to_have_url(re.compile(r"/episodes$"))
            expect(them.get_by_role("heading", name="Episodes")).to_be_visible()
            expect(them.get_by_role("banner").get_by_role("combobox", name="Hospital")).to_have_value(hospital)
        finally:
            cara.close()
    finally:
        theirs.close()

    # The platform admin sees the first admin has signed in.
    page.reload()
    expect(page.get_by_label(f"Admin {admin}")).to_contain_text("Active")


def test_a_suspended_trusts_staff_cant_sign_in_until_it_is_re_activated(page: Page, browser: Browser, base_url, as_user):
    sam = as_user("Sam Superadmin", None)
    admin = f"Zoe E2E Admin {unique()}"
    t = sam.new_trust(f"E2E Trust {unique()}", fresh_code(), fresh_email("admin"), admin)
    sam.new_hospital(t["id"], f"E2E General {unique()}")

    sign_in_as(page, "Sam Superadmin")
    page.goto(f"/admin/trusts/{t['id']}")
    page.get_by_role("button", name="Suspend trust").click()
    page.get_by_role("dialog").get_by_role("button", name="Suspend trust").click()
    expect(page.get_by_role("status")).to_contain_text("is suspended")
    expect(page.get_by_text("This trust is suspended")).to_be_visible()

    theirs = browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
    try:
        them = theirs.new_page()
        sign_in_as(them, admin)
        expect(them.get_by_text("Your trust or hospital is suspended")).to_be_visible()
        expect(them).to_have_url(re.compile(r"/login$"))

        page.get_by_role("button", name="Re-activate trust").click()
        expect(page.get_by_role("status")).to_contain_text("is active again")

        them.get_by_role("button", name=admin).click()
        expect(them).to_have_url(re.compile(r"/questionnaires$"))
    finally:
        theirs.close()

    # The list shows its status.
    page.get_by_role("banner").get_by_role("link", name="Trusts").click()
    expect(page.get_by_role("row", name=t["name"])).to_contain_text("Active")


def test_only_platform_admins_reach_the_admin_console(page: Page):
    base = sign_in(page.context, "Ada Admin")
    page.goto(f"{base}/episodes")
    expect(page.get_by_role("heading", name="Episodes")).to_be_visible()
    expect(page.get_by_role("banner").get_by_role("link", name="Platform admin")).to_have_count(0)
    page.goto("/admin")
    expect(page.get_by_text("Only Poised's platform admins can open this page")).to_be_visible()
    expect(page.get_by_role("link", name="New trust")).to_have_count(0)


def test_a_duplicate_trust_code_is_refused(page: Page, as_user):
    sam = as_user("Sam Superadmin", None)
    code = fresh_code()
    sam.new_trust(f"E2E Trust {unique()}", code, fresh_email("admin"), f"Zoe E2E Admin {unique()}")

    sign_in_as(page, "Sam Superadmin")
    page.get_by_role("link", name="New trust").click()
    page.get_by_label("Trust name").fill(f"E2E Trust {unique()}")
    page.get_by_label("Code").fill(code.lower())
    page.get_by_label("Contact name").fill("Pat Contact")
    page.get_by_label("Contact email").fill("pat.contact@e2e.example")
    page.get_by_label("Admin's email").fill(fresh_email("admin"))
    page.get_by_role("button", name="Create trust").click()
    expect(page.get_by_text(f"Another trust already has the code {code}")).to_be_visible()
    expect(page).to_have_url(re.compile(r"/admin/trusts/new$"))


def test_a_trust_admin_gives_one_hospital_its_own_settings(page: Page, as_user):
    sam = as_user("Sam Superadmin", None)
    admin = f"Zoe E2E Admin {unique()}"
    t = sam.new_trust(f"E2E Trust {unique()}", fresh_code(), fresh_email("admin"), admin)
    north, south = f"E2E North {unique()}", f"E2E South {unique()}"
    sam.new_hospital(t["id"], north)
    sam.new_hospital(t["id"], south)

    sign_in_as(page, admin)
    page.get_by_role("link", name=north).click()  # the hospital picker: two hospitals
    page.get_by_role("banner").get_by_role("link", name="Settings").click()
    expect(page.get_by_role("heading", name="Settings")).to_be_visible()

    # The trust's phone, which both hospitals follow.
    page.get_by_role("checkbox", name="Use Poised's default: Phone number for patients").uncheck()
    page.get_by_role("textbox", name="Phone number for patients", exact=True).fill("0100 111 1111")
    page.get_by_role("button", name="Save the trust's settings").click()
    expect(page.get_by_role("status")).to_contain_text(f"{t['name']}'s settings were saved")
    # Next to the Save button, not 2,000px above it (S10 test report 1, F2).
    expect(page.get_by_role("status")).to_be_in_viewport()
    # Saving again with nothing changed replaces the message rather than leaving "saved" behind.
    page.get_by_role("button", name="Save the trust's settings").click()
    expect(page.get_by_role("status")).to_have_text("Nothing has changed.")

    # South gets its own phone; North still follows the trust.
    pick(page, "Settings for", south)
    expect(page.get_by_role("textbox", name="Phone number for patients", exact=True)).to_have_value("0100 111 1111")
    expect(page.get_by_role("textbox", name="Phone number for patients", exact=True)).to_be_disabled()
    page.get_by_role("checkbox", name="Use the trust's setting: Phone number for patients").uncheck()
    page.get_by_role("textbox", name="Phone number for patients", exact=True).fill("0100 222 2222")
    page.get_by_role("button", name=f"Save {south}'s settings").click()
    expect(page.get_by_role("status")).to_contain_text(f"{south}'s settings were saved")
    expect(page.get_by_role("status")).to_be_in_viewport()

    page.reload()
    pick(page, "Settings for", south)
    expect(page.get_by_role("textbox", name="Phone number for patients", exact=True)).to_have_value("0100 222 2222")
    pick(page, "Settings for", north)
    expect(page.get_by_role("textbox", name="Phone number for patients", exact=True)).to_have_value("0100 111 1111")
    expect(page.get_by_role("textbox", name="Phone number for patients", exact=True)).to_be_disabled()
    # The opt-ins and trust-only settings are shown, not edited, at a hospital (F1: the trust-only
    # ones were missing).
    expect(page.get_by_label("PROMs")).to_contain_text("Off")
    expect(page.get_by_label("PROMs")).to_contain_text("Set by Poised's platform admin")
    retention = page.get_by_label("Keep episode data for (years)")
    expect(retention).to_contain_text("Set for the whole trust")
    expect(retention).to_contain_text("8")
    expect(page.get_by_role("spinbutton", name="Keep episode data for (years)")).to_have_count(0)


def test_the_platform_admin_edits_a_trusts_and_a_hospitals_details(page: Page, as_user):
    """S10 answers 3 (Rahul, 13 Oct): the platform admin edits them at any time."""
    sam = as_user("Sam Superadmin", None)
    t = sam.new_trust(f"E2E Trust {unique()}", fresh_code(), fresh_email("admin"), f"Zoe E2E Admin {unique()}")
    hospital, renamed, new_name, new_code = f"E2E General {unique()}", f"E2E Central {unique()}", f"E2E Renamed {unique()}", fresh_code()
    sam.new_hospital(t["id"], hospital)

    sign_in_as(page, "Sam Superadmin")
    page.goto(f"/admin/trusts/{t['id']}")
    page.get_by_role("button", name="Edit details").click()
    dialog = page.get_by_role("dialog")
    expect(dialog.get_by_role("textbox", name="Trust name")).to_have_value(t["name"])
    dialog.get_by_role("textbox", name="Trust name").fill(new_name)
    dialog.get_by_role("textbox", name="Code").fill(new_code.lower())
    dialog.get_by_role("textbox", name="Contact phone").fill("0100 999 9999")
    dialog.get_by_role("button", name="Save").click()
    expect(page.get_by_role("status")).to_contain_text(f"{new_name}'s details were saved")
    expect(page.get_by_role("heading", name=new_name, exact=True)).to_be_visible()
    expect(page.get_by_text(f"Code {new_code}")).to_be_visible()
    expect(page.get_by_text("0100 999 9999")).to_be_visible()

    page.get_by_role("button", name=f"Edit {hospital}").click()
    dialog = page.get_by_role("dialog")
    expect(dialog.get_by_role("textbox", name="Address", exact=True)).to_have_value("1 E2E Road")
    dialog.get_by_role("textbox", name="Hospital name").fill(renamed)
    dialog.get_by_role("textbox", name="Email for patients").fill("central@e2e.example")
    pick(page, "Time zone", "America/Toronto")
    dialog.get_by_role("button", name="Save").click()
    expect(page.get_by_role("status")).to_contain_text(f"{renamed}'s details were saved")
    row = page.get_by_role("row", name=renamed, exact=True)
    expect(row).to_contain_text("central@e2e.example")
    expect(row).to_contain_text("America/Toronto")

    page.reload()
    expect(page.get_by_role("heading", name=new_name, exact=True)).to_be_visible()
    expect(page.get_by_role("row", name=renamed, exact=True)).to_contain_text("America/Toronto")

    # An opt-in's message shows by the opt-ins, not at the top of the page (S10 test report 1, F2).
    page.get_by_label("PROMs: trust default").click()
    expect(page.get_by_role("status")).to_contain_text("PROMs is now on by default")
    expect(page.get_by_role("status")).to_be_in_viewport()

    # A code another trust has is refused in the dialog, which stays open.
    page.get_by_role("button", name="Edit details").click()
    page.get_by_role("dialog").get_by_role("textbox", name="Code").fill("TA")
    page.get_by_role("dialog").get_by_role("button", name="Save").click()
    expect(page.get_by_role("dialog")).to_contain_text("Another trust already has the code TA")
