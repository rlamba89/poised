"""Patient records (S13, plan C2): the New episode page. A clinician searches her hospital's
patients, or types a new patient's details; the trust's existing record is linked to silently, and
nothing about another hospital is ever shown. Wrong NHS numbers, mismatches and possible
duplicates are explained. Made-up patients only; each run's NHS numbers and surnames are new."""

import re
import secrets

from playwright.sync_api import Page, expect

from support import nhs_number, pick, sign_in, unique

# The new patient's date of birth (Mantine adds " *" to a required field's label), not the search's.
DOB = re.compile(r"^Date of birth\W*$")


def new_episode(page: Page, base: str) -> None:
    page.goto(f"{base}/episodes")
    page.get_by_role("link", name="New episode").click()
    expect(page.get_by_role("heading", name="New episode")).to_be_visible()


def type_patient(page: Page, first: str, last: str, dob: str, nhs: str = "", mobile: str = "") -> None:
    page.get_by_label("First name").fill(first)
    page.get_by_label("Surname").fill(last)
    page.get_by_label(DOB).fill(dob)
    pick(page, "Sex", "Female")
    page.get_by_label("NHS number").fill(nhs)
    page.get_by_label("Mobile").fill(mobile)


def episode_details(page: Page, hq: dict, hospital_number: str) -> None:
    page.get_by_label("Hospital number").fill(hospital_number)
    pick(page, "Health questionnaire", f"{hq['name']} v1")


def test_a_patient_seen_at_the_other_hospital_is_linked_silently(page: Page, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E S13 {unique()}")
    nhs, surname = nhs_number(), f"Link {unique()}"
    # Hospital A2 saw the patient first, with its own hospital number.
    as_user("Val Clinician", "Hospital A2").episode(hq, surname, hospital_number="A2-SECRET", nationalId=nhs,
                                                    mobile="07700 900555", email="elsewhere@example.test")

    base = sign_in(page.context, "Cara Clinician")  # Hospital A1 only
    new_episode(page, base)
    # A1's search never finds a patient known only at A2: by NHS number, hospital number or surname.
    for term in [nhs, "A2-SECRET", surname]:
        page.get_by_label("Search", exact=True).fill(term)
        expect(page.get_by_text("No patients at this hospital match.")).to_be_visible()

    # Cara types the referral's details; the NHS number, date of birth and surname agree.
    type_patient(page, "S", surname, "1980-05-17", nhs=nhs)
    episode_details(page, hq, "A1-77")
    page.get_by_role("button", name="Create episode").click()
    expect(page.get_by_role("heading", name=re.compile(surname))).to_be_visible()
    # Nothing about Hospital A2: not its name, its hospital number or its episode.
    expect(page.get_by_text("A1-77", exact=True)).to_be_visible()
    expect(page.get_by_text("A2-SECRET")).to_have_count(0)
    expect(page.get_by_text("Hospital A2")).to_have_count(0)

    # The patient's page lists their episodes at A1 only.
    page.get_by_role("heading", name=re.compile(surname)).get_by_role("link").click()
    expect(page.get_by_role("heading", name="Episodes at Hospital A1")).to_be_visible()
    expect(page.get_by_role("table", name="Episodes").locator("tbody tr")).to_have_count(1)
    expect(page.get_by_text("A2-SECRET")).to_have_count(0)

    # From now on A1's search finds them, with A1's hospital number.
    new_episode(page, base)
    page.get_by_label("Search", exact=True).fill(nhs)
    results = page.get_by_role("table", name="Patients found")
    expect(results.locator("tbody tr")).to_have_count(1)
    expect(results).to_contain_text("A1-77")
    expect(results).not_to_contain_text("A2-SECRET")


def test_a_wrong_nhs_number_and_a_mismatch_are_refused_without_showing_the_record(page: Page, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E S13 {unique()}")
    nhs, surname = nhs_number(), f"Mismatch {unique()}"
    as_user("Val Clinician", "Hospital A2").episode(hq, surname, nationalId=nhs)  # born 1980-05-17

    base = sign_in(page.context, "Cara Clinician")
    new_episode(page, base)
    # A wrong check digit.
    bad = nhs[:9] + str((int(nhs[9]) + 1) % 10)
    type_patient(page, "Sam", surname, "1980-05-17", nhs=bad)
    episode_details(page, hq, "")
    page.get_by_role("button", name="Create episode").click()
    expect(page.get_by_role("alert")).to_contain_text("That isn't a valid NHS number")

    # The right NHS number with another date of birth: refused, with no stored values.
    page.get_by_label("NHS number").fill(nhs)
    page.get_by_label(DOB).fill("1981-06-18")
    page.get_by_role("button", name="Create episode").click()
    alert = page.get_by_role("alert")
    expect(alert).to_contain_text("The details don't match")
    expect(alert).not_to_contain_text("1980")
    expect(alert).not_to_contain_text("May")
    expect(page).to_have_url(re.compile(r"/episodes/new$"))  # nothing was created


def test_a_possible_duplicate_is_warned_about_and_can_be_created_anyway(page: Page, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E S13 {unique()}")
    elsewhere, here = f"Dup {unique()}", f"Here {unique()}"
    # The same name and date of birth at A2 (no NHS number), and one of A1's own patients.
    as_user("Val Clinician", "Hospital A2").episode(hq, elsewhere, mobile="07700 900666", email="dup@example.test")
    own = as_user("Cara Clinician").episode(hq, here, hospital_number="A1-OWN")

    base = sign_in(page.context, "Cara Clinician")
    new_episode(page, base)
    # Known only at A2, with another mobile: a warning that names nobody.
    type_patient(page, "Sam", elsewhere, "1980-05-17", mobile="07700 900777")
    episode_details(page, hq, "")
    page.get_by_role("button", name="Create episode").click()
    warning = page.get_by_role("alert").filter(has_text="Possible duplicate")
    expect(warning).to_contain_text("may already be registered in the trust")
    expect(warning).not_to_contain_text("dup@example.test")
    expect(warning).not_to_contain_text("+447700900666")
    expect(warning.get_by_role("button", name="Use this record")).to_have_count(0)
    warning.get_by_role("button", name="Create anyway").click()
    expect(page.get_by_role("heading", name=f"Sam {elsewhere}")).to_be_visible()
    expect(page.get_by_text("+447700900777", exact=True)).to_be_visible()  # a new record: what Cara typed

    # One of A1's own patients: the match is shown, and Cara uses that record.
    new_episode(page, base)
    type_patient(page, "Sam", here, "1980-05-17", mobile="07700 900888")
    episode_details(page, hq, "")
    page.get_by_role("button", name="Create episode").click()
    warning = page.get_by_role("alert").filter(has_text="Possible duplicate")
    expect(warning).to_contain_text(f"Sam {here}")
    expect(warning).to_contain_text("A1-OWN")
    warning.get_by_role("button", name="Use this record").click()
    expect(page.get_by_text(f"Sam {here}", exact=True)).to_be_visible()  # the chosen patient's card
    expect(page.get_by_label("Hospital number")).to_have_value("A1-OWN")
    page.get_by_role("button", name="Create episode").click()
    expect(page.get_by_role("heading", name=f"Sam {here}")).to_be_visible()
    page.get_by_role("heading", name=f"Sam {here}").get_by_role("link").click()
    expect(page).to_have_url(re.compile(own["patientId"]))
    expect(page.get_by_role("table", name="Episodes").locator("tbody tr")).to_have_count(2)


def test_search_finds_this_hospitals_patients_and_a_found_patient_gets_a_second_episode(page: Page, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E S13 {unique()}")
    surname, number = f"Zed{unique().replace(' ', '')}", f"F-{secrets.token_hex(3)}"
    nhs = nhs_number()
    # A date of birth of its own, so "Ze" + it finds this run's patient among earlier runs' Zeds.
    dob = f"19{secrets.randbelow(70) + 30}-{secrets.randbelow(12) + 1:02d}-{secrets.randbelow(28) + 1:02d}"
    as_user("Cara Clinician").episode(hq, surname, hospital_number=number, nationalId=nhs, dateOfBirth=dob)

    base = sign_in(page.context, "Cara Clinician")
    new_episode(page, base)
    search = page.get_by_label("Search", exact=True)
    results = page.get_by_role("table", name="Patients found")
    # Too short: nothing is searched.
    search.fill("Ze")
    expect(results).to_have_count(0)
    expect(page.get_by_text("No patients at this hospital match.")).to_have_count(0)
    # By surname, NHS number (with spaces) and hospital number (any case).
    for term in [surname, f"{nhs[:3]} {nhs[3:6]} {nhs[6:]}", number.lower()]:
        search.fill(term)
        expect(results.locator("tbody tr")).to_have_count(1)
        expect(results).to_contain_text(f"Sam {surname}")
    # A short surname with a date of birth.
    search.fill("Ze")
    page.get_by_label("Date of birth (optional)").fill(dob)
    expect(results.locator("tbody tr").filter(has_text=surname)).to_have_count(1)

    page.get_by_role("button", name=f"Choose Sam {surname}").click()
    expect(page.get_by_role("button", name="Change patient")).to_be_visible()
    expect(page.get_by_label("Hospital number")).to_have_value(number)
    # Change patient clears the number: it was that patient's (S13 test report 1).
    page.get_by_role("button", name="Change patient").click()
    expect(page.get_by_label("Hospital number")).to_have_value("")
    page.get_by_label("Search", exact=True).fill(surname)  # the search starts afresh
    page.get_by_role("button", name=f"Choose Sam {surname}").click()
    expect(page.get_by_label("Hospital number")).to_have_value(number)
    pick(page, "Health questionnaire", f"{hq['name']} v1")
    page.get_by_role("button", name="Create episode").click()
    expect(page.get_by_role("heading", name=f"Sam {surname}")).to_be_visible()
    page.get_by_role("heading", name=f"Sam {surname}").get_by_role("link").click()
    expect(page.get_by_role("table", name="Episodes").locator("tbody tr")).to_have_count(2)  # two open at once (PAT-02)
    # Reload and Back keep working on the patient's page.
    page.reload()
    expect(page.get_by_role("heading", name="Episodes at Hospital A1")).to_be_visible()
    page.go_back()
    expect(page.get_by_role("heading", name=f"Sam {surname}")).to_be_visible()
    # New episode from the patient's page: the patient is chosen and their hospital number filled in.
    page.go_forward()
    page.get_by_role("link", name="New episode").click()
    expect(page.get_by_role("button", name="Change patient")).to_be_visible()
    expect(page.get_by_label("Hospital number")).to_have_value(number)
    # Change patient forgets them in the address too, so a reload doesn't choose them again (test report 2).
    expect(page).to_have_url(re.compile(r"\?patient="))
    page.get_by_role("button", name="Change patient").click()
    expect(page).to_have_url(re.compile(r"/episodes/new$"))
    page.reload()
    expect(page.get_by_label("Search", exact=True)).to_be_visible()
    expect(page.get_by_role("button", name="Change patient")).to_have_count(0)
    # "Episodes" stays highlighted on a patient's page; Back returns to it.
    page.go_back()
    expect(page.get_by_role("heading", name="Episodes at Hospital A1")).to_be_visible()
    expect(page.get_by_role("banner").get_by_role("link", name="Episodes")).to_have_attribute("data-variant", "light")
