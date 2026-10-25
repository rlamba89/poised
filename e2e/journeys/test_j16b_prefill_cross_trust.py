"""J16b Pre-fill from another trust (S26; changed in S27, Rahul 24 Oct: no trust setting and no
question first). A patient who sent Trust B's Demo pre-op HQ gets a Trust A episode on the Demo
follow-up HQ. On their phone, Trust B's answers are already filled in: each set opens on the card,
which names Hospital B1, and adds Start fresh. Check each answer walks Your health with the markers
as for their own trust's (J16); Start fresh empties About you, and it stays empty after a reload.
The clinician's copy says the answers came from another trust's hospital, with none of its ids.

On the patient's behalf (S28): a patient with a Trust B history who never opens Trust A's link.
Cara fills in for them; their Trust A record joins their account on the four fields (date of
birth, email, first name, surname), so the card names Hospital B1 and Start fresh works. Trust A's
log names Cara for the join; Trust B's names only "Staff at Trust A". With one detail different,
nothing comes from Trust B."""

from datetime import datetime

from playwright.sync_api import BrowserContext, Page, expect

import csv
import io

from support import invite_path, sign_in, unique
from test_j16_prefill import published, with_sets


def trust_b_then_trust_a(as_user, patient_link, answers: dict) -> tuple[dict, dict, object, datetime]:
    """Trust B's Demo pre-op HQ answered (answers by set) and sent; then Trust A's follow-up HQ for
    the same person (the same name, date of birth and email: a record of Trust A's own, which joins
    the same account when they prove their date of birth). Returns both episodes, Cara and when it was sent."""
    last_name = f"J16b {unique()}"
    bea = as_user("Bea Author", "Hospital B1")
    first = with_sets(bea, bea.episode(published(bea, "Demo pre-op HQ"), last_name))
    p = patient_link(first)
    for set_name, values in answers.items():
        p.answer(set_name, values)
    p.submit()
    sent_on = datetime.now()
    cara = as_user("Cara Clinician")
    second = with_sets(cara, cara.episode(published(cara, "Demo follow-up HQ"), last_name, email=first["email"]))
    return first, second, cara, sent_on


def open_as_patient(phone: BrowserContext, email: str) -> Page:
    patient = phone.new_page()
    patient.goto(invite_path(email))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    return patient


def test_another_trusts_answers_are_filled_in_without_asking_and_one_set_starts_fresh(phone: BrowserContext, as_user, patient_link):
    first, second, cara, sent_on = trust_b_then_trust_a(as_user, patient_link, {
        "About you": {"smoker": "ex", "alcohol_units": 6, "height": 170, "weight": 82},
        "Your health": {"q_diabetes": "o_no", "snoring": "yes", "pregnant": "no", "ponv_history": "no",
                        "anaesthetic_problems": "no", "family_anaesthetic_problems": "no"},
    })
    patient = open_as_patient(phone, first["email"])

    # No question first: Trust B's answers are in, waiting to be checked.
    you = patient.get_by_test_id("set-About you")
    yours = patient.get_by_test_id("set-Your health")
    expect(you.get_by_text("Please check")).to_be_visible()
    expect(yours.get_by_text("Please check")).to_be_visible()
    expect(patient.get_by_text("Use your earlier answers?")).to_have_count(0)
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")

    # Your health: the card names Hospital B1; Check each answer, with the markers.
    yours.get_by_role("button", name="Check").click()
    card = patient.get_by_test_id("prefill-card")
    day = f"{sent_on.day} {sent_on:%b %Y}"  # en-GB, make dev's default locale
    expect(card).to_contain_text(f"from your questionnaire at Hospital B1 on {day}. Please check them.")
    expect(card.get_by_role("button", name="Start fresh")).to_be_visible()
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
    card.get_by_role("button", name="Check each answer").click()
    expect(patient.get_by_role("radiogroup", name="Do you have diabetes?").get_by_role("radio", name="No", exact=True)).to_be_checked()
    expect(patient.get_by_role("radiogroup", name="Do you snore loudly?").get_by_role("radio", name="Yes", exact=True)).to_be_checked()
    expect(patient.get_by_test_id("prefill-marker").first).to_have_text(f"From your answers on {day}")
    patient.get_by_role("button", name="All sections").click()

    # About you: Start fresh asks first, then the set opens empty, with no card and no markers.
    you.get_by_role("button", name="Check").click()
    card = patient.get_by_test_id("prefill-card")
    card.get_by_role("button", name="Start fresh").click()
    confirm = patient.get_by_test_id("start-fresh-confirm")
    expect(confirm).to_contain_text("This can't be undone.")
    confirm.get_by_role("button", name="Keep the answers").click()  # backs out
    expect(card.get_by_role("button", name="Check each answer")).to_be_visible()
    card.get_by_role("button", name="Start fresh").click()
    with patient.expect_response(lambda r: r.request.method == "POST" and "/prefill/" in r.url):
        patient.get_by_test_id("start-fresh-confirm").get_by_role("button", name="Yes, start fresh").click()
    expect(patient.get_by_test_id("prefill-card")).to_have_count(0)
    expect(patient.get_by_role("radiogroup", name="Do you smoke?")).to_be_visible()
    expect(patient.get_by_role("radiogroup", name="Do you smoke?").get_by_role("radio", name="I used to smoke")).not_to_be_checked()
    expect(patient.get_by_test_id("prefill-marker")).to_have_count(0)

    # Coming back: About you is still empty (not filled again), Your health keeps its answers.
    patient.reload()
    expect(patient.get_by_test_id("set-About you").get_by_role("button", name="Start")).to_be_visible()
    expect(patient.get_by_test_id("set-About you").get_by_text("Please check")).to_have_count(0)

    # As the clinician will see it: another trust's hospital, how it matched, and none of its ids;
    # About you records Start fresh with its answers gone.
    rows = cara.call("GET", cara.h(f"/episodes/{second['id']}/hq"))["answers"]
    mine = {r["chapterId"]: r for r in rows if r["actor"] == "patient"}
    health = mine[second["hq"]["chapters"]["Your health"]]
    snoring = health["prefill"]["snoring"]
    assert snoring["otherTrust"] is True and snoring["hospital"] == "Hospital B1" and snoring["questionnaire"] == "Demo pre-op HQ"
    # Snoring is a platform library key, the only kind of key that matches across trusts ("by
    # link"); diabetes is matched by its wording (S26, after verification 1).
    assert snoring["matchedBy"] == "key" and "fromForm" not in snoring
    assert health["prefill"]["fu_diabetes"]["matchedBy"] == "wording"
    about = mine[second["hq"]["chapters"]["About you"]]
    assert about["prefillChoice"] == "start_fresh" and about["data"] == {}


def test_a_refused_start_fresh_says_why_and_keeps_the_answers(phone: BrowserContext, as_user, patient_link):
    """If Start fresh is refused (e.g. the set changed in another tab), the reason shows and nothing
    is cleared silently (healthcare: nothing may break silently)."""
    first, _, _, _ = trust_b_then_trust_a(as_user, patient_link, {"Your health": {"snoring": "yes"}})
    patient = open_as_patient(phone, first["email"])
    patient.route("**/prefill/*", lambda route: route.fulfill(
        status=409, content_type="application/json",
        body='{"error":"The answers in this section have changed since you opened it, so they weren\'t cleared. Please reload the page."}'))
    patient.get_by_test_id("set-Your health").get_by_role("button", name="Check").click()
    patient.get_by_test_id("prefill-card").get_by_role("button", name="Start fresh").click()
    patient.get_by_test_id("start-fresh-confirm").get_by_role("button", name="Yes, start fresh").click()
    expect(patient.get_by_text("so they weren't cleared. Please reload the page.")).to_be_visible()
    expect(patient.get_by_test_id("start-fresh-confirm")).to_have_count(0)
    expect(patient.get_by_test_id("prefill-card").get_by_role("button", name="Check each answer")).to_be_visible()


def test_staff_filling_in_for_a_patient_who_never_opened_the_link_get_trust_bs_answers(page: Page, as_user, patient_link):
    first, second, cara, _ = trust_b_then_trust_a(as_user, patient_link, {
        "About you": {"smoker": "ex", "alcohol_units": 6, "height": 170, "weight": 82},
        "Your health": {"q_diabetes": "o_no", "snoring": "yes"},
    })
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{second['id']}")
    page.get_by_role("link", name="Fill in for the patient").click()
    expect(page.get_by_test_id("on-behalf-banner")).to_be_visible()

    # Trust B's answers are in: the card names Hospital B1, and Start fresh clears the set.
    yours = page.get_by_test_id("set-Your health")
    expect(yours.get_by_text("Please check")).to_be_visible()
    yours.get_by_role("button", name="Check").click()
    card = page.get_by_test_id("prefill-card")
    expect(card).to_contain_text("at Hospital B1")
    card.get_by_role("button", name="Start fresh").click()
    with page.expect_response(lambda r: r.request.method == "POST" and "/prefill/" in r.url):
        page.get_by_test_id("start-fresh-confirm").get_by_role("button", name="Yes, start fresh").click()
    expect(page.get_by_test_id("prefill-card")).to_have_count(0)
    expect(page.get_by_role("radiogroup", name="Do you snore loudly?").get_by_role("radio", name="Yes", exact=True)).not_to_be_checked()

    # Trust A's record joined the account: Trust A's log names Cara for the join, and the record
    # shows no contact only Trust B holds (it had the same email already, so none is added).
    ada = as_user("Ada Admin")
    trust_a = list(csv.DictReader(io.StringIO(ada.request.get(f"/api{ada.o('/audit')}?format=csv&patient={second['patientId']}").text())))
    joins = [r for r in trust_a if r["Action"] == "Patient's record joined their sign-in account"]
    assert len(joins) == 1 and joins[0]["Who"] == "Cara Clinician", joins

    # Trust B's log: the read is by "Staff at Trust A", with no name or email, in the CSV and the Who filter.
    max_ = as_user("Max Multi", "Hospital B1")
    trust_b = list(csv.DictReader(io.StringIO(max_.request.get(f"/api{max_.o('/audit')}?format=csv&patient={first['patientId']}").text())))
    reads = [r for r in trust_b if r["Action"] == "Patient's answers read to pre-fill their HQ at another trust"]
    assert len(reads) == 1, reads
    assert reads[0]["Who"] == "Staff at Trust A" and reads[0]["Email"] == "", reads[0]
    assert not any("Cara" in r["Who"] or "cara" in r["Email"] for r in trust_b), trust_b
    users = max_.call("GET", max_.o("/audit"))["users"]
    assert any(u["name"] == "Staff at Trust A" and u["email"] == "" and u["id"].startswith("trust:") for u in users), users

    # Opened again: Trust B is read no more (one row per form).
    page.reload()
    expect(page.get_by_test_id("on-behalf-banner")).to_be_visible()
    trust_b = list(csv.DictReader(io.StringIO(max_.request.get(f"/api{max_.o('/audit')}?format=csv&patient={first['patientId']}").text())))
    assert len([r for r in trust_b if r["Action"].startswith("Patient's answers read")]) == 1


def test_on_behalf_one_detail_different_gets_nothing_from_trust_b(page: Page, as_user, patient_link):
    last_name = f"J16b {unique()}"
    bea = as_user("Bea Author", "Hospital B1")
    first = with_sets(bea, bea.episode(published(bea, "Demo pre-op HQ"), last_name))
    p = patient_link(first)
    p.answer("Your health", {"q_diabetes": "o_no", "snoring": "yes"})
    p.submit()
    cara = as_user("Cara Clinician")
    # The same person to the eye, but another surname spelling at Trust A.
    second = with_sets(cara, cara.episode(published(cara, "Demo follow-up HQ"), last_name + "e", email=first["email"]))
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{second['id']}/patient-form")
    expect(page.get_by_test_id("on-behalf-banner")).to_be_visible()
    expect(page.get_by_test_id("set-Your health").get_by_role("button", name="Start")).to_be_visible()
    expect(page.get_by_text("Please check")).to_have_count(0)
    ada = as_user("Ada Admin")
    trust_a = ada.request.get(f"/api{ada.o('/audit')}?format=csv&patient={second['patientId']}").text()
    assert "joined their sign-in account" not in trust_a


NETWORK_ERROR = "We couldn't reach the server. Check your connection and try again."


def test_start_fresh_whose_reload_fails_says_the_answers_were_cleared(phone: BrowserContext, as_user, patient_link):
    """Start fresh worked, but the HQ didn't load again (S28; S27 verification 3): the card with the
    old answers mustn't stay up as if nothing happened. The page says the answers were cleared, with
    Reload, which then opens the set empty, as a Start fresh that loads does. The card already says
    the page couldn't load, so no network banner over it (S29, S30)."""
    first, _, _, _ = trust_b_then_trust_a(as_user, patient_link, {"Your health": {"snoring": "yes"}})
    patient = open_as_patient(phone, first["email"])
    patient.get_by_test_id("set-Your health").get_by_role("button", name="Check").click()
    failing = {"on": False}

    def reload_fails(route):
        if failing["on"] and route.request.method == "GET":
            route.abort("internetdisconnected")
        else:
            if route.request.method == "POST" and "/prefill/" in route.request.url:
                failing["on"] = True
            route.continue_()

    patient.route("**/api/p/episodes/**", reload_fails)
    patient.get_by_test_id("prefill-card").get_by_role("button", name="Start fresh").click()
    patient.get_by_test_id("start-fresh-confirm").get_by_role("button", name="Yes, start fresh").click()
    failed = patient.get_by_test_id("fresh-reload-failed")
    expect(failed).to_contain_text("The answers in this section were cleared")
    expect(patient.get_by_test_id("prefill-card")).to_have_count(0)
    expect(patient.get_by_text(NETWORK_ERROR)).to_have_count(0)
    failing["on"] = False
    failed.get_by_role("button", name="Reload").click()
    expect(patient.get_by_test_id("fresh-reload-failed")).to_have_count(0)
    # As after a Start fresh that loads: the set open, empty, with no card.
    expect(patient.get_by_test_id("prefill-card")).to_have_count(0)
    expect(patient.get_by_role("radiogroup", name="Do you snore loudly?").get_by_role("radio", name="Yes", exact=True)).not_to_be_checked()


def test_a_reload_refused_for_another_reason_still_says_why_over_the_cleared_card(page: Page, as_user, patient_link):
    """The cleared card hides only the network message (S30): Cara, filling in for the patient, starts
    a set fresh while reads fail, and the episode is cancelled before she presses Reload. The
    refusal says it was cancelled, above the card."""
    _, second, cara, _ = trust_b_then_trust_a(as_user, patient_link, {"Your health": {"snoring": "yes"}})
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{second['id']}/patient-form")
    page.get_by_test_id("set-Your health").get_by_role("button", name="Check").click()
    failing = {"on": False}

    def reload_fails(route):
        if failing["on"] and route.request.method == "GET":
            route.abort("internetdisconnected")
        else:
            if route.request.method == "POST" and "/prefill/" in route.request.url:
                failing["on"] = True
            route.continue_()

    page.route("**/api/o/**/episodes/**", reload_fails)
    page.get_by_test_id("prefill-card").get_by_role("button", name="Start fresh").click()
    page.get_by_test_id("start-fresh-confirm").get_by_role("button", name="Yes, start fresh").click()
    failed = page.get_by_test_id("fresh-reload-failed")
    expect(failed).to_contain_text("The answers in this section were cleared")
    expect(page.get_by_text(NETWORK_ERROR)).to_have_count(0)

    # Cancelled meanwhile (in another tab, say): Reload is refused, and the page says why.
    failing["on"] = False
    revision = cara.call("GET", cara.h(f"/episodes/{second['id']}"))["episode"]["revision"]
    cara.call("POST", cara.h(f"/episodes/{second['id']}/cancel"), {"reason": "patient_withdrew", "revision": revision})
    failed.get_by_role("button", name="Reload").click()
    expect(page.get_by_role("alert").filter(has_text="This questionnaire was cancelled by the hospital")).to_be_visible()
    expect(failed).to_be_visible()
