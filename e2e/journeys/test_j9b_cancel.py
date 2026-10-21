"""J9b Cancel an episode with a reason (S20; EPI-03, S20 answers 3 and 5).

  - The patient opens their invite on their phone (iPhone 13) and starts their questionnaire.
  - Cara cancels the episode: "Other" needs a sentence; with one, the episode is cancelled, the
    status control offers nothing more, and the status history shows the change, who and why.
  - On the patient's phone the session has ended, the invite link no longer works, and when they
    sign in by email their home shows the questionnaire "Cancelled by the hospital", with no button.

J9 (the full lifecycle, with archive) comes in S22.
"""

import re
import secrets

from playwright.sync_api import BrowserContext, Page, expect

from support import invite_path, link_path, sign_in, unique


def enter_dob(patient: Page, day: str, month: str, year: str) -> None:
    patient.get_by_label("Day").fill(day)
    patient.get_by_label("Month").fill(month)
    patient.get_by_label("Year").fill(year)
    patient.get_by_role("button", name="Continue").click()


def test_j9b_cancel_with_a_reason_ends_the_patients_side(page: Page, phone: BrowserContext, as_user):
    day, month, year = secrets.randbelow(28) + 1, secrets.randbelow(12) + 1, 1940 + secrets.randbelow(50)
    dob = f"{year}-{month:02d}-{day:02d}"
    email = f"j9b.{secrets.token_hex(4)}@e2e.example"
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J9b {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J9b {unique()}", email=email, dateOfBirth=dob)

    # The patient opens their invite on their phone.
    patient = phone.new_page()
    invite = invite_path(email)
    patient.goto(invite)
    patient.get_by_role("button", name="Continue").click()
    enter_dob(patient, str(day), str(month), str(year))
    expect(patient.get_by_role("heading", name=hq["name"])).to_be_visible()

    # Cara cancels it. "Other" needs a sentence.
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    control = page.get_by_test_id("status-control")
    control.get_by_role("button", name="Cancel episode").click()
    form = page.get_by_test_id("status-form")
    expect(form.get_by_role("heading", name="Cancel this episode")).to_be_visible()
    form.get_by_role("radio", name="Other", exact=True).check()
    form.get_by_role("button", name="Cancel episode").click()
    expect(form.get_by_text("Say why the episode is cancelled.")).to_be_visible()
    form.get_by_label("Say why").fill("Made-up reason: the patient has moved abroad")
    form.get_by_label("Say why").press("Enter")  # Enter sends it (test report 1, F-1)

    # Cancelled: final, with the reason; nothing more is offered.
    expect(page.get_by_test_id("cancelled")).to_contain_text("Reason: Other: Made-up reason: the patient has moved abroad")
    expect(control.get_by_role("combobox", name="Episode status")).to_have_value("Cancelled")
    expect(control.get_by_role("combobox", name="Episode status")).to_be_disabled()
    expect(control.get_by_role("button", name="Cancel episode")).to_have_count(0)
    expect(page.get_by_test_id("invite")).to_have_count(0)
    history = page.get_by_test_id("status-history-row")
    expect(history).to_have_count(2)
    expect(history.nth(0)).to_contain_text("Created: HQ not complete")
    expect(history.nth(1)).to_contain_text("Cancelled: HQ not complete → Cancelled")
    expect(history.nth(1)).to_contain_text("Cara Clinician")
    expect(history.nth(1)).to_contain_text("Other: Made-up reason: the patient has moved abroad")
    page.reload()
    expect(page.get_by_test_id("cancelled")).to_be_visible()

    # The patient's session has ended, and the invite link no longer works.
    patient.reload()
    expect(patient.get_by_test_id("patient-signed-out")).to_be_visible()
    patient.goto(invite)
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("heading", name="This link doesn't work")).to_be_visible()

    # Signing in by email, their home shows it cancelled by the hospital, with no form.
    patient.goto("/signin")
    patient.get_by_label("Mobile number or email address").fill(email)
    patient.get_by_role("button", name="Send me a link").click()
    expect(patient.get_by_role("heading", name="Check your email")).to_be_visible()
    patient.goto(link_path(email))
    patient.get_by_role("button", name="Continue").click()
    enter_dob(patient, str(day), str(month), str(year))
    expect(patient).to_have_url(re.compile(r"/home$"))
    card = patient.get_by_test_id("home-episode")
    expect(card).to_have_count(1)
    expect(card).to_contain_text("Cancelled by the hospital")
    expect(card.get_by_role("link")).to_have_count(0)
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
    patient.goto(f"/home/episodes/{episode['id']}")
    expect(patient.get_by_text("This questionnaire was cancelled by the hospital")).to_be_visible()
    patient.get_by_role("button", name="Sign out").click()  # test report 1: Sign out is on this page too
    expect(patient.get_by_test_id("patient-signed-out")).to_be_visible()
