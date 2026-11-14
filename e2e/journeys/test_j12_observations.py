"""J12: observations (plan C7, session S46; OBS-01–04), and STOP-Bang as an assessment (S47; §9).

  - Cara records height, weight and two blood pressures on an episode. A negative weight is refused
    with Go's reason; a heart rate of 220 warns ("outside the usual range"), then saves with Save
    anyway. The episode page shows the hospital BMI.
  - She corrects the weight, with a reason: the old reading stays, struck through, and the hospital
    BMI follows the corrected weight. A blood pressure marked entered in error stays in the history.
  - The POA Summary shows the hospital BMI beside the patient's own (their HQ's BMI question).
  - S47: on a sent HQ that asked S, T, O and P, Cara records height and weight and adds STOP-Bang:
    it opens with 7 of 8 items filled, each with its source, the neck missing. She answers it and
    confirms; the score and band show on the POA Summary beside the hospital BMI. A change needs a
    new Confirm. An HQ holding the ready-made STOP-Bang opens it with every item filled, waiting.
  - S47 answers 5 and 6: the pre-fill uses the clinician's copy of a set once they've started it,
    and a confirmed STOP-Bang says when a value it used has changed since (a corrected weight),
    while its score stays as confirmed.
  - S48: a patient sends, on their phone, an HQ holding the ready-made DASI, PRISMA-7 and Apfel's
    questions. DASI opens complete (12 of 12) with its score, band and METs; Cara confirms it, and
    its METs show on the POA Summary. PRISMA-7 opens 7 of 7; Apfel 3 of 4, waiting only for the
    post-op opioids.
  - S49: on a sent HQ holding MUST's, Morse's and VTE's patient questions, MUST opens 3 of 4: the
    hospital BMI and weight from Cara's readings, the weight lost from the HQ. A corrected weight
    changes its BMI, weight and score before Confirm; the POA Summary shows the result.
"""

from playwright.sync_api import BrowserContext, Page, expect

from support import choose, invite_path, pick, sign_in, unique


def add(page: Page, kind: str, values: dict[str, str]) -> None:
    """Adds a reading on the episode page: the kind, then each box by its label ("Weight (kg)")."""
    form = page.get_by_test_id("observation-add")
    pick(page, "Add a reading", kind)
    for label, value in values.items():
        form.get_by_label(label).fill(value)
    form.get_by_role("button", name="Save reading").click()


def test_a_clinician_records_corrects_and_sees_the_hospital_bmi(page: Page, as_user, patient_link):
    hq = as_user("Alex Author").published_hq("hidden-bmi-hq.json", f"E2E J12 {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 {unique()}")
    # The patient's own BMI, from their HQ: 170 cm, 82 kg → 28.4.
    patient_link(episode).answer("Weight", {"q_e2e_bmi": {"height": 170, "weight": 82}})

    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    obs = page.get_by_test_id("observations")
    expect(obs.get_by_text("No readings yet.")).to_be_visible()
    expect(page.get_by_test_id("hospital-bmi")).to_contain_text("Not measured yet")

    # A negative weight is refused with Go's reason; nothing is saved.
    add(page, "Weight", {"Weight (kg)": "-3"})
    expect(obs.get_by_role("alert")).to_contain_text("Weight must be from")
    expect(obs.get_by_text("No readings yet.")).to_be_visible()
    # Measured at says it's the time of Save until it's changed (test report 2).
    expect(obs.get_by_text("The time you save it, unless you change it.")).to_be_visible()

    # Height and weight: the hospital BMI (180 cm, 100 kg → 30.9).
    add(page, "Height", {"Height (cm)": "180"})
    expect(obs.get_by_test_id("observation")).to_have_count(1)
    add(page, "Weight", {"Weight (kg)": "100"})
    expect(page.get_by_test_id("hospital-bmi")).to_contain_text("30.9")

    # Two blood pressures.
    add(page, "Blood pressure", {"Systolic (mmHg)": "128", "Diastolic (mmHg)": "82"})
    add(page, "Blood pressure", {"Systolic (mmHg)": "132", "Diastolic (mmHg)": "84"})
    expect(obs.get_by_test_id("observation").filter(has_text="Blood pressure")).to_have_count(2)
    expect(obs.get_by_text("Blood pressure: 128/82 mmHg")).to_be_visible()

    # A diastolic above the systolic is refused at once, with no "usual range" warning first
    # (test report 1, F-2).
    add(page, "Blood pressure", {"Systolic (mmHg)": "80", "Diastolic (mmHg)": "120"})
    expect(obs.get_by_role("alert")).to_have_text("The diastolic must be lower than the systolic.")
    expect(page.get_by_test_id("observation-warning")).to_have_count(0)

    # An ECG note (test report 1, F-1: the note kinds couldn't be saved).
    pick(page, "Add a reading", "ECG")
    page.get_by_test_id("observation-add").get_by_label("ECG note").fill("Made-up: sinus rhythm")
    page.get_by_test_id("observation-add").get_by_role("button", name="Save reading").click()
    expect(obs.get_by_text("ECG: Made-up: sinus rhythm")).to_be_visible()

    # An implausible heart rate warns; Save anyway saves it.
    add(page, "Heart rate", {"Heart rate (beats/min)": "220"})
    expect(page.get_by_test_id("observation-warning")).to_contain_text("outside the usual range")
    expect(obs.get_by_test_id("observation").filter(has_text="Heart rate")).to_have_count(0)
    page.get_by_role("button", name="Save anyway").click()
    expect(obs.get_by_text("Heart rate: 220 beats/min")).to_be_visible()

    # Correct the weight, with a reason: the old one stays struck through, the BMI follows.
    page.get_by_role("button", name="Correct Weight 100.0 kg").click()
    fix = page.get_by_test_id("observation-correct")
    fix.get_by_label("Weight (kg)").fill("90")
    fix.get_by_role("button", name="Save correction").click()
    expect(fix.get_by_role("alert")).to_contain_text("Say why the reading is corrected.")
    fix.get_by_label("Why is it corrected?").fill("Made-up: scale not zeroed")
    fix.get_by_role("button", name="Save correction").click()
    expect(page.get_by_test_id("hospital-bmi")).to_contain_text("27.8")  # 90 / 1.80²
    old = obs.get_by_test_id("observation").filter(has_text="100.0 kg")
    expect(old).to_have_attribute("data-counts", "false")
    expect(old).to_contain_text("Corrected to 90.0 kg")
    expect(obs.get_by_test_id("observation").filter(has_text="90.0 kg").first).to_contain_text("Made-up: scale not zeroed")
    expect(old.get_by_role("alert")).to_have_count(0)  # the "Say why…" goes once it's saved (test report 2, F-4)

    # A correction typed outside the usual range asks first, like a new reading (verification, F-1).
    fix2 = "Correct Weight 90.0 kg"
    page.get_by_role("button", name=fix2).click()
    fixing = page.get_by_test_id("observation-correct")
    fixing.get_by_label("Weight (kg)").fill("380")
    fixing.get_by_label("Why is it corrected?").fill("Made-up: a typing test")
    fixing.get_by_role("button", name="Save correction").click()
    warned = fixing.get_by_test_id("observation-warning")
    expect(warned).to_contain_text("Weight 380 kg is outside the usual range (35 to 200).")
    expect(page.get_by_test_id("hospital-bmi")).to_contain_text("27.8")  # nothing saved yet
    warned.get_by_role("button", name="Change it").click()
    fixing.get_by_label("Weight (kg)").fill("381")
    fixing.get_by_role("button", name="Save correction").click()
    warned.get_by_role("button", name="Save anyway").click()
    expect(page.get_by_test_id("hospital-bmi")).to_contain_text("117.6")
    # Marking that correction entered in error says the old value won't come back (verification, F-2).
    page.get_by_role("button", name="Mark Weight 381.0 kg entered in error").click()
    wrong_fix = page.get_by_test_id("observation-error-form")
    expect(wrong_fix).to_contain_text("The value it corrected won't count again. To put a value back, use Correct.")
    wrong_fix.get_by_role("button", name="Back").click()
    # Correcting it again is the way back.
    page.get_by_role("button", name="Correct Weight 381.0 kg").click()
    fixing.get_by_label("Weight (kg)").fill("90")
    fixing.get_by_label("Why is it corrected?").fill("Made-up: back to the right value")
    fixing.get_by_role("button", name="Save correction").click()
    expect(page.get_by_test_id("hospital-bmi")).to_contain_text("27.8")

    # Back clears a refusal too.
    page.get_by_role("button", name="Correct Heart rate 220 beats/min").click()
    page.get_by_test_id("observation-correct").get_by_role("button", name="Save correction").click()
    expect(obs.get_by_role("alert")).to_have_count(1)
    page.get_by_test_id("observation-correct").get_by_role("button", name="Back").click()
    expect(obs.get_by_role("alert")).to_have_count(0)

    # A blood pressure entered in error stays in the history, with who and why.
    page.get_by_role("button", name="Mark Blood pressure 128/82 mmHg entered in error").click()
    wrong = page.get_by_test_id("observation-error-form")
    wrong.get_by_role("button", name="Mark entered in error").click()
    expect(wrong.get_by_role("alert")).to_have_text("Say why the reading was entered in error.")
    wrong.get_by_label("Why was it entered in error?").fill("Made-up: another patient's")
    wrong.get_by_role("button", name="Mark entered in error").click()
    marked = obs.get_by_test_id("observation").filter(has_text="128/82")
    expect(marked).to_have_attribute("data-counts", "false")
    expect(marked.get_by_test_id("observation-error")).to_contain_text("marked by Cara Clinician")
    expect(obs.get_by_role("alert")).to_have_count(0)

    # Reload keeps it all.
    page.reload()
    expect(obs.get_by_test_id("observation")).to_have_count(9)  # height, weight and its three corrections, two blood pressures, an ECG, a heart rate
    expect(page.get_by_test_id("hospital-bmi")).to_contain_text("27.8")

    # The POA Summary: the hospital BMI beside the patient's own.
    page.goto(f"{base}/episodes/{episode['id']}/poa")
    expect(page.get_by_test_id("poa-detail-Patient reported BMI")).to_contain_text("28.4")
    expect(page.get_by_test_id("poa-detail-Hospital BMI")).to_contain_text("27.8")
    # Switching tabs neither reloads the readings nor writes an audit row (verification, F-4).
    loads: list[str] = []
    page.on("request", lambda q: loads.append(q.url) if q.url.endswith("/observations") else None)
    page.get_by_role("tab", name="Patient answers").click()
    expect(page.get_by_test_id("poa-detail-Hospital BMI")).to_contain_text("27.8")
    page.get_by_role("tab", name="Validated summary").click()
    expect(page.get_by_test_id("poa-detail-Hospital BMI")).to_contain_text("27.8")
    assert loads == []


def test_a_stale_tab_is_told_and_shown_the_list_as_it_is(page: Page, as_user):
    """Test report 1: correcting a reading another tab corrected meanwhile says so, and the list
    reloads (the reading shows as corrected, with no buttons)."""
    hq = as_user("Alex Author").published_hq("hidden-bmi-hq.json", f"E2E J12 {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    add(page, "Weight", {"Weight (kg)": "80"})
    other = page.context.new_page()
    other.goto(f"{base}/episodes/{episode['id']}")
    other.get_by_role("button", name="Correct Weight 80.0 kg").click()
    other.get_by_test_id("observation-correct").get_by_label("Weight (kg)").fill("81")
    other.get_by_test_id("observation-correct").get_by_label("Why is it corrected?").fill("Made-up: first tab")
    other.get_by_test_id("observation-correct").get_by_role("button", name="Save correction").click()
    expect(other.get_by_text("Weight: 81.0 kg")).to_be_visible()

    page.get_by_role("button", name="Correct Weight 80.0 kg").click()
    page.get_by_test_id("observation-correct").get_by_label("Weight (kg)").fill("82")
    page.get_by_test_id("observation-correct").get_by_label("Why is it corrected?").fill("Made-up: second tab")
    page.get_by_test_id("observation-correct").get_by_role("button", name="Save correction").click()
    obs = page.get_by_test_id("observations")
    expect(obs.get_by_role("alert")).to_have_text("This reading has already been corrected or marked entered in error.")
    old = obs.get_by_test_id("observation").filter(has_text="80.0 kg")
    expect(old).to_have_attribute("data-counts", "false")
    expect(old.get_by_role("button", name="Correct Weight 80.0 kg")).to_have_count(0)
    expect(obs.get_by_text("Weight: 81.0 kg")).to_be_visible()


def test_a_long_unbroken_note_wraps(page: Page, as_user):
    """Test report 3: a pasted link or code stays readable on the card."""
    hq = as_user("Alex Author").published_hq("hidden-bmi-hq.json", f"E2E J12 note {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 note {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    add(page, "ECG", {"ECG note": "x" * 400})
    entry = page.get_by_test_id("observations").get_by_test_id("observation")
    expect(entry).to_have_count(1)
    assert entry.evaluate("e => e.scrollWidth <= e.clientWidth")
    text = entry.get_by_text("xxx")
    assert text.evaluate("e => e.scrollWidth <= e.clientWidth")
    # Too long: said so, and what was pasted stays (test report 4: it was cut with no message).
    form = page.get_by_test_id("observation-add")
    form.get_by_label("ECG note").fill("y" * 2200)
    form.get_by_role("button", name="Save reading").click()
    expect(page.get_by_test_id("observations").get_by_role("alert")).to_have_text("The note is too long (2000 characters at most).")
    expect(form.get_by_label("ECG note")).to_have_value("y" * 2200)
    expect(entry).to_have_count(1)


def test_the_poa_summary_says_when_the_readings_did_not_load(page: Page, as_user):
    """Verification 2, F-3: a failed load mustn't look the same as observations being off."""
    hq = as_user("Alex Author").published_hq("hidden-bmi-hq.json", f"E2E J12 load {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 load {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    page.route("**/observations", lambda route: route.fulfill(status=500, json={"error": "Something went wrong."}))
    page.goto(f"{base}/episodes/{episode['id']}/poa")
    expect(page.get_by_test_id("poa-detail-Hospital BMI")).to_contain_text("Couldn't be loaded")


def test_observations_off_never_calls_the_route(page: Page, as_user):
    """Verification 2: with observations off, each page open wrote a refused audit row."""
    hq = as_user("Alex Author").published_hq("hidden-bmi-hq.json", f"E2E J12 off {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 off {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    calls: list[str] = []
    page.on("request", lambda q: calls.append(q.url) if q.url.endswith("/observations") else None)

    def off(route):
        reply = route.fetch()
        route.fulfill(response=reply, json={**reply.json(), "observationsOn": False})

    page.route(f"**/api/**/episodes/{episode['id']}", off)
    page.goto(f"{base}/episodes/{episode['id']}")
    expect(page.get_by_role("heading", name="Tasks")).to_be_visible()
    expect(page.get_by_test_id("observations")).to_have_count(0)
    page.goto(f"{base}/episodes/{episode['id']}/poa")
    expect(page.get_by_test_id("poa-detail-Patient reported BMI")).to_be_visible()
    expect(page.get_by_test_id("poa-detail-Hospital BMI")).to_have_count(0)
    assert calls == []


def test_two_admins_editing_the_ranges(page: Page):
    """Test report 5: a page left open put back the other admin's change, row by row."""
    base = sign_in(page.context, "Ada Admin")
    other = page.context.new_page()
    follow = page.get_by_role("checkbox", name="Use Preopsis's default: Observation ranges")
    save = page.get_by_role("button", name="Save the trust's settings")
    page.goto(f"{base}/settings")
    expect(page.get_by_role("heading", name="Settings")).to_be_visible()
    follow.uncheck()
    page.get_by_role("textbox", name="Heart rate: Usual to").fill("115")
    save.click()
    try:
        expect(page.get_by_text("Trust A's settings were saved.")).to_be_visible()
        page.reload()
        other.goto(f"{base}/settings")
        other.get_by_role("textbox", name="Heart rate: Usual to").fill("108")
        other.get_by_role("button", name="Save the trust's settings").click()
        expect(other.get_by_text("Trust A's settings were saved.")).to_be_visible()

        # The page left open still shows 115, and changes another row.
        page.get_by_role("textbox", name="Weight: Usual to").fill("190")
        save.click()
        expect(page.get_by_text("Someone else changed the observation ranges.")).to_be_visible()
        other.reload()
        expect(other.get_by_role("textbox", name="Heart rate: Usual to")).to_have_value("108")
        expect(other.get_by_role("textbox", name="Weight: Usual to")).to_have_value("200")
    finally:
        # Trust A's ranges back as they were, for the other journeys.
        other.goto(f"{base}/settings")
        back = other.get_by_role("checkbox", name="Use Preopsis's default: Observation ranges")
        back.check()
        other.get_by_role("button", name="Save the trust's settings").click()
        expect(other.get_by_text("Trust A's settings were saved.")).to_be_visible()
        other.close()


def test_saving_the_ranges_unchanged_changes_nothing(page: Page):
    """Verification 2, F-1: stored ranges compared unequal, so every Save sent them, and a page left
    open undid another admin's limit while changing something else."""
    base = sign_in(page.context, "Ada Admin")
    page.goto(f"{base}/settings")
    expect(page.get_by_role("heading", name="Settings")).to_be_visible()
    follow = page.get_by_role("checkbox", name="Use Preopsis's default: Observation ranges")
    save = page.get_by_role("button", name="Save the trust's settings")
    if follow.is_checked():
        follow.uncheck()
    page.get_by_role("textbox", name="Heart rate: Usual to").fill("115")
    save.click()
    try:
        expect(page.get_by_text("Trust A's settings were saved.")).to_be_visible()
        save.click()
        expect(page.get_by_text("Nothing has changed.")).to_be_visible()
    finally:
        # Trust A's ranges back as they were, for the other journeys.
        follow.check()
        save.click()
        expect(page.get_by_text("Trust A's settings were saved.")).to_be_visible()


def test_a_trust_range_below_zero_is_refused(page: Page):
    """Test report 1, F-3: "-1" was quietly saved as 1. The box keeps the minus and Go refuses it."""
    base = sign_in(page.context, "Ada Admin")
    page.goto(f"{base}/settings")
    expect(page.get_by_role("heading", name="Settings")).to_be_visible()
    follow = page.get_by_role("checkbox", name="Use Preopsis's default: Observation ranges")
    if follow.is_checked():
        follow.uncheck()
    box = page.get_by_role("textbox", name="Heart rate: Refused below")
    box.fill("-1")
    expect(box).to_have_value("-1")
    page.get_by_role("button", name="Save the trust's settings").click()
    expect(page.get_by_text("Heart rate: a limit can't be below 0.")).to_be_visible()


def add_stop_bang(page: Page) -> None:
    """Add assessment → STOP-Bang on the episode page."""
    page.get_by_role("combobox", name="Add assessment").click()
    page.get_by_role("option", name="STOP-Bang", exact=True).click()
    expect(page.get_by_test_id("assessment-stopbang")).to_be_visible()


def item(page: Page, letter: str):
    return page.get_by_test_id("assessment-stopbang").get_by_test_id(f"item-{letter}")


def test_stop_bang_opens_7_of_8_filled_and_is_confirmed(page: Page, as_user, patient_link):
    hq = as_user("Alex Author").published_hq("stopbang-items-hq.json", f"E2E J12 STOP-Bang {unique()}")
    # Made up: a man of 66, so his age and sex each score.
    episode = as_user("Cara Clinician").episode(hq, f"J12 SB {unique()}", dateOfBirth="1960-03-01", sex="male")
    patient = patient_link(episode)
    # His own BMI (170 cm, 82 kg → 28.4), and S, T and P Yes, O No.
    patient.answer("Sleep", {"q_e2e_bmi": {"height": 170, "weight": 82}, "stopbang_snoring": "o_yes", "stopbang_tired": "o_yes",
                             "stopbang_observed": "o_no", "stopbang_pressure": "o_yes"})
    patient.submit()

    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    expect(page.get_by_test_id("assessments").get_by_text("No assessments yet.")).to_be_visible()
    # Height and weight: the hospital BMI, 180 cm and 120 kg → 37.0.
    add(page, "Height", {"Height (cm)": "180"})
    add(page, "Weight", {"Weight (kg)": "120"})
    expect(page.get_by_test_id("hospital-bmi")).to_contain_text("37")

    add_stop_bang(page)
    sb = page.get_by_test_id("assessment-stopbang")
    expect(sb.get_by_test_id("assessment-filled")).to_contain_text("7 of 8 filled")
    for letter, source in [("S", "HQ"), ("T", "HQ"), ("O", "HQ"), ("P", "HQ"), ("B", "Hospital BMI"), ("A", "Profile"), ("N", "Missing"), ("G", "Profile")]:
        expect(item(page, letter).get_by_test_id("item-source")).to_have_text(source)
    expect(item(page, "N")).to_have_attribute("data-missing", "true")
    expect(item(page, "B").get_by_role("textbox")).to_have_value("37")
    expect(sb.get_by_test_id("assessment-result")).to_have_text("Incomplete")
    expect(sb.get_by_role("button", name="Confirm")).to_be_disabled()
    expect(sb.get_by_test_id("assessment-confirmed")).to_have_text("Not confirmed")

    # The neck, then Confirm: S, T, P, B, A and G score → 6, High risk.
    sb.get_by_role("button", name="N: No").click()
    expect(item(page, "N").get_by_test_id("item-source")).to_have_text("Clinician")
    expect(sb.get_by_test_id("assessment-result")).to_have_text("6 (High risk)")
    sb.get_by_role("button", name="Confirm").click()
    expect(sb.get_by_test_id("assessment-confirmed")).to_contain_text("Confirmed by Cara Clinician")
    page.reload()
    expect(sb.get_by_test_id("assessment-confirmed")).to_contain_text("Confirmed by Cara Clinician")
    expect(item(page, "N").get_by_test_id("item-source")).to_have_text("Clinician")
    expect(sb.get_by_role("button", name="Confirm")).to_be_disabled()  # nothing changed since
    # One STOP-Bang per episode: the menu offers only the others (S48).
    page.get_by_role("combobox", name="Add assessment").click()
    expect(page.get_by_role("option", name="DASI", exact=True)).to_be_visible()
    expect(page.get_by_role("option", name="STOP-Bang", exact=True)).to_have_count(0)
    page.keyboard.press("Escape")

    # The POA Summary: the score and band beside the hospital BMI.
    page.goto(f"{base}/episodes/{episode['id']}/poa")
    expect(page.get_by_test_id("poa-detail-Hospital BMI")).to_contain_text("37")
    expect(page.get_by_test_id("poa-detail-STOP-Bang")).to_have_text("STOP-Bang6 (High risk)")

    # A change needs a new Confirm: saved without it, the POA Summary says Not confirmed.
    page.goto(f"{base}/episodes/{episode['id']}")
    sb.get_by_role("button", name="N: Yes").click()
    expect(sb.get_by_test_id("assessment-confirmed")).to_have_text("Changed: confirm it again")
    expect(sb.get_by_test_id("assessment-result")).to_have_text("7 (High risk)")
    sb.get_by_role("button", name="Save").click()
    expect(sb.get_by_test_id("assessment-confirmed")).to_have_text("Not confirmed")
    page.goto(f"{base}/episodes/{episode['id']}/poa")
    expect(page.get_by_test_id("poa-detail-STOP-Bang")).to_contain_text("Not confirmed")
    page.goto(f"{base}/episodes/{episode['id']}")
    sb.get_by_role("button", name="Confirm").click()
    expect(sb.get_by_test_id("assessment-confirmed")).to_contain_text("Confirmed by Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}/poa")
    expect(page.get_by_test_id("poa-detail-STOP-Bang")).to_contain_text("7 (High risk)")


def test_an_hq_with_stop_bang_opens_it_complete_waiting_for_confirm(page: Page, as_user, patient_link):
    hq = as_user("Alex Author").published_hq("stopbang-hq.json", f"E2E J12 STOP-Bang HQ {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 SB HQ {unique()}")  # a woman of 46
    patient = patient_link(episode)
    patient.answer("Sleep", {"q_e2e_sb_bmi": {"height": 170, "weight": 82}, "stopbang_snoring": "o_yes", "stopbang_tired": "o_no",
                             "stopbang_observed": "o_no", "stopbang_pressure": "o_no", "stopbang_neck": "o_no"})
    patient.submit()

    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    add_stop_bang(page)
    sb = page.get_by_test_id("assessment-stopbang")
    expect(sb.get_by_test_id("assessment-filled")).to_contain_text("8 of 8 filled")
    expect(item(page, "N").get_by_test_id("item-source")).to_have_text("HQ")
    expect(item(page, "B").get_by_test_id("item-source")).to_have_text("Patient reported BMI")  # no readings yet
    expect(sb.get_by_test_id("assessment-result")).to_have_text("1 (Low risk)")
    expect(sb.get_by_test_id("assessment-confirmed")).to_have_text("Not confirmed")
    page.goto(f"{base}/episodes/{episode['id']}/poa")
    expect(page.get_by_test_id("poa-detail-STOP-Bang")).to_contain_text("Not confirmed")
    page.goto(f"{base}/episodes/{episode['id']}")
    # A hospital BMI recorded with the assessment open takes the item's place at once (observations
    # come first: the trust's precedence is on), in the box too: 180 cm, 125 kg → 38.6.
    expect(item(page, "B").get_by_role("textbox")).to_have_value("28.4")
    add(page, "Height", {"Height (cm)": "180"})
    add(page, "Weight", {"Weight (kg)": "125"})
    expect(item(page, "B").get_by_test_id("item-source")).to_have_text("Hospital BMI")
    expect(item(page, "B").get_by_role("textbox")).to_have_value("38.6")
    expect(sb.get_by_test_id("assessment-result")).to_have_text("2 (Low risk)")
    sb.get_by_role("button", name="Confirm").click()
    expect(sb.get_by_test_id("assessment-confirmed")).to_contain_text("Confirmed by Cara Clinician")


def test_a_second_tab_adding_stop_bang_is_told_it_is_already_there(page: Page, as_user):
    """S47 test report 1, F-1: the 409's message stays on screen once the card has reloaded."""
    hq = as_user("Alex Author").published_hq("stopbang-items-hq.json", f"E2E J12 STOP-Bang {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 SB tabs {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    expect(page.get_by_test_id("assessments").get_by_text("No assessments yet.")).to_be_visible()
    other = page.context.new_page()
    other.goto(f"{base}/episodes/{episode['id']}")
    add_stop_bang(other)

    page.get_by_role("combobox", name="Add assessment").click()
    page.get_by_role("option", name="STOP-Bang", exact=True).click()
    cards = page.get_by_test_id("assessments")
    expect(cards.get_by_test_id("assessment-stopbang")).to_have_count(1)
    expect(cards.get_by_text("STOP-Bang is already on this episode.")).to_be_visible()
    page.wait_for_timeout(500)  # it stays, after the reload
    expect(cards.get_by_text("STOP-Bang is already on this episode.")).to_be_visible()


def test_a_bmi_outside_its_limits_is_flagged_and_not_saved(page: Page, as_user):
    """S47 test report 1: the box says the limits, and Save and Confirm wait for a BMI that fits."""
    hq = as_user("Alex Author").published_hq("stopbang-items-hq.json", f"E2E J12 STOP-Bang {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 SB BMI {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    add_stop_bang(page)
    sb = page.get_by_test_id("assessment-stopbang")
    box = item(page, "B").get_by_role("textbox")
    save = sb.get_by_role("button", name="Save")
    page.set_viewport_size({"width": 1456, "height": 829})
    x = box.bounding_box()["x"]  # B Missing
    # Verification 2, minor 2: a box that started empty can be emptied again, leaving B missing.
    box.fill("40")
    box.fill("")
    expect(item(page, "B").get_by_test_id("item-source")).to_have_text("Missing")
    expect(item(page, "B").get_by_text("Enter a BMI")).to_have_count(0)
    expect(sb.get_by_role("button", name="Undo changes")).to_have_count(0)
    box.fill("40")
    expect(item(page, "B").get_by_test_id("item-source")).to_have_text("Entered by clinician")
    assert box.bounding_box()["x"] == x  # the box doesn't move as the source changes (report 2)
    # The badge shows its words whole (report 1: they were cut off at 1456 wide).
    badge = item(page, "B").get_by_test_id("item-source")
    assert badge.evaluate("b => [b, ...b.querySelectorAll('*')].every(e => e.scrollWidth <= e.clientWidth)")
    expect(save).to_be_enabled()
    box.fill("2")
    expect(item(page, "B").get_by_text("Between 5 and 150")).to_be_visible()
    expect(save).to_be_disabled()
    # Report 2: B is missing while its box is wrong; the score doesn't use the 40 typed before.
    expect(item(page, "B").get_by_test_id("item-source")).to_have_text("Missing")
    box.fill("abc")  # not the 40 typed before
    expect(item(page, "B").get_by_text("A number, one decimal place")).to_be_visible()
    expect(save).to_be_disabled()
    box.fill("36,5")
    expect(save).to_be_enabled()
    save.click()
    expect(sb.get_by_test_id("assessment-confirmed")).to_have_text("Not confirmed")
    expect(item(page, "B").get_by_role("textbox")).to_have_value("36.5")
    item(page, "B").get_by_role("textbox").fill("")
    expect(item(page, "B").get_by_text("Enter a BMI")).to_be_visible()
    expect(item(page, "B").get_by_test_id("item-source")).to_have_text("Missing")
    expect(save).to_be_disabled()
    sb.get_by_role("button", name="Undo changes").click()
    expect(item(page, "B").get_by_role("textbox")).to_have_value("36.5")
    expect(item(page, "B").get_by_test_id("item-source")).to_have_text("Entered by clinician")


def test_stop_bang_waits_for_the_observations(page: Page, as_user, patient_link):
    """S47 verification 1, finding 2: with the observations failing to load, the card says so and
    can't be confirmed, or B would quietly be the patient's BMI against the trust's setting."""
    hq = as_user("Alex Author").published_hq("stopbang-hq.json", f"E2E J12 STOP-Bang HQ {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 SB load {unique()}")
    patient = patient_link(episode)
    patient.answer("Sleep", {"q_e2e_sb_bmi": {"height": 170, "weight": 82}, "stopbang_snoring": "o_yes", "stopbang_tired": "o_no",
                             "stopbang_observed": "o_no", "stopbang_pressure": "o_no", "stopbang_neck": "o_no"})
    patient.submit()
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    add_stop_bang(page)
    sb = page.get_by_test_id("assessment-stopbang")
    expect(sb.get_by_role("button", name="Confirm")).to_be_enabled()  # 8 of 8 filled

    page.route("**/observations", lambda route: route.fulfill(status=500, json={"error": "Something went wrong."}))
    page.reload()
    expect(sb.get_by_test_id("assessment-filled")).to_contain_text("8 of 8 filled")
    expect(sb.get_by_test_id("assessment-inputs")).to_have_text(
        "The observations didn't load, so the hospital's readings can't be checked. Reload the page to save or confirm.")
    expect(sb.get_by_role("button", name="Confirm")).to_be_disabled()


def test_the_poa_summary_says_when_the_assessments_did_not_load(page: Page, as_user):
    """S47 verification 2: a confirmed STOP-Bang mustn't silently vanish from the POA Summary."""
    hq = as_user("Alex Author").published_hq("stopbang-items-hq.json", f"E2E J12 STOP-Bang {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 SB POA load {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    page.route("**/assessments", lambda route: route.fulfill(status=500, json={"error": "Something went wrong."}))
    page.goto(f"{base}/episodes/{episode['id']}/poa")
    expect(page.get_by_test_id("poa-detail-Assessments")).to_contain_text("Couldn't be loaded")


def test_a_confirmed_stop_bang_says_what_has_changed_since(page: Page, as_user, patient_link):
    """S47 answers 5 and 6 (Rahul, 6 Nov)."""
    hq = as_user("Alex Author").published_hq("stopbang-items-hq.json", f"E2E J12 STOP-Bang {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"J12 SB since {unique()}", dateOfBirth="1960-03-01", sex="male")  # made up: a man of 66
    answers = {"q_e2e_bmi": {"height": 170, "weight": 82}, "stopbang_snoring": "o_yes", "stopbang_tired": "o_yes",
               "stopbang_observed": "o_no", "stopbang_pressure": "o_yes"}
    patient = patient_link(episode)
    patient.answer("Sleep", answers)
    patient.submit()
    # Answers 5: Cara has started validating Sleep and changed Tired to No, but not pressed Validate.
    cara.validate(episode, "Sleep", {**answers, "stopbang_tired": "o_no"}, validated=False)

    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    add(page, "Height", {"Height (cm)": "180"})
    add(page, "Weight", {"Weight (kg)": "120"})  # 37.0
    add_stop_bang(page)
    sb = page.get_by_test_id("assessment-stopbang")
    expect(item(page, "T").get_by_role("button", name="T: No")).to_have_attribute("aria-pressed", "true")
    expect(item(page, "T").get_by_test_id("item-source")).to_have_text("HQ")
    # S, P, B, A and G score → 5, High risk.
    sb.get_by_role("button", name="N: No").click()
    expect(sb.get_by_test_id("assessment-result")).to_have_text("5 (High risk)")
    sb.get_by_role("button", name="Confirm").click()
    expect(sb.get_by_test_id("assessment-confirmed")).to_contain_text("Confirmed by Cara Clinician")
    expect(sb.get_by_test_id("assessment-since")).to_have_count(0)

    # Answers 6: the weight corrected, the card says so; the score stays as confirmed.
    page.get_by_role("button", name="Correct Weight 120.0 kg").click()
    fix = page.get_by_test_id("observation-correct")
    fix.get_by_label("Weight (kg)").fill("90")
    fix.get_by_label("Why is it corrected?").fill("Made-up: scale not zeroed")
    fix.get_by_role("button", name="Save correction").click()
    expect(page.get_by_test_id("hospital-bmi")).to_contain_text("27.8")
    expect(sb.get_by_test_id("assessment-since")).to_contain_text("The hospital BMI is now 27.8 (37 when confirmed).")
    expect(item(page, "B").get_by_role("textbox")).to_have_value("37")
    expect(sb.get_by_test_id("assessment-result")).to_have_text("5 (High risk)")
    expect(sb.get_by_test_id("assessment-confirmed")).to_contain_text("Confirmed by Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}/poa")
    expect(page.get_by_test_id("poa-detail-STOP-Bang")).to_contain_text("5 (High risk)")


DASI = [
    ("Can you take care of yourself, that is, eating, dressing, bathing or using the toilet?", "Yes"),
    ("Can you walk indoors, such as around your house?", "Yes"),
    ("Can you walk a block or two on level ground?", "No"),
    ("Can you climb a flight of stairs or walk up a hill?", "Yes"),
    ("Can you run a short distance?", "Yes"),
    ("Can you do light work around the house like dusting or washing dishes?", "No"),
    ("Can you do moderate work around the house like vacuuming, sweeping floors, or carrying in groceries?", "Yes"),
    ("Can you do heavy work around the house like scrubbing floors, or lifting or moving heavy furniture?", "Yes"),
    ("Can you do yardwork like raking leaves, weeding or pushing a power mower?", "Yes"),
    ("Can you have sexual relations?", "No"),
    ("Can you participate in moderate recreational activities like golf, bowling, dancing, doubles tennis, or throwing a baseball or football?", "No"),
    ("Can you participate in strenuous sports like swimming, singles tennis, football, basketball or skiing?", "No"),
]  # 2.75 + 1.75 + 5.5 + 8 + 3.5 + 8 + 4.5 = 34: Poor functional capacity, the cut-off itself


def add_assessment(page: Page, name: str, test_id: str):
    """Add assessment → <name> on the episode page; its card."""
    page.get_by_role("combobox", name="Add assessment").click()
    page.get_by_role("option", name=name, exact=True).click()
    card = page.get_by_test_id(f"assessment-{test_id}")
    expect(card).to_be_visible()
    return card


def test_an_hq_with_dasi_sent_on_a_phone_opens_dasi_complete(page: Page, phone: BrowserContext, as_user):
    hq = as_user("Alex Author").published_hq("fitness-hq.json", f"E2E J12 fitness HQ {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 DASI {unique()}")  # a woman of 46

    # The patient, on their phone: DASI, PRISMA-7 and Apfel's questions, one page each.
    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_test_id("set-Fitness and recovery").get_by_role("button", name="Start").click()
    for question, answer in DASI:
        choose(patient, question, answer)
    expect(patient.get_by_text("DASI score")).to_have_count(0)  # the score and METs are the clinician's
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
    patient.get_by_role("button", name="Next", exact=True).click()
    choose(patient, "In general, do you have any health problems that require you to limit your activities?", "No")
    choose(patient, "Do you need someone to help you on a regular basis?", "No")
    choose(patient, "In general, do you have any health problems that require you to stay at home?", "No")
    choose(patient, "If you need help, can you count on someone close to you?", "Yes")
    choose(patient, "Do you regularly use a stick, walker or wheelchair to move about?", "No")
    patient.get_by_role("button", name="Next", exact=True).click()
    choose(patient, "Do you smoke?", "No")
    choose(patient, "Have you ever felt or been sick after an anaesthetic, or do you get travel sickness (motion sickness)?", "Yes")
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
    patient.get_by_role("button", name="Finish this section").click()
    patient.get_by_role("button", name="Send my answers").click()
    patient.get_by_role("dialog", name="Send your answers?").get_by_role("button", name="Send my answers").click()
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()

    # Cara: DASI opens complete, from the HQ, with its score, band and METs; she confirms it.
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    dasi = add_assessment(page, "DASI", "dasi")
    expect(dasi.get_by_test_id("assessment-filled")).to_have_text("12 of 12 filled.")
    expect(dasi.get_by_test_id("item-source").filter(has_text="HQ")).to_have_count(12)
    expect(dasi.get_by_test_id("assessment-result")).to_have_text("34 (Poor functional capacity), 6.9 METs")
    # The badge shows its words whole, at a laptop's width (S47 report 1's lesson for the source badges).
    page.set_viewport_size({"width": 1456, "height": 829})
    badge = dasi.get_by_test_id("assessment-result")
    assert badge.evaluate("b => [b, ...b.querySelectorAll('*')].every(e => e.scrollWidth <= e.clientWidth)")
    expect(dasi.get_by_test_id("assessment-note")).to_have_count(0)  # DASI uses no age or sex
    dasi.get_by_role("button", name="Confirm").click()
    expect(dasi.get_by_test_id("assessment-confirmed")).to_contain_text("Confirmed by Cara Clinician")

    # PRISMA-7: 7 of 7, questions 1 and 2 from her record (46, female); question 6's Yes scores.
    prisma = add_assessment(page, "PRISMA-7", "prisma7")
    expect(prisma.get_by_test_id("assessment-filled")).to_have_text("7 of 7 filled.")
    expect(prisma.get_by_test_id("item-1").get_by_test_id("item-source")).to_have_text("Profile")
    expect(prisma.get_by_test_id("assessment-result")).to_have_text("1 (Not at increased risk)")

    # Apfel: 3 of 4, only the post-op opioids missing; with it, the score and the published risk.
    apfel = add_assessment(page, "Apfel PONV", "apfel")
    expect(apfel.get_by_test_id("assessment-filled")).to_have_text("3 of 4 filled: answer the item highlighted.")
    expect(apfel.locator("[data-missing=true]")).to_have_count(1)
    expect(apfel.get_by_test_id("item-4")).to_have_attribute("data-missing", "true")
    expect(apfel.get_by_test_id("assessment-result")).to_have_text("Incomplete")
    apfel.get_by_role("button", name="4: Yes").click()
    expect(apfel.get_by_test_id("item-4").get_by_test_id("item-source")).to_have_text("Clinician")
    expect(apfel.get_by_test_id("assessment-result")).to_have_text("4 (PONV risk about 79%)")
    apfel.get_by_role("button", name="Confirm").click()
    expect(apfel.get_by_test_id("assessment-confirmed")).to_contain_text("Confirmed by Cara Clinician")

    # The POA Summary: DASI with its METs, Apfel's risk, PRISMA-7 not confirmed.
    page.goto(f"{base}/episodes/{episode['id']}/poa")
    expect(page.get_by_test_id("poa-detail-DASI")).to_contain_text("34 (Poor functional capacity), 6.9 METs")
    expect(page.get_by_test_id("poa-detail-Apfel PONV")).to_contain_text("4 (PONV risk about 79%)")
    expect(page.get_by_test_id("poa-detail-PRISMA-7")).to_contain_text("Not confirmed")


def test_must_takes_the_hospital_bmi_and_follows_a_corrected_weight(page: Page, as_user, patient_link):
    hq = as_user("Alex Author").published_hq("risk-hq.json", f"E2E J12 risk HQ {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 MUST {unique()}")  # a woman of 46
    patient = patient_link(episode)
    # Her own height and weight (165 cm, 70 kg), 6 kg lost without trying; no falls, no clot risks.
    patient.answer("Weight, falls and clots", {
        "height": 165, "weight": 70, "must_wtloss": "o_yes", "must_wtloss_kg": 6, "morse_falls": "o_no", "morse_aid": "o_aid_none",
        **{k: "o_no" for k in ["vte_cancer", "vte_thrombophilia", "vte_history", "vte_hrt", "vte_contraceptive", "vte_varicose",
                               "vte_pregnancy", "vte_anticoagulant", "vte_inherited_bleeding"]},
    })
    patient.submit()

    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    add(page, "Height", {"Height (cm)": "170"})
    add(page, "Weight", {"Weight (kg)": "60"})
    expect(page.get_by_test_id("hospital-bmi")).to_contain_text("20.8")

    # MUST opens 3 of 4: the hospital BMI and weight, the weight lost from the HQ; the acute disease effect is hers.
    must = add_assessment(page, "MUST", "must")
    expect(must.get_by_test_id("assessment-filled")).to_have_text("3 of 4 filled: answer the item highlighted.")
    for letter, source in [("1", "Hospital BMI"), ("2", "Hospital weight"), ("3", "HQ"), ("4", "Missing")]:
        expect(must.get_by_test_id(f"item-{letter}").get_by_test_id("item-source")).to_have_text(source)
    expect(must.get_by_test_id("item-1").get_by_role("textbox")).to_have_value("20.8")
    expect(must.get_by_test_id("item-2").get_by_role("textbox")).to_have_value("60")
    expect(must.get_by_test_id("item-3").get_by_role("textbox")).to_have_value("6")
    expect(must.get_by_test_id("assessment-result")).to_have_text("Incomplete")
    must.get_by_role("button", name="4: No").click()
    # BMI 20.8 is 0; 6 kg of 66 is 9.1%, 1.
    expect(must.get_by_test_id("assessment-result")).to_have_text("1 (Medium risk), 9.1% of body weight lost")
    must.get_by_role("button", name="Save").click()
    expect(must.get_by_test_id("assessment-confirmed")).to_have_text("Not confirmed")

    # The weight corrected (S46) before Confirm: MUST takes the new BMI and weight.
    page.get_by_role("button", name="Correct Weight 60.0 kg").click()
    fix = page.get_by_test_id("observation-correct")
    fix.get_by_label("Weight (kg)").fill("55")
    fix.get_by_label("Why is it corrected?").fill("Made-up: weighed in outdoor clothes")
    fix.get_by_role("button", name="Save correction").click()
    expect(page.get_by_test_id("hospital-bmi")).to_contain_text("19")
    expect(must.get_by_test_id("item-1").get_by_role("textbox")).to_have_value("19")
    expect(must.get_by_test_id("item-2").get_by_role("textbox")).to_have_value("55")
    # BMI 19 is 1; 6 kg of 61 is 9.8%, 1.
    expect(must.get_by_test_id("assessment-result")).to_have_text("2 (High risk), 9.8% of body weight lost")
    must.get_by_role("button", name="Confirm").click()
    expect(must.get_by_test_id("assessment-confirmed")).to_contain_text("Confirmed by Cara Clinician")

    # A weight typed on the card leaves the BMI for her: never one body's BMI beside another's weight (verification 3).
    must.get_by_role("textbox", name="2: Weight now").fill("50")
    expect(must.get_by_test_id("item-1").get_by_test_id("item-source")).to_have_text("Missing")
    expect(must.get_by_test_id("item-1").get_by_role("textbox")).to_have_value("")
    expect(must.get_by_test_id("assessment-result")).to_have_text("Incomplete")
    expect(must.get_by_role("button", name="Confirm")).to_be_disabled()
    must.get_by_role("textbox", name="1: BMI").fill("17.3")
    expect(must.get_by_test_id("assessment-result")).to_have_text("4 (High risk), 10.7% of body weight lost")
    must.get_by_role("button", name="Undo changes").click()
    expect(must.get_by_test_id("assessment-result")).to_have_text("2 (High risk), 9.8% of body weight lost")
    # A BMI typed, then the hospital's own weight typed back: that weight is hers too (verification 4, finding 1).
    must.get_by_role("textbox", name="1: BMI").fill("30")
    expect(must.get_by_test_id("item-2").get_by_test_id("item-source")).to_have_text("Missing")
    must.get_by_role("textbox", name="2: Weight now").fill("55")
    expect(must.get_by_test_id("item-2").get_by_test_id("item-source")).to_have_text("Entered by clinician")
    # BMI 30 is 0; 6 kg of 61 is 9.8%, 1.
    expect(must.get_by_test_id("assessment-result")).to_have_text("1 (Medium risk), 9.8% of body weight lost")
    expect(must.get_by_role("button", name="Confirm")).to_be_enabled()
    must.get_by_role("button", name="Undo changes").click()
    expect(must.get_by_test_id("assessment-result")).to_have_text("2 (High risk), 9.8% of body weight lost")
    # A BMI mistyped and put right, then a weight typed: the BMI is still left for her (verification 5, finding 1).
    bmi_box = must.get_by_role("textbox", name="1: BMI")
    bmi_box.fill("36")
    bmi_box.fill("19")
    expect(must.get_by_test_id("item-1").get_by_test_id("item-source")).to_have_text("Hospital BMI")
    must.get_by_role("textbox", name="2: Weight now").fill("50")
    expect(must.get_by_test_id("item-1").get_by_test_id("item-source")).to_have_text("Missing")
    # Its box is empty too, not the 19 typed before (test report 7, F-1).
    expect(bmi_box).to_have_value("")
    # Typing in the emptied box and deleting it isn't a mistake: Save stays on (finding 2).
    bmi_box.fill("1")
    bmi_box.fill("")
    expect(must.get_by_text("Enter a BMI")).to_have_count(0)
    expect(must.get_by_role("button", name="Save")).to_be_enabled()
    must.get_by_role("button", name="Undo changes").click()
    expect(must.get_by_test_id("assessment-result")).to_have_text("2 (High risk), 9.8% of body weight lost")
    # The weight changed, the BMI typed into its emptied box, the weight put back, then another weight:
    # the BMI typed earlier is left for her again (verification 6, finding 1).
    weight_box = must.get_by_role("textbox", name="2: Weight now")
    weight_box.fill("56")
    bmi_box.fill("19")
    expect(must.get_by_test_id("item-1").get_by_test_id("item-source")).to_have_text("Entered by clinician")
    weight_box.fill("55")
    expect(must.get_by_test_id("item-1").get_by_test_id("item-source")).to_have_text("Hospital BMI")
    expect(must.get_by_test_id("item-2").get_by_test_id("item-source")).to_have_text("Hospital weight")
    weight_box.fill("50")
    expect(must.get_by_test_id("item-1").get_by_test_id("item-source")).to_have_text("Missing")
    expect(bmi_box).to_have_value("")
    expect(must.get_by_test_id("assessment-result")).to_have_text("Incomplete")
    expect(must.get_by_role("button", name="Confirm")).to_be_disabled()
    must.get_by_role("button", name="Undo changes").click()
    expect(must.get_by_test_id("assessment-result")).to_have_text("2 (High risk), 9.8% of body weight lost")
    # The mirror: a weight typed and put back, then a BMI typed: the weight's box is empty (F-1).
    weight_box.fill("61")
    weight_box.fill("55")
    expect(must.get_by_test_id("item-2").get_by_test_id("item-source")).to_have_text("Hospital weight")
    bmi_box.fill("17.3")
    expect(must.get_by_test_id("item-2").get_by_test_id("item-source")).to_have_text("Missing")
    expect(weight_box).to_have_value("")
    must.get_by_role("button", name="Undo changes").click()
    expect(must.get_by_test_id("assessment-result")).to_have_text("2 (High risk), 9.8% of body weight lost")

    # The POA Summary: MUST beside the hospital BMI.
    page.goto(f"{base}/episodes/{episode['id']}/poa")
    expect(page.get_by_test_id("poa-detail-Hospital BMI")).to_contain_text("19")
    expect(page.get_by_test_id("poa-detail-MUST")).to_contain_text("2 (High risk), 9.8% of body weight lost")


def test_must_with_a_hospital_weight_but_no_height_keeps_the_weight_when_the_bmi_is_typed(page: Page, as_user):
    """S49 verification 5, Rahul 13 Nov: the BMI the card asks for keeps the hospital's weight beside it."""
    hq = as_user("Alex Author").published_hq("risk-hq.json", f"E2E J12 MUST weight {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 MUST weight {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    add(page, "Weight", {"Weight (kg)": "60"})
    must = add_assessment(page, "MUST", "must")
    expect(must.get_by_test_id("item-1").get_by_test_id("item-source")).to_have_text("Missing")
    expect(must.get_by_test_id("item-2").get_by_test_id("item-source")).to_have_text("Hospital weight")
    must.get_by_role("textbox", name="1: BMI").fill("20.8")
    expect(must.get_by_test_id("item-1").get_by_test_id("item-source")).to_have_text("Entered by clinician")
    expect(must.get_by_test_id("item-2").get_by_test_id("item-source")).to_have_text("Hospital weight")
    expect(must.get_by_test_id("item-2").get_by_role("textbox")).to_have_value("60")
    # Saved, reloaded, then that BMI changed: the hospital's weight still stays beside it (S49a).
    must.get_by_role("button", name="Save").click()
    expect(must.get_by_test_id("assessment-confirmed")).to_have_text("Not confirmed")
    page.reload()
    must = page.get_by_test_id("assessment-must")
    expect(must.get_by_test_id("item-1").get_by_role("textbox")).to_have_value("20.8")
    expect(must.get_by_test_id("item-2").get_by_test_id("item-source")).to_have_text("Hospital weight")
    must.get_by_role("textbox", name="1: BMI").fill("22")
    expect(must.get_by_test_id("item-1").get_by_test_id("item-source")).to_have_text("Entered by clinician")
    expect(must.get_by_test_id("item-2").get_by_test_id("item-source")).to_have_text("Hospital weight")
    expect(must.get_by_test_id("item-2").get_by_role("textbox")).to_have_value("60")


def test_a_long_result_lines_up_on_the_poa_summary(page: Page, as_user, patient_link):
    """S49 test report 1: VTE's result is the longest; its value starts where the others do (F-1). The
    Frailty Scale's longest level shows whole in its menu (exploring 2)."""
    hq = as_user("Alex Author").published_hq("risk-hq.json", f"E2E J12 VTE {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 VTE {unique()}", dateOfBirth="1950-03-01")
    patient = patient_link(episode)
    patient.answer("Weight, falls and clots", {
        "height": 165, "weight": 70, "must_wtloss": "o_no", "morse_falls": "o_no", "morse_aid": "o_aid_none",
        **{k: "o_no" for k in ["vte_cancer", "vte_thrombophilia", "vte_history", "vte_hrt", "vte_contraceptive", "vte_varicose",
                               "vte_pregnancy", "vte_anticoagulant", "vte_inherited_bleeding"]},
    })
    patient.submit()

    base = sign_in(page.context, "Cara Clinician")
    page.set_viewport_size({"width": 1456, "height": 900})
    page.goto(f"{base}/episodes/{episode['id']}")
    add(page, "Height", {"Height (cm)": "170"})
    add(page, "Weight", {"Weight (kg)": "95"})
    vte = add_assessment(page, "VTE risk", "vte")
    expect(vte.get_by_test_id("assessment-filled")).to_have_text("11 of 30 filled: answer the 19 items highlighted.")
    missing = vte.locator("[data-missing=true]")
    for left in range(19, 0, -1):  # each No takes its row off the missing list
        expect(missing).to_have_count(left)
        if left == 1:  # every thrombosis item answered, the last bleeding one not (verification 2, finding 1)
            expect(vte.get_by_test_id("assessment-result")).to_have_text("Incomplete")
            expect(vte.get_by_role("button", name="Confirm")).to_be_disabled()
        missing.first.get_by_role("button", name="No").click()
    expect(vte.get_by_test_id("assessment-result")).to_have_text("2 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0")
    vte.get_by_role("button", name="Confirm").click()
    expect(vte.get_by_test_id("assessment-confirmed")).to_contain_text("Confirmed by Cara Clinician")

    # The Frailty Scale's menu shows the longest level whole (report 1, exploring 2).
    cfs = add_assessment(page, "Clinical Frailty Scale", "rockwood")
    cfs.get_by_label("1: Level").click()
    page.get_by_role("option", name="8: Living with Very Severe Frailty").click()
    level = cfs.get_by_label("1: Level")
    expect(level).to_have_value("8: Living with Very Severe Frailty")
    assert level.evaluate("e => e.scrollWidth <= e.clientWidth")

    page.goto(f"{base}/episodes/{episode['id']}/poa")
    long = page.get_by_test_id("poa-detail-VTE risk")
    expect(long).to_contain_text("2 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0")
    value_x = lambda label: page.get_by_test_id(f"poa-detail-{label}").locator("p").nth(1).bounding_box()["x"]
    assert value_x("VTE risk") == value_x("Hospital BMI")


def test_a_man_is_not_asked_about_pregnancy_contraception_or_hrt(phone: BrowserContext, as_user):
    """S49 verification 1 (Rahul's answer 2): VTE's HQ part skips three questions for a patient recorded as male."""
    hq = as_user("Alex Author").published_hq("risk-hq.json", f"E2E J12 VTE male {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J12 VTE man {unique()}", dateOfBirth="1960-03-01", sex="male")
    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("1")
    patient.get_by_label("Month").fill("3")
    patient.get_by_label("Year").fill("1960")
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_test_id("set-Weight, falls and clots").get_by_role("button", name="Start").click()
    patient.get_by_label("How tall are you?").fill("178")
    patient.get_by_label("What do you weigh now?").fill("80")
    choose(patient, "Have you lost weight without trying in the last 3 to 6 months?", "No")
    patient.get_by_role("button", name="Next", exact=True).click()
    choose(patient, "Have you had a fall in the last 3 months?", "No")
    choose(patient, "Do you use anything to help you walk?", "Nothing")
    patient.get_by_role("button", name="Next", exact=True).click()
    expect(patient.get_by_text("Do you have cancer, or are you having treatment for cancer?")).to_be_visible()
    for question in ["Are you pregnant, or have you had a baby in the last 6 weeks?", "Are you taking hormone replacement therapy (HRT)?",
                     "Are you using a contraceptive that contains oestrogen (such as the combined pill, patch or ring)?"]:
        expect(patient.get_by_text(question)).to_have_count(0)
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
