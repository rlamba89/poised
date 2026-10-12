"""J4 Clinician validates: opens each Question Set, corrects one answer (a "Patient answered…" note
appears), validates every set and completes the review. The episode is then Ready for POA.

Two tabs (Rahul, 9 Oct; guards S02a's fix): a stale tab's Validate gets the "changed somewhere
else" message with Reload, and the newer answer is kept."""

from playwright.sync_api import Page, expect

from support import PATIENT_ANSWERS, choose, sign_in, unique


def is_answers_save(response) -> bool:
    return response.request.method == "PUT" and "/answers/" in response.url


def test_clinician_validates_every_set_and_completes_the_review(page: Page, as_user, patient_link):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J4 {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J4 {unique()}")
    patient = patient_link(episode)
    for set_name, answers in PATIENT_ANSWERS.items():  # Smoker, drinks sometimes, no diabetes
        patient.answer(set_name, answers)
    patient.submit()

    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    expect(page.get_by_role("combobox", name="Episode status")).to_have_value("Ready for review")
    complete = page.get_by_role("button", name="Complete HQ review")
    expect(complete).to_be_disabled()

    # "About you" is open in two tabs.
    page.get_by_role("row", name="About you").get_by_role("link", name="Validate").click()
    expect(page.get_by_role("heading", name="About you")).to_be_visible()
    stale = page.context.new_page()
    stale.goto(page.url)
    expect(stale.get_by_role("radiogroup", name="Do you smoke?").get_by_role("radio", name="Yes")).to_be_checked()

    # The first tab corrects the patient's answer, and it saves.
    with page.expect_response(is_answers_save) as saved:
        choose(page, "Do you smoke?", "No")
    assert saved.value.ok
    expect(page.get_by_text("Patient answered: Yes")).to_be_visible()
    expect(page.get_by_test_id("validate-save-status")).to_have_text("Saved")

    # The second tab, still showing "Yes", can't validate over it…
    stale.get_by_role("button", name="Validate").click()
    conflict = stale.get_by_role("alert", name="Not saved")
    expect(conflict).to_contain_text("These answers were changed somewhere else")
    expect(stale.get_by_role("button", name="Validate")).to_be_disabled()

    # …and Reload shows the first tab's answer, which it then validates.
    conflict.get_by_role("button", name="Reload").click()
    expect(stale.get_by_role("radiogroup", name="Do you smoke?").get_by_role("radio", name="No")).to_be_checked()
    expect(stale.get_by_text("Patient answered: Yes")).to_be_visible()
    stale.get_by_role("button", name="Validate").click()
    expect(stale.get_by_role("row", name="About you")).to_contain_text("Validated by Cara Clinician")
    stale.close()

    # "Your health" is validated as the patient answered it.
    page.goto(f"{base}/episodes/{episode['id']}")
    expect(page.get_by_role("row", name="About you")).to_contain_text("Validated by Cara Clinician")
    page.get_by_role("row", name="Your health").get_by_role("link", name="Validate").click()
    expect(page.get_by_role("radiogroup", name="Do you have diabetes?").get_by_role("radio", name="No")).to_be_checked()
    page.get_by_role("button", name="Validate").click()
    expect(page.get_by_role("row", name="Your health")).to_contain_text("Validated by Cara Clinician")
    expect(complete).to_be_disabled()

    # "Nurse checks" is the clinician's own: Validate needs its required answer first.
    page.get_by_role("row", name="Nurse checks").get_by_role("link", name="Validate").click()
    page.get_by_role("button", name="Validate").click()
    expect(page.get_by_text("Some required questions have no answer yet.")).to_be_visible()
    page.get_by_role("textbox", name="Blood pressure").fill("120/80")
    page.get_by_role("button", name="Validate").click()
    expect(page.get_by_role("row", name="Nurse checks")).to_contain_text("Validated by Cara Clinician")

    # Every set is validated, so the review can be completed.
    complete.click()
    expect(page.get_by_role("combobox", name="Episode status")).to_have_value("Ready for POA")
    expect(page.get_by_text("HQ review completed by Cara Clinician", exact=True)).to_be_visible()  # General notes
    expect(page.get_by_role("row", name="About you").get_by_role("link", name="View")).to_be_visible()
