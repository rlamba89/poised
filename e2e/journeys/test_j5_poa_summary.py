"""J5 POA Summary: both tabs. The corrected answer is marked on the validated tab. Change the
episode status; a comment is added to General notes. The Print button exists.

Its second test (S27): each answer is a link. A disclosure on the validated tab, the clinician's
change and the patient's answer each open Validate for that Question Set at the question,
highlighted; Back (the browser's, or Validate's "POA Summary" link) returns to the same tab, with
the answer clicked still on the screen.

Its third (S28): a Question Set started fresh says so on both tabs, with Validate's wording, and
the Patient reported BMI comes from the patient's own height and weight before About you is
validated, then from the validated ones.

Its fifth (S39): the suggested ASA. The patient smokes (ASA II) and has heart failure (ASA III, from
the ASA guide's "moderate reduction of ejection fraction"). The episode page says "Suggested ASA III"
in one line and lists the clinician's Anaesthetic plan, shown only from ASA III. The POA Summary's ASA
section says it with its reason on both tabs, above the nurse's and anaesthetist's grades; the reason
opens Validate at its question. Cara corrects heart failure to No: the validated tab drops to ASA II
(the smoking reason) and the Anaesthetic plan goes, while the patient's tab still says ASA III.

Its fourth (S29): when the server can't be reached, the POA Summary says so plainly ("We couldn't
reach the server…"), never the browser's "Failed to fetch": a request that fails outright, and a
gateway's 502 with no body."""

import re

from playwright.sync_api import Page, expect

from support import PATIENT_ANSWERS, change_status, sign_in, unique
from test_j16b_prefill_cross_trust import trust_b_then_trust_a


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


def test_a_set_started_fresh_says_so_and_the_bmi_comes_from_the_patients_answers(page: Page, as_user, patient_link):
    _, second, cara, _ = trust_b_then_trust_a(as_user, patient_link, {"Your health": {"q_diabetes": "o_no", "snoring": "yes"}})
    # Cara fills in for the patient: Your health holds Trust B's answers, and is started fresh;
    # About you gets a height and a weight.
    form = cara.h(f"/episodes/{second['id']}/patient-form")
    hq = cara.call("GET", form)
    health, about = second["hq"]["chapters"]["Your health"], second["hq"]["chapters"]["About you"]
    assert hq["prefill"][health]["answers"]["snoring"]["otherTrust"] is True
    cara.call("POST", f"{form}/prefill/{health}", {"choice": "start_fresh"})
    cara.call("PUT", f"{form}/answers/{about}", {"data": {**hq["answers"].get(about, {}), "height": 170, "weight": 82},
                                                 "revision": hq["revisions"].get(about, 0)})

    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{second['id']}/poa")
    fresh = "Answers were pre-filled here from the patient's earlier questionnaires. Cara Clinician chose “Start fresh” with the patient on "
    for tab in ("Validated summary", "Patient answers"):
        page.get_by_role("tab", name=tab).click()
        expect(page.get_by_role("region", name="Your health").get_by_test_id("poa-started-fresh")).to_contain_text(fresh)
        expect(page.get_by_test_id("poa-detail-Patient reported BMI")).to_contain_text("28.4")  # 82 / 1.70²

    # Sent, then validated with another weight: the validated one counts. Every required answer
    # first (S44: the server refuses a send with a set incomplete), Your health given again.
    now = cara.call("GET", form)
    cara.call("PUT", f"{form}/answers/{about}", {"data": {**now["answers"].get(about, {}), "smoker": "never", "alcohol_units": 0,
                                                          "height": 170, "weight": 82, "fu_stairs": "yes"},
                                                 "revision": now["revisions"].get(about, 0)})
    cara.call("PUT", f"{form}/answers/{health}", {"data": {"snoring": "no", "pregnant": "no", "ponv_history": "no", "anaesthetic_problems": "no"},
                                                  "revision": now["revisions"].get(health, 0)})
    cara.call("POST", f"{form}/submit")
    as_user("Alex Author").review_by_hand(second)  # the seeded follow-up HQ's rules auto-triage it (S42)
    cara.validate(second, "About you", {"height": 170, "weight": 90})
    page.reload()
    expect(page.get_by_test_id("poa-detail-Patient reported BMI")).to_contain_text("31.1")


NETWORK_ERROR = "We couldn't reach the server. Check your connection and try again."


def test_the_poa_summary_says_plainly_when_the_server_cant_be_reached(page: Page, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J5d {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J5d {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    poa = f"{base}/episodes/{episode['id']}/poa"

    # Offline (the request never gets an answer): the page says so in words.
    page.route(re.compile(r"/api/.*/episodes/[^/]+/hq$"), lambda route: route.abort("internetdisconnected"))
    page.goto(poa)
    expect(page.get_by_role("alert")).to_have_text(NETWORK_ERROR)
    expect(page.get_by_text("Failed to fetch")).to_have_count(0)
    page.unroute_all()

    # Back online, the page loads; a comment that meets a gateway's empty 502 says the same.
    page.reload()
    notes = page.get_by_role("region", name="General notes")
    notes.get_by_role("button", name="Add comment").click()
    notes.get_by_role("textbox", name="Comment").fill("Made-up comment that won't arrive.")
    page.route(re.compile(r"/api/.*/episodes/[^/]+/notes$"), lambda route: route.fulfill(status=502, body=""))
    notes.get_by_role("button", name="Save comment").click()
    expect(notes.get_by_role("alert")).to_have_text(NETWORK_ERROR)

    # On AWS, API Gateway answers its own 5xx with JSON that isn't the API's (S30): the same words.
    page.unroute_all()
    page.reload()  # no earlier message on screen
    notes.get_by_role("button", name="Add comment").click()
    notes.get_by_role("textbox", name="Comment").fill("Made-up comment that won't arrive.")
    expect(notes.get_by_role("alert")).to_have_count(0)
    page.route(re.compile(r"/api/.*/episodes/[^/]+/notes$"), lambda route: route.fulfill(
        status=502, content_type="application/json", body='{"message":"Internal Server Error"}'))
    notes.get_by_role("button", name="Save comment").click()
    expect(notes.get_by_role("alert")).to_have_text(NETWORK_ERROR)
    expect(page.get_by_text("Something went wrong")).to_have_count(0)


HF_REASON = "Heart failure unspecified (I50.9) – Moderate reduction of ejection fraction"
SMOKER_REASON = "Smoker (77176002) – Current smoker"


def test_the_suggested_asa_with_its_reasons_on_both_tabs(page: Page, as_user, patient_link):
    hq = as_user("Alex Author").published_hq("asa-hq.json", f"E2E J5e {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"J5e {unique()}")
    patient = patient_link(episode)
    patient.answer("About you", {"q_smokes": "o_yes"})
    patient.answer("Your heart", {"q_heart_failure": "o_yes"})
    patient.submit()

    base = sign_in(page.context, "Cara Clinician")
    path = f"{base}/episodes/{episode['id']}"
    page.goto(path)
    line = page.get_by_test_id("suggested-asa-line")
    expect(line).to_contain_text(f"Suggested ASA III: {HF_REASON}")
    expect(page.get_by_test_id("hq-set-Anaesthetic plan")).to_be_visible()  # shown from ASA III (S39 answers 2)

    page.goto(f"{path}/poa")
    asa = page.get_by_role("region", name="Pre operative assessment ASA grades")
    for tab in ("Validated summary", "Patient answers"):
        page.get_by_role("tab", name=tab).click()
        suggested = asa.get_by_test_id("suggested-asa")
        expect(suggested).to_contain_text("Suggested ASA III")
        expect(suggested).to_contain_text(HF_REASON)
        expect(suggested).not_to_contain_text(SMOKER_REASON)  # only the top grade's disclosures
    expect(asa.get_by_label("Nurse ASA grade")).to_have_count(0)  # the grades are on the validated tab only
    page.get_by_role("tab", name="Validated summary").click()
    expect(asa.get_by_label("Nurse ASA grade")).to_be_visible()

    # The reason is a link to its question on Validate.
    asa.get_by_role("link", name=HF_REASON).click()
    expect(page).to_have_url(re.compile(rf"/sets/{hq['chapters']['Your heart']}\?q=q_heart_failure$"))
    page.go_back()

    # Cara corrects heart failure to No: the validated suggestion drops to ASA II, the patient's doesn't.
    cara.validate(episode, "About you", {"q_smokes": "o_yes"})
    cara.validate(episode, "Your heart", {"q_heart_failure": "o_no"})
    page.reload()
    page.get_by_role("tab", name="Validated summary").click()
    expect(asa.get_by_test_id("suggested-asa")).to_contain_text("Suggested ASA II")
    expect(asa.get_by_test_id("suggested-asa")).to_contain_text(SMOKER_REASON)
    page.get_by_role("tab", name="Patient answers").click()
    expect(asa.get_by_test_id("suggested-asa")).to_contain_text("Suggested ASA III")
    expect(asa.get_by_test_id("suggested-asa")).to_contain_text(HF_REASON)

    page.goto(path)
    expect(line).to_contain_text(f"Suggested ASA II: {SMOKER_REASON}")
    expect(page.get_by_test_id("hq-set-Your heart")).to_be_visible()
    expect(page.get_by_test_id("hq-set-Anaesthetic plan")).to_have_count(0)

    # Smoking corrected too: no answer gives a grade, and the page says so plainly.
    cara.validate(episode, "About you", {"q_smokes": "o_no"})
    page.reload()
    expect(line).to_contain_text("No suggested ASA")
    page.goto(f"{path}/poa")
    expect(asa.get_by_test_id("suggested-asa")).to_have_text("No suggested ASA: no answer gives an ASA grade.")
