// OBSOLETE (S23, 22 Oct 2026): this uses the API paths from before S08 (/api/h/…), so it no
// longer runs. Don't revive it as it is: its q_angina and q_diabetes have other answers than the
// seeded Demo pre-op HQ's, and since S23 a key published with different answers in one trust
// blocks publishing that question anywhere in it. The three "S02 test HQ" it made were removed.
// S02 manual test helper. Paste the whole file into the DevTools Console while signed in as
// Alex Author on Hospital A's questionnaire list. It creates and publishes "S02 test HQ <time>":
//   - "About you" (patient): two pages. Every question a patient must NOT see has STAFF ONLY in
//     its title, and the "Yes" answer to smoking carries the disclosure note "Current smoker".
//   - "Your health" (patient): one question.
//   - "Nurse checks" (clinician): one question.
// It prints the questionnaire's name when it's done. Made-up content only.
(async () => {
  const hid = location.pathname.split("/")[2];
  const call = async (method, path, body) => {
    const r = await fetch(`/api/h/${hid}${path}`, {
      method, headers: { "Content-Type": "application/json" }, body: body && JSON.stringify(body),
    });
    if (!r.ok) throw new Error(`${method} ${path}: ${r.status} ${await r.text()}`);
    return r.status === 204 ? null : r.json();
  };
  // The editor's own Yes / No question: a radiogroup with Yes and No choices (not SurveyJS's
  // "boolean", which the app never makes).
  const yesNo = (name, title, id) => ({ type: "radiogroup", name, title, yesNo: true, colCount: 0, isRequired: true,
    choices: [{ value: `${id}y`, text: "Yes" }, { value: `${id}n`, text: "No" }] });
  const sets = [
    { name: "About you", audience: "patient", content: {
      testCases: [{ id: "tc_s02", name: "STAFF ONLY test case", answers: { q_smokes: "o_yes" } }],
      pages: [
        { name: "p_lifestyle", title: "Lifestyle", clinicalSummary: true, elements: [
          { type: "radiogroup", name: "q_smokes", title: "Do you smoke?", yesNo: true, colCount: 0, isRequired: true, choices: [
            { value: "o_yes", text: "Yes", clinicalOutputs: [{ id: "out_s02a", codes: [], note: { text: "Current smoker", category: "" } }] },
            { value: "o_no", text: "No" },
          ] },
          { type: "comment", name: "q_advice", title: "STAFF ONLY: cessation advice given?", clinicianOnly: true },
          { type: "panel", name: "g_heart", title: "Your heart", elements: [
            yesNo("q_angina", "Do you have angina?", "o_ang"),
            { type: "text", name: "q_ecg", title: "STAFF ONLY: ECG reading", clinicianOnly: true },
          ] },
        ] },
        { name: "p_staff", title: "STAFF ONLY page", clinicianOnly: true, elements: [
          { type: "text", name: "q_staff_page", title: "STAFF ONLY: question on a staff page" },
        ] },
      ] } },
    { name: "Your health", audience: "patient", content: {
      pages: [{ name: "p_health", title: "Health", elements: [
        yesNo("q_diabetes", "Do you have diabetes?", "o_dia"),
      ] }] } },
    { name: "Nurse checks", audience: "clinician", content: {
      pages: [{ name: "p_checks", title: "Checks", elements: [
        { type: "text", name: "q_bp", title: "STAFF ONLY: blood pressure", isRequired: true },
      ] }] } },
  ];
  const time = new Date().toTimeString().slice(0, 5);
  const name = `S02 test HQ ${time}`;
  const { id } = await call("POST", "/questionnaires", { name, description: "Made-up HQ for S02 testing." });
  for (const s of sets) {
    const ch = await call("POST", `/questionnaires/${id}/chapters`, { name: s.name });
    await call("PATCH", `/chapters/${ch.id}`, { audience: s.audience });
    await call("PUT", `/chapters/${ch.id}/content`, { content: s.content, revision: 0 });
  }
  const q = await call("GET", `/questionnaires/${id}`);
  await call("POST", `/questionnaires/${id}/publish`, { versionId: q.questionnaire.versionId });
  console.log(`Created and published: ${name}`);
})();
