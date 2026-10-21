"""J3 Patient: the clinician creates an episode, which texts and emails the patient their invite
(captured, S14). The patient opens the link on a phone, taps Continue, confirms their date of
birth, fills every Question Set (leaving part-way and coming back keeps the answers) and sends it.
The episode is then Ready for review."""

import re

from playwright.sync_api import BrowserContext, Page, expect

from support import choose, invite_path, nhs_number, pick, sign_in, unique

# The new patient's date of birth (Mantine adds " *" to a required field's label), not the search's.
DOB = re.compile(r"^Date of birth\W*$")


def test_patient_fills_in_and_sends_their_hq_on_a_phone(page: Page, phone: BrowserContext, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J3 {unique()}")
    last_name = f"J3 {unique()}"

    # The clinician creates the episode on the New episode page (S13). The search doesn't find
    # this new (made-up) patient, so she types their details from the referral.
    nhs = nhs_number()
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes")
    page.get_by_role("link", name="New episode").click()
    expect(page).to_have_url(f"{base}/episodes/new")
    page.get_by_label("Search", exact=True).fill(nhs)
    expect(page.get_by_text("No patients at this hospital match.")).to_be_visible()
    page.get_by_label("First name").fill("Sam")
    page.get_by_label("Surname").fill(last_name)
    page.get_by_label(DOB).fill("1980-05-17")
    pick(page, "Sex", "Female")
    page.get_by_label("NHS number").fill(f"{nhs[:3]} {nhs[3:6]} {nhs[6:]}")
    page.get_by_label("Mobile").fill("07700 900123")
    page.get_by_label("Hospital number").fill("J3-1")
    # The procedure from the trust's list (S19) chooses its default HQ (the seed's demo HQ); she
    # picks the journey's HQ instead, and the dates.
    pick(page, "Procedure", "Knee arthroscopy (KNEE-02)")
    expect(page.get_by_role("combobox", name="Health questionnaire")).to_have_value("Demo pre-op HQ v1")
    pick(page, "Health questionnaire", f"{hq['name']} v1")
    page.get_by_label("Planned procedure date").fill("2026-11-03")
    page.get_by_label("POA appointment date").fill("2026-10-28")
    page.get_by_role("button", name="Create episode").click()
    expect(page.get_by_role("heading", name=f"Sam {last_name}")).to_be_visible()
    # The NHS number and mobile are stored cleaned up; the hospital number is the episode's.
    expect(page.get_by_text(nhs, exact=True)).to_be_visible()
    expect(page.get_by_text("+447700900123", exact=True)).to_be_visible()
    expect(page.get_by_text("J3-1", exact=True)).to_be_visible()
    expect(page.get_by_role("combobox", name="Episode status")).to_have_value("HQ not complete")
    expect(page.get_by_role("combobox", name="Episode status")).to_be_disabled()
    expect(page.get_by_text("Episode created", exact=True)).to_be_visible()  # General notes
    expect(page.get_by_text("Invite sent by text message.", exact=True)).to_be_visible()  # the mobile only
    # Dates use the deployment's locale from GET /api/config (A-20): en-GB, make dev's default.
    expect(page.get_by_text("17 May 1980 (age")).to_be_visible()
    expect(page.get_by_text("Knee arthroscopy", exact=True)).to_be_visible()
    expect(page.get_by_text("03/11/2026", exact=True)).to_be_visible()  # the planned procedure date
    expect(page.get_by_text("28/10/2026", exact=True)).to_be_visible()  # the POA appointment date
    expect(page.get_by_text(f"{hq['name']} v1", exact=True)).to_be_visible()

    # The patient taps the link in their text message (captured, not sent), on their phone.
    patient = phone.new_page()
    patient.goto(invite_path("+447700900123"))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("heading", name=hq["name"])).to_be_visible()
    expect(patient.get_by_text("Hello Sam.")).to_be_visible()
    # Only the patient's two Question Sets: the clinician's "Nurse checks" never appears.
    expect(patient.get_by_text("0 of 2 sections complete")).to_be_visible()
    expect(patient.get_by_text("Nurse checks")).to_have_count(0)
    # Nothing is wider than the phone's screen (no sideways scrolling).
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
    send = patient.get_by_role("button", name="Send my answers")
    expect(send).to_be_disabled()

    # They answer the first page of "About you", then leave part-way…
    about = patient.get_by_test_id("set-About you")
    about.get_by_role("button", name="Start").click()
    choose(patient, "Do you smoke?", "Yes")
    patient.get_by_role("button", name="All sections").click()
    expect(about.get_by_text("In progress")).to_be_visible()

    # …and come back later: the answer was kept.
    patient.reload()
    about.get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("radiogroup", name="Do you smoke?").get_by_role("radio", name="Yes")).to_be_checked()
    patient.get_by_role("button", name="Next", exact=True).click()
    choose(patient, "How often do you drink alcohol?", "Sometimes")
    patient.get_by_role("button", name="Finish this section").click()
    expect(about.get_by_text("Complete")).to_be_visible()

    # The second Question Set.
    health = patient.get_by_test_id("set-Your health")
    health.get_by_role("button", name="Start").click()
    choose(patient, "Do you have diabetes?", "No")
    patient.get_by_role("button", name="Finish this section").click()
    expect(health.get_by_text("Complete")).to_be_visible()
    expect(patient.get_by_text("2 of 2 sections complete")).to_be_visible()

    # They send their answers, after confirming.
    send.click()
    confirm = patient.get_by_role("dialog", name="Send your answers?")
    confirm.get_by_role("button", name="Send my answers").click()
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()
    patient.reload()  # opening the link again shows the thank-you page, not the form
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()

    # The clinician sees the episode is ready for review, and the invite is no longer offered.
    page.reload()
    expect(page.get_by_role("combobox", name="Episode status")).to_have_value("Ready for review")
    expect(page.get_by_test_id("invite")).to_be_hidden()
