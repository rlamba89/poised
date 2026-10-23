"""J16b Pre-fill from another trust (S26; §6 point 7, ORG-04's crossTrustPrefill): with both trusts'
"Offer answers patients gave other trusts" on, a patient who sent Trust B's Demo pre-op HQ gets a
Trust A episode on the Demo follow-up HQ. On their phone, before the first Question Set, they're asked
"Use the answers you gave Hospital B1 in <month>?"; they say Yes, and Trust B's answers are filled
in, with the opening card and markers as for their own trust's (J16). The clinician's copy says the
answers came from another trust's hospital. The settings are put back afterwards."""

from datetime import datetime

import pytest
from playwright.sync_api import BrowserContext, expect

from support import choose, invite_path, unique
from test_j16_prefill import published, with_sets


@pytest.fixture
def both_trusts_offer(as_user):
    """Both trusts' crossTrustPrefill on (Ada for Trust A, Max for Trust B), put back afterwards."""
    admins = [as_user("Ada Admin"), as_user("Max Multi", "Hospital B1")]
    before = [a.call("GET", a.o("/settings"))["trust"].get("crossTrustPrefill", False) for a in admins]
    for a in admins:
        a.call("PATCH", a.o("/settings"), {"hospitalId": None, "values": {"crossTrustPrefill": True}})
    yield
    for a, was in zip(admins, before):
        a.call("PATCH", a.o("/settings"), {"hospitalId": None, "values": {"crossTrustPrefill": was}})


def test_a_patient_agrees_to_use_another_trusts_answers_on_a_phone(phone: BrowserContext, as_user, patient_link, both_trusts_offer):
    last_name = f"J16b {unique()}"

    # Trust B: the Demo pre-op HQ, answered and sent.
    bea = as_user("Bea Author", "Hospital B1")
    first = with_sets(bea, bea.episode(published(bea, "Demo pre-op HQ"), last_name))
    p = patient_link(first)
    p.answer("About you", {"smoker": "ex", "alcohol_units": 6, "height": 170, "weight": 82})
    p.answer("Your health", {"q_diabetes": "o_no", "snoring": "yes", "pregnant": "no", "ponv_history": "no",
                             "anaesthetic_problems": "no", "family_anaesthetic_problems": "no"})
    p.submit()
    sent_on = datetime.now()

    # Trust A: the follow-up HQ for the same person (the same name, date of birth and email: a record
    # of Trust A's own, which joins the same account when they prove their date of birth).
    cara = as_user("Cara Clinician")
    second = with_sets(cara, cara.episode(published(cara, "Demo follow-up HQ"), last_name, email=first["email"]))

    patient = phone.new_page()
    patient.goto(invite_path(first["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()

    # Asked first, before any Question Set.
    ask = patient.get_by_test_id("prefill-consent")
    expect(ask).to_contain_text(f"Use the answers you gave Hospital B1 in {sent_on:%B %Y}?")  # en-GB, make dev's default locale
    expect(patient.get_by_test_id("set-About you")).to_have_count(0)
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
    ask.get_by_role("button", name="Yes, use them").click()

    # Trust B's answers are in, waiting to be checked.
    you = patient.get_by_test_id("set-About you")
    yours = patient.get_by_test_id("set-Your health")
    expect(you.get_by_text("Please check")).to_be_visible()
    expect(yours.get_by_text("Please check")).to_be_visible()
    expect(patient.get_by_test_id("prefill-consent")).to_have_count(0)
    yours.get_by_role("button", name="Check").click()
    card = patient.get_by_test_id("prefill-card")
    day = f"{sent_on.day} {sent_on:%b %Y}"
    expect(card).to_contain_text(f"from your questionnaire on {day}")
    card.get_by_role("button", name="Check each answer").click()
    expect(patient.get_by_role("radiogroup", name="Do you have diabetes?").get_by_role("radio", name="No", exact=True)).to_be_checked()
    expect(patient.get_by_role("radiogroup", name="Do you snore loudly?").get_by_role("radio", name="Yes", exact=True)).to_be_checked()
    expect(patient.get_by_test_id("prefill-marker").first).to_have_text(f"From your answers on {day}")

    # Coming back: not asked again.
    patient.reload()
    expect(patient.get_by_test_id("set-Your health")).to_be_visible()
    expect(patient.get_by_test_id("prefill-consent")).to_have_count(0)

    # Where each answer came from, as the clinician will see it: another trust's hospital.
    rows = cara.call("GET", cara.h(f"/episodes/{second['id']}/hq"))["answers"]
    health = next(r for r in rows if r["actor"] == "patient" and r["chapterId"] == second["hq"]["chapters"]["Your health"])
    snoring = health["prefill"]["snoring"]
    assert snoring["otherTrust"] is True and snoring["hospital"] == "Hospital B1" and snoring["questionnaire"] == "Demo pre-op HQ"
    # Another trust's answers keep none of its ids (S26 verification 1). Snoring is a platform
    # library key, the only kind of key that matches across trusts ("by link"); diabetes is matched
    # by its wording (Rahul, after verification 1).
    assert snoring["matchedBy"] == "key" and "fromForm" not in snoring
    assert health["prefill"]["fu_diabetes"]["matchedBy"] == "wording"


def test_no_thanks_is_remembered(phone: BrowserContext, as_user, patient_link, both_trusts_offer):
    """No, thanks: nothing from Trust B, and not asked again on the next visit."""
    last_name = f"J16b {unique()}"
    bea = as_user("Bea Author", "Hospital B1")
    first = with_sets(bea, bea.episode(published(bea, "Demo pre-op HQ"), last_name))
    p = patient_link(first)
    p.answer("Your health", {"snoring": "yes"})
    p.submit()
    cara = as_user("Cara Clinician")
    cara.episode(published(cara, "Demo follow-up HQ"), last_name, email=first["email"])

    patient = phone.new_page()
    patient.goto(invite_path(first["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_test_id("prefill-consent").get_by_role("button", name="No, thanks").click()
    yours = patient.get_by_test_id("set-Your health")
    expect(yours.get_by_role("button", name="Start")).to_be_visible()  # nothing filled in
    patient.reload()
    expect(yours.get_by_role("button", name="Start")).to_be_visible()
    expect(patient.get_by_test_id("prefill-consent")).to_have_count(0)
    yours.get_by_role("button", name="Start").click()
    expect(patient.get_by_test_id("prefill-card")).to_have_count(0)
    expect(patient.get_by_role("radiogroup", name="Do you snore loudly?").get_by_role("radio", name="Yes", exact=True)).not_to_be_checked()
    choose(patient, "Do you snore loudly?", "No")


def test_a_refused_answer_keeps_its_message(phone: BrowserContext, as_user, patient_link, both_trusts_offer):
    """If recording the answer fails, the question comes back with the reason showing, not silently
    (S26 verification 1, finding 4)."""
    last_name = f"J16b {unique()}"
    bea = as_user("Bea Author", "Hospital B1")
    first = with_sets(bea, bea.episode(published(bea, "Demo pre-op HQ"), last_name))
    p = patient_link(first)
    p.answer("Your health", {"snoring": "yes"})
    p.submit()
    cara = as_user("Cara Clinician")
    cara.episode(published(cara, "Demo follow-up HQ"), last_name, email=first["email"])

    patient = phone.new_page()
    patient.route("**/prefill-consent", lambda route: route.fulfill(
        status=500, content_type="application/json", body='{"error":"Something went wrong. Please try again."}'))
    patient.goto(invite_path(first["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    with patient.expect_response(lambda r: r.request.method == "GET" and "/api/p/episodes/" in r.url):  # the HQ loads again
        patient.get_by_test_id("prefill-consent").get_by_role("button", name="Yes, use them").click()
    patient.wait_for_load_state("networkidle")
    expect(patient.get_by_text("Something went wrong. Please try again.")).to_be_visible()
    expect(patient.get_by_test_id("prefill-consent")).to_be_visible()  # still to answer
