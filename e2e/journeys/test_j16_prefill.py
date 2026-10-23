"""J16 Pre-fill, the patient's half (S25; PRE-01–03): a patient whose first episode (the seeded Demo
pre-op HQ) was sent and reviewed, with one answer changed by the clinician, gets a second episode on
the seeded Demo follow-up HQ, which asks some of the same questions. On their phone each Question
Set opens on a card saying how many answers were filled in from their earlier questionnaire; they
choose "Everything is still correct" in one set (it jumps to the first question that wasn't filled)
and "Check each answer" in the other, see the marker under each pre-filled answer (gone once they
change it, and nothing filled in again after a reload), answer the rest and send it. The buttons
pressed are recorded. Then the clinician's half (S26, PRE-04): Cara, validating the set, sees where
each pre-filled answer came from and how it matched, the one the patient changed marked "Changed by
patient" with the pre-filled answer, and the button the patient pressed; the POA Summary shows the
same. Needs the seed's S25 follow-up HQ."""

from datetime import datetime

from playwright.sync_api import BrowserContext, Page, expect

from support import choose, invite_path, sign_in, unique


def published(api, name: str) -> dict:
    """The seeded HQ's newest published version at the user's hospital."""
    hqs = [h for h in api.call("GET", api.h("/published-hqs")) if h["name"] == name]
    assert hqs, f"No published {name!r}. Run `make seed`."
    newest = max(hqs, key=lambda h: h["versionNo"])
    return {"name": name, "versionId": newest["versionId"], "chapters": {}}


def with_sets(api, episode: dict) -> dict:
    """Fills in the episode's HQ's Question Set ids, by name, from the clinician's view of it."""
    for c in api.call("GET", api.h(f"/episodes/{episode['id']}/hq"))["chapters"]:
        episode["hq"]["chapters"][c["name"]] = c["id"]
    return episode


def test_a_returning_patient_confirms_their_pre_filled_answers_on_a_phone(phone: BrowserContext, page: Page, as_user, patient_link):
    cara = as_user("Cara Clinician")
    last_name = f"J16 {unique()}"

    # The first episode, on the Demo pre-op HQ: sent by the patient, reviewed by Cara, who
    # corrects "Do you snore loudly?" from No to Yes.
    first = with_sets(cara, cara.episode(published(cara, "Demo pre-op HQ"), last_name))
    about = {"smoker": "ex", "alcohol_units": 6, "height": 170, "weight": 82}
    health = {"q_diabetes": "o_no", "snoring": "no", "pregnant": "no", "ponv_history": "yes",
              "anaesthetic_problems": "yes", "anaesthetic_problems_details": "I was sick all night",
              "family_anaesthetic_problems": "no",
              "allergies": [{"allergies_what": "Penicillin", "allergies_reaction": "A rash"}]}
    p = patient_link(first)
    p.answer("About you", about)
    p.answer("Your health", health)
    p.submit()
    cara.validate(first, "About you", about)
    cara.validate(first, "Your health", {**health, "snoring": "yes"})
    cara.validate(first, "Nurse checks", {"q_bp": "120/80"})
    cara.complete_review(first)
    sent_on = datetime.now()

    # The second episode, on the follow-up HQ, for the same patient (their details match the
    # trust's record, so it's linked to it).
    second = with_sets(cara, cara.episode(published(cara, "Demo follow-up HQ"), last_name, email=first["email"]))
    assert second["patientId"] == first["patientId"]

    patient = phone.new_page()
    patient.goto(invite_path(first["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("heading", name="Demo follow-up HQ")).to_be_visible()

    # Both sets were filled in, so both wait to be checked, and the HQ can't be sent yet.
    you = patient.get_by_test_id("set-About you")
    yours = patient.get_by_test_id("set-Your health")
    expect(you.get_by_text("Please check")).to_be_visible()
    expect(yours.get_by_text("Please check")).to_be_visible()
    expect(patient.get_by_text("0 of 2 sections complete")).to_be_visible()
    expect(patient.get_by_role("button", name="Send my answers")).to_be_disabled()

    # Your health: diabetes (by its wording), snoring (the clinician's Yes), sickness after an
    # anaesthetic, a problem with one and what happened, and allergies. Not pregnancy (Always ask
    # fresh).
    day = f"{sent_on.day} {sent_on:%b %Y}"  # en-GB, make dev's default locale
    yours.get_by_role("button", name="Check").click()
    card = patient.get_by_test_id("prefill-card")
    expect(card).to_contain_text(f"We've filled in 6 answers from your questionnaire on {day}. Please check them.")
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
    card.get_by_role("button", name="Everything is still correct").click()
    # It goes to the first question that wasn't filled in: the pregnancy question.
    pregnant = patient.get_by_role("radiogroup", name="Is there any chance you could be pregnant?")
    expect(pregnant).to_be_visible()
    # Scrolled to with its text on screen, not past it (test report 1, F1).
    question = patient.locator(".sd-question", has=patient.get_by_text("Is there any chance you could be pregnant?"))
    expect(question.get_by_role("radio", name="Yes", exact=True)).to_be_focused()
    patient.wait_for_timeout(1200)  # past the second the page keeps it in place while SurveyJS settles
    top = question.evaluate("el => el.getBoundingClientRect().top")
    assert 0 <= top < patient.viewport_size["height"] / 2, f"the question's top is at {top}px"
    expect(pregnant.get_by_role("radio", name="No", exact=True)).not_to_be_checked()
    expect(patient.get_by_role("radiogroup", name="Do you have diabetes?").get_by_role("radio", name="No", exact=True)).to_be_checked()
    snoring = patient.get_by_role("radiogroup", name="Do you snore loudly?")
    expect(snoring.get_by_role("radio", name="Yes", exact=True)).to_be_checked()
    markers = patient.get_by_test_id("prefill-marker")
    expect(markers).to_have_count(5)
    expect(markers.first).to_have_text(f"From your answers on {day}")
    choose(patient, "Is there any chance you could be pregnant?", "No")
    # Changing a pre-filled answer takes its marker away.
    choose(patient, "Have you ever been sick, or felt very sick, after an anaesthetic?", "No")
    expect(markers).to_have_count(4)
    # A pre-filled answer its question's condition hides isn't lost: it's back when it shows again.
    details = patient.get_by_label("What happened with your anaesthetic?")
    choose(patient, "Have you ever had a problem with an anaesthetic?", "No")
    expect(details).to_be_hidden()
    expect(markers).to_have_count(2)
    choose(patient, "Have you ever had a problem with an anaesthetic?", "Yes")
    expect(details).to_have_value("I was sick all night")
    expect(markers).to_have_count(4)
    # Leaving and coming back: nothing is filled in again over the change, and no card.
    patient.get_by_role("button", name="All sections").click()
    expect(yours.get_by_text("Complete")).to_be_visible()  # every required answer is there
    patient.reload()
    yours.get_by_role("button", name="Review").click()
    expect(patient.get_by_test_id("prefill-card")).to_have_count(0)
    expect(patient.get_by_role("radiogroup", name="Have you ever been sick, or felt very sick, after an anaesthetic?")
           .get_by_role("radio", name="No", exact=True)).to_be_checked()
    patient.get_by_role("button", name="Next", exact=True).click()
    expect(patient.get_by_label("What are you allergic to?")).to_have_value("Penicillin")
    expect(markers).to_have_count(1)  # under the allergies
    patient.get_by_role("button", name="Finish this section").click()
    expect(yours.get_by_text("Complete")).to_be_visible()

    # About you: smoking, alcohol and height; not weight (Always ask fresh) or the new question.
    you.get_by_role("button", name="Check").click()
    expect(card).to_contain_text("We've filled in 3 answers")
    card.get_by_role("button", name="Check each answer").click()
    smoker = patient.get_by_role("radiogroup", name="Do you smoke?")
    expect(smoker.get_by_role("radio", name="I used to smoke")).to_be_checked()
    expect(markers).to_have_count(2)  # smoking and alcohol
    patient.get_by_role("button", name="Next", exact=True).click()
    expect(patient.get_by_label("What do you weigh now?")).to_have_value("")
    patient.get_by_label("What do you weigh now?").fill("80")
    choose(patient, "Can you climb two flights of stairs without stopping?", "Yes")
    patient.get_by_role("button", name="Finish this section").click()
    expect(you.get_by_text("Complete")).to_be_visible()

    # They send it.
    patient.get_by_role("button", name="Send my answers").click()
    patient.get_by_role("dialog", name="Send your answers?").get_by_role("button", name="Send my answers").click()
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()

    # The buttons pressed are recorded per set, with the answers' sources.
    rows = cara.call("GET", cara.h(f"/episodes/{second['id']}/hq"))["answers"]
    mine = {r["chapterId"]: r for r in rows if r["actor"] == "patient"}
    assert mine[second["hq"]["chapters"]["Your health"]]["prefillChoice"] == "still_correct"
    assert mine[second["hq"]["chapters"]["About you"]]["prefillChoice"] == "check_each"
    sources = mine[second["hq"]["chapters"]["Your health"]]["prefill"]
    assert sources["fu_diabetes"]["matchedBy"] == "wording" and sources["snoring"]["value"] == "yes"
    assert mine[second["hq"]["chapters"]["Your health"]]["data"]["ponv_history"] == "no"

    # The clinician's half (S26): Cara validates Your health.
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{second['id']}/sets/{second['hq']['chapters']['Your health']}")
    expect(page.get_by_role("heading", name="Your health")).to_be_visible()
    line = page.get_by_test_id("prefill-set")
    expect(line).to_contain_text("6 answers here were pre-filled from the patient's earlier questionnaires, and the patient changed 1.")
    expect(line).to_contain_text(f"The patient chose “Everything is still correct” on {datetime.now().day} ")
    origins = page.get_by_test_id("prefill-origin")
    diabetes = page.locator(".sd-question", has=page.get_by_text("Do you have diabetes?", exact=True))
    expect(diabetes.get_by_test_id("prefill-origin")).to_have_text(f"Pre-filled from Demo pre-op HQ, {day} (matched by same wording)")
    snore = page.locator(".sd-question", has=page.get_by_text("Do you snore loudly?", exact=True))
    expect(snore.get_by_test_id("prefill-origin")).to_have_text(f"Pre-filled from Demo pre-op HQ, {day} (matched by link)")
    expect(origins.first).to_be_visible()
    changed = page.get_by_test_id("prefill-changed")
    expect(changed).to_have_count(1)
    expect(changed).to_have_text("Changed by patient: pre-filled “Yes”")
    sick = page.locator(".sd-question", has=page.get_by_text("Have you ever been sick, or felt very sick, after an anaesthetic?", exact=True))
    expect(sick.get_by_test_id("prefill-changed")).to_be_visible()
    # A question that wasn't pre-filled has no marker: pregnancy (Always ask fresh).
    pregnancy = page.locator(".sd-question", has=page.get_by_text("Is there any chance you could be pregnant?", exact=True))
    expect(pregnancy.get_by_test_id("prefill-origin")).to_have_count(0)

    # The POA Summary shows the same.
    page.goto(f"{base}/episodes/{second['id']}/poa")
    health_section = page.get_by_role("region", name="Your health")
    expect(health_section.get_by_test_id("poa-prefill-choice")).to_contain_text("The patient chose “Everything is still correct”")
    expect(health_section.get_by_test_id("poa-prefilled").first).to_contain_text(f"Pre-filled from Demo pre-op HQ, {day}")
    expect(health_section.get_by_text("Changed by patient")).to_have_count(1)
    expect(health_section).to_contain_text("pre-filled “Yes”")


def test_everything_still_correct_moves_on_when_nothing_is_left(phone: BrowserContext, as_user, patient_link):
    """A set whose every question was filled in: "Everything is still correct" opens the next set,
    and after the last one the list, where both are complete and the HQ can be sent."""
    hq = as_user("Alex Author").published_hq("prefill-hq.json", f"E2E J16 {unique()}")
    cara = as_user("Cara Clinician")
    last_name = f"J16b {unique()}"
    first = cara.episode(hq, last_name)
    p = patient_link(first)
    p.answer("Smoking", {"smoker": "never"})
    p.answer("Sleep", {"snoring": "no"})
    p.submit()
    cara.episode(hq, last_name, email=first["email"])

    patient = phone.new_page()
    patient.goto(invite_path(first["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_test_id("set-Smoking").get_by_role("button", name="Check").click()
    card = patient.get_by_test_id("prefill-card")
    expect(card).to_contain_text("We've filled in 1 answer from your questionnaire on")
    card.get_by_role("button", name="Everything is still correct").click()
    expect(patient.get_by_role("heading", name="Sleep")).to_be_visible()  # the next set, on its own card
    card.get_by_role("button", name="Everything is still correct").click()
    expect(patient.get_by_text("2 of 2 sections complete")).to_be_visible()
    expect(patient.get_by_role("button", name="Send my answers")).to_be_enabled()
