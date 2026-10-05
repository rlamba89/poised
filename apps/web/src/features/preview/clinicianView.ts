// The clinician view (VEW-02), shared by the preview and episode validation: whole pages, each
// page's Clinical summary box (QT-09), and clinician items marked so the two-column layout
// (.sj-clinician-layout in globals.css) can put them on the right.
import { Model } from "survey-core";
import {
  isClinicianOnly, setViewer, showBands, withClinicalSummaries, SUMMARY_PREFIX, type ChapterJson, type SamplePatient,
} from "@poised/clinical";
import { showUnits } from "./units";

export function clinicianModel(content: ChapterJson, patient?: SamplePatient): Model {
  const m = new Model(withClinicalSummaries(content));
  setViewer(m, "clinician");
  if (patient) {
    m.setVariable("patientName", patient.name);
    m.setVariable("patientAge", patient.age);
    m.setVariable("patientSex", patient.sex);
  }
  showUnits(m);
  showBands(m);
  m.widthMode = "responsive"; // two columns need the room
  m.questionsOnPageMode = "standard";
  m.showProgressBar = false;
  m.onUpdateQuestionCssClasses.add((_, o) => {
    // mainRoot is the question's box; root is only its input (e.g. the textarea wrapper).
    if (isClinicianOnly(o.question)) o.cssClasses.mainRoot += " sj-clinician";
  });
  m.onUpdatePanelCssClasses.add((_, o) => {
    if (isClinicianOnly(o.panel)) o.cssClasses.panel.container += o.panel.name.startsWith(SUMMARY_PREFIX) ? " sj-clinician sj-summary" : " sj-clinician";
  });
  return m;
}
