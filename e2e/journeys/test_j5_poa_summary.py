"""J5 POA Summary: both tabs. The corrected answer is marked on the validated tab. Change the
episode status; a comment is added to General notes. The Print button exists."""

from playwright.sync_api import Page, expect

from support import PATIENT_ANSWERS, pick, sign_in, unique


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

    hospital = sign_in(page.context, "Cara Clinician")
    page.goto(f"/h/{hospital}/episodes/{episode['id']}")
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
    pick(page, "Episode status", "POA complete")
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
