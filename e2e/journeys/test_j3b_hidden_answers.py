"""J3b Hidden answers (S44c, Rahul 28 Oct): the patient's form clears an answer as soon as a condition
hides its question, so what's on screen is what's saved and sent. The chain at phone size: "Short of
breath?" Yes shows "Worse at night?", whose Yes shows "What wakes you?" and the Pillows
Question Set; back to No, all three go at once (on screen and saved), a reload agrees, and Yes again
shows them empty.

Then (S44c answer 5) the clinician's correction brings them back on Validate: a question the
patient's form never showed has no "Patient answered" line there or on the POA Summary, while the
corrected question keeps its line.

Its second test (verification 2, F3): a typed number edited 20 → 25 (Backspace, then 5) hides its
follow-up only for an instant, so the follow-up's answer is kept, on screen and stored; the clearing
happens at the autosave after the pause in typing. Left at 2, the answer goes. Its third
(verification 3, F4): the same for a BMI's weight, 120 → 125, whose follow-up is on the next page."""

from playwright.sync_api import BrowserContext, Page, expect

from support import choose, invite_path, sign_in, unique


def is_answers_save(response) -> bool:
    return response.request.method == "PUT" and "/answers/" in response.url


def test_a_hidden_follow_up_is_cleared_at_once_and_a_clinicians_correction_has_no_patient_line(
        page: Page, phone: BrowserContext, as_user, patient_link):
    hq = as_user("Alex Author").published_hq("hidden-answers-hq.json", f"E2E J3b {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"J3b {unique()}")

    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_text("0 of 1 sections complete")).to_be_visible()

    # B Yes → A Yes → C answered: the Pillows set is listed, and answered.
    breathing = patient.get_by_test_id("set-Breathing")
    breathing.get_by_role("button", name="Start").click()
    choose(patient, "Do you get short of breath?", "Yes")
    choose(patient, "Is it worse at night?", "Yes")
    patient.get_by_role("textbox", name="What wakes you?").fill("Coughing")
    patient.get_by_role("button", name="Finish this section").click()
    night = patient.get_by_test_id("set-Pillows")
    night.get_by_role("button", name="Start").click()
    choose(patient, "Do you use more than two pillows?", "Yes")
    patient.get_by_role("button", name="Finish this section").click()
    expect(patient.get_by_text("2 of 2 sections complete")).to_be_visible()

    # B back to No: A and C go at once, and the save reaches the server.
    breathing.get_by_role("button", name="Review").click()
    with patient.expect_response(is_answers_save) as saved:
        choose(patient, "Do you get short of breath?", "No")
    assert saved.value.ok
    expect(patient.get_by_role("radiogroup", name="Is it worse at night?")).to_have_count(0)
    expect(patient.get_by_role("textbox", name="What wakes you?")).to_have_count(0)
    expect(patient.get_by_test_id("patient-save-status")).to_have_text("Saved")
    patient.get_by_role("button", name="All sections").click()
    expect(night).to_have_count(0)  # the follow-up set is hidden…
    expect(patient.get_by_text("1 of 1 sections complete")).to_be_visible()

    # …and its answers are gone too, as a reload shows: Yes again brings A, C and the set back empty.
    patient.reload()
    breathing.get_by_role("button", name="Review").click()
    choose(patient, "Do you get short of breath?", "Yes")
    worse = patient.get_by_role("radiogroup", name="Is it worse at night?")
    expect(worse.get_by_role("radio", name="Yes")).not_to_be_checked()
    expect(worse.get_by_role("radio", name="No")).not_to_be_checked()
    choose(patient, "Is it worse at night?", "Yes")
    expect(patient.get_by_role("textbox", name="What wakes you?")).to_have_value("")
    # Back to No for the send: the patient isn't short of breath.
    choose(patient, "Do you get short of breath?", "No")
    patient.get_by_role("button", name="Finish this section").click()
    expect(night).to_have_count(0)
    rows = cara.call("GET", cara.h(f"/episodes/{episode['id']}/hq"))["answers"]
    stored = {r["chapterId"]: r["data"] for r in rows if r["actor"] == "patient"}
    assert stored.get(hq["chapters"]["Breathing"]) == {"q_e2e_hb": "o_no"}, stored
    assert stored.get(hq["chapters"]["Pillows"], {}) == {}, stored
    patient.get_by_role("button", name="Send my answers").click()
    patient.get_by_role("dialog", name="Send your answers?").get_by_role("button", name="Send my answers").click()
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()

    # The clinician corrects B to Yes and answers A and C, which the patient never saw: only B has a
    # "Patient answered" line. Pillows is then listed, and its answer has none either.
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    expect(page.get_by_role("combobox", name="Episode status")).to_have_value("Ready for review")
    page.get_by_role("row", name="Breathing").get_by_role("link", name="Validate").click()
    choose(page, "Do you get short of breath?", "Yes")
    expect(page.get_by_text("Patient answered: No")).to_be_visible()
    choose(page, "Is it worse at night?", "Yes")
    page.get_by_role("textbox", name="What wakes you?").fill("Coughing")
    page.get_by_role("button", name="Validate").click()
    expect(page.get_by_role("row", name="Breathing")).to_contain_text("Validated by Cara Clinician")
    expect(page.get_by_text("Patient answered:")).to_have_count(0)  # the episode page has none
    page.get_by_role("row", name="Pillows").get_by_role("link", name="Validate").click()
    choose(page, "Do you use more than two pillows?", "Yes")
    expect(page.get_by_test_id("validate-save-status")).to_have_text("Saved")
    expect(page.get_by_text("Patient answered:")).to_have_count(0)
    page.get_by_role("button", name="Validate").click()
    page.get_by_role("row", name="Breathing").get_by_role("link", name="Review").click()
    expect(page.get_by_text("Patient answered: No")).to_be_visible()
    expect(page.get_by_text("Patient answered:")).to_have_count(1)  # under B only

    # The POA Summary lists the change to B only.
    page.goto(f"{base}/episodes/{episode['id']}")
    page.get_by_role("link", name="POA Summary").click()
    changes = page.get_by_role("alert", name="Changed by the clinician on Breathing")
    expect(changes).to_contain_text("Do you get short of breath? Patient answered “No”, now “Yes”")
    expect(changes).not_to_contain_text("Is it worse at night?")
    expect(changes).not_to_contain_text("What wakes you?")
    expect(page.get_by_role("alert", name="Changed by the clinician on Pillows")).to_have_count(0)


def test_editing_a_typed_number_keeps_the_follow_up_it_still_shows(page: Page, phone: BrowserContext, as_user):
    hq = as_user("Alex Author").published_hq("hidden-number-hq.json", f"E2E J3b number {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"J3b {unique()}")

    def stored() -> dict:
        rows = cara.call("GET", cara.h(f"/episodes/{episode['id']}/hq"))["answers"]
        return next((r["data"] for r in rows if r["actor"] == "patient"), {})

    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_test_id("set-Alcohol").get_by_role("button", name="Review").click()  # nothing required: Complete
    units = patient.get_by_role("spinbutton", name="How many units of alcohol a week?")
    why = patient.get_by_role("textbox", name="Tell us more about your drinking")
    status = patient.get_by_test_id("patient-save-status")

    units.fill("20")
    why.fill("Wine with dinner")
    patient.wait_for_timeout(1500)  # past the autosave's pause
    expect(status).to_have_text("Saved")
    assert stored() == {"q_e2e_units": 20, "q_e2e_why": "Wine with dinner"}, stored()
    # The patient corrects 20 to 25 as a person does: Backspace, then 5.
    units.click()
    units.press("End")
    units.press("Backspace")
    units.press("5")
    expect(units).to_have_value("25")
    patient.wait_for_timeout(1500)  # past the autosave's pause
    expect(status).to_have_text("Saved")
    expect(why).to_have_value("Wine with dinner")
    assert stored() == {"q_e2e_units": 25, "q_e2e_why": "Wine with dinner"}, stored()

    # Left at 2, the follow-up hides and its answer goes at the save.
    units.press("End")
    units.press("Backspace")
    expect(units).to_have_value("2")
    patient.wait_for_timeout(1500)
    expect(status).to_have_text("Saved")
    expect(why).to_have_count(0)
    assert stored() == {"q_e2e_units": 2}, stored()


def test_editing_a_bmi_weight_keeps_the_follow_up_it_still_shows(page: Page, phone: BrowserContext, as_user):
    hq = as_user("Alex Author").published_hq("hidden-bmi-hq.json", f"E2E J3b BMI {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"J3b {unique()}")

    def stored() -> dict:
        rows = cara.call("GET", cara.h(f"/episodes/{episode['id']}/hq"))["answers"]
        return next((r["data"] for r in rows if r["actor"] == "patient"), {})

    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_test_id("set-Weight").get_by_role("button", name="Review").click()  # nothing required: Complete
    height = patient.get_by_role("spinbutton", name="What is your height?")
    weight = patient.get_by_role("spinbutton", name="What is your weight?")
    status = patient.get_by_test_id("patient-save-status")
    height.fill("170")
    weight.fill("120")
    patient.get_by_role("button", name="Next", exact=True).click()
    why = patient.get_by_role("textbox", name="Tell us about your weight")
    why.fill("Diet plan")
    patient.wait_for_timeout(1500)  # past the autosave's pause
    expect(status).to_have_text("Saved")
    assert stored().get("q_e2e_weight_why") == "Diet plan", stored()

    # The patient corrects the weight 120 → 125: Backspace, then 5.
    patient.get_by_role("button", name="Previous").click()
    weight.click()
    weight.press("End")
    weight.press("Backspace")
    weight.press("5")
    expect(weight).to_have_value("125")
    patient.wait_for_timeout(1500)
    expect(status).to_have_text("Saved")
    patient.get_by_role("button", name="Next", exact=True).click()
    expect(why).to_have_value("Diet plan")
    data = stored()
    assert data.get("q_e2e_weight_why") == "Diet plan" and data["q_e2e_bmi"]["weight"] == 125, data
