"""J5 POA Summary: both tabs. The corrected answer is marked on the validated tab. Change the
episode status; a comment is added to General notes. The Print button exists.

Its second test (S27): each answer is a link. A disclosure on the validated tab, the clinician's
change and the patient's answer each open Validate for that Question Set at the question,
highlighted; Back (the browser's, or Validate's "POA Summary" link) returns to the same tab, with
the answer clicked still on the screen."""

import re

from playwright.sync_api import Page, expect

from support import PATIENT_ANSWERS, change_status, sign_in, unique


def test_poa_summary_shows_both_tabs_and_records_status_and_comments(page: Page, as_user, patient_link):
    # A reviewed episode: the patient said they smoke; the clinician corrected it to No.
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J5 {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"J5 {unique()}")
    patient = patient_link(episode)
    for set_name, answers in PATIENT_ANSWERS.items():
        patient.answer(set_name, answers)
    patient.submit()
    cara.validate(episode, "About you", {**PATIENT_ANSWERS["About you"], "q_smokes": "o_no"})
    cara.validate(episode, "Your health", PATIENT_ANSWERS["Your health"])
    cara.validate(episode, "Nurse checks", {"q_bp": "120/80"})
    cara.complete_review(episode)

    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    page.get_by_role("link", name="POA Summary").click()
    expect(page.get_by_role("heading", name="POA Summary")).to_be_visible()
    expect(page.get_by_role("button", name="Print")).to_be_enabled()

    # Validated summary: the clinician's answer, marked as changed.
    expect(page.get_by_role("tab", name="Validated summary")).to_have_attribute("aria-selected", "true")
    about = page.get_by_role("region", name="About you")
    expect(about.get_by_text("Non-smoker", exact=True)).to_be_visible()
    expect(about.get_by_text("Changed by clinician")).to_be_visible()
    expect(about.get_by_role("alert", name="Changed by the clinician on Smoking")).to_contain_text(
        "Do you smoke? Patient answered “Yes”, now “No”")

    # Patient answers: what the patient said, with no change marked.
    page.get_by_role("tab", name="Patient answers").click()
    expect(about.get_by_text("Smoker", exact=True)).to_be_visible()
    expect(about.get_by_text("Non-smoker", exact=True)).to_be_hidden()
    expect(about.get_by_text("Changed by clinician")).to_be_hidden()

    # The status changes, and General notes records it.
    notes = page.get_by_role("region", name="General notes")
    change_status(page, "POA complete")  # S20: confirmed in a short form
    expect(notes.get_by_text("Status changed to POA complete", exact=True)).to_be_visible()

    # A comment is added to General notes.
    notes.get_by_role("button", name="Add comment").click()
    notes.get_by_role("textbox", name="Comment").fill("Made-up comment: fit for surgery.")
    notes.get_by_role("button", name="Save comment").click()
    expect(notes.get_by_text("Made-up comment: fit for surgery.", exact=True)).to_be_visible()

    # Both are kept after a reload.
    page.reload()
    expect(page.get_by_role("combobox", name="Episode status")).to_have_value("POA complete")
    expect(notes.get_by_text("Made-up comment: fit for surgery.", exact=True)).to_be_visible()
    expect(notes.get_by_text("Status changed to POA complete", exact=True)).to_be_visible()


def test_poa_summary_answers_open_validate_at_their_question_and_back_returns(page: Page, as_user, patient_link):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J5b {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"J5b {unique()}")
    patient = patient_link(episode)
    for set_name, answers in PATIENT_ANSWERS.items():
        patient.answer(set_name, answers)
    patient.submit()
    cara.validate(episode, "About you", {**PATIENT_ANSWERS["About you"], "q_smokes": "o_no"})
    cara.validate(episode, "Your health", PATIENT_ANSWERS["Your health"])
    cara.validate(episode, "Nurse checks", {"q_bp": "120/80"})

    # A short window, so the answers are below the fold and coming back to them is a real test.
    page.set_viewport_size({"width": 1280, "height": 420})
    base = sign_in(page.context, "Cara Clinician")
    poa = f"{base}/episodes/{episode['id']}/poa"
    about_set = episode["hq"]["chapters"]["About you"]
    at_smoking = re.compile(rf"/sets/{about_set}\?q=q_smokes$")
    page.goto(poa)
    about = page.get_by_role("region", name="About you")

    def opened_at_smoking() -> None:
        expect(page).to_have_url(at_smoking)
        question = page.locator('[data-name="q_smokes"]')
        expect(question).to_have_attribute("data-jumped", "true")  # highlighted for a moment
        expect(question).to_be_in_viewport()
        expect(page.get_by_test_id("back-to-poa")).to_be_visible()

    # The validated tab's disclosure: Validate at "Do you smoke?"; the browser's Back returns.
    link = about.get_by_role("link", name="Non-smoker", exact=True)
    link.click()
    opened_at_smoking()
    page.go_back()
    expect(page.get_by_role("tab", name="Validated summary")).to_have_attribute("aria-selected", "true")
    expect(about.get_by_role("link", name="Non-smoker", exact=True)).to_be_in_viewport()

    # The clinician's change: the same question; Validate's own "POA Summary" link returns.
    about.get_by_role("link", name=re.compile("^Do you smoke\\? Patient answered")).click()
    opened_at_smoking()
    page.get_by_test_id("back-to-poa").click()
    expect(page).to_have_url(re.compile(r"/poa$"))
    expect(about.get_by_role("link", name=re.compile("^Do you smoke\\? Patient answered"))).to_be_in_viewport()

    # The patient's answer, on the Patient answers tab: Back returns to that tab, kept in the address.
    page.get_by_role("tab", name="Patient answers").click()
    expect(page).to_have_url(re.compile(r"/poa\?tab=patient$"))
    about.get_by_role("link", name="Smoker", exact=True).click()
    opened_at_smoking()
    page.go_back()
    expect(page.get_by_role("tab", name="Patient answers")).to_have_attribute("aria-selected", "true")
    expect(about.get_by_role("link", name="Smoker", exact=True)).to_be_in_viewport()
    page.reload()
    expect(page.get_by_role("tab", name="Patient answers")).to_have_attribute("aria-selected", "true")
