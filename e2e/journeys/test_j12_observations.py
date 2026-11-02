"""J12: observations (plan C7, session S46; OBS-01–04). S47 adds STOP-Bang.

  - Cara records height, weight and two blood pressures on an episode. A negative weight is refused
    with Go's reason; a heart rate of 220 warns ("outside the usual range"), then saves with Save
    anyway. The episode page shows the hospital BMI.
  - She corrects the weight, with a reason: the old reading stays, struck through, and the hospital
    BMI follows the corrected weight. A blood pressure marked entered in error stays in the history.
  - The POA Summary shows the hospital BMI beside the patient's own (their HQ's BMI question).
"""

from playwright.sync_api import Page, expect

from support import pick, sign_in, unique


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
