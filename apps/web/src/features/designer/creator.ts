// SurveyJS Creator set up for clinical authoring. Uses only public extension APIs.
import { ComputedUpdater, Serializer, setLicenseKey, type ItemValue, type PanelModel, type Question } from "survey-core";
import { SurveyCreator } from "survey-creator-react";
import {
  assignGroupId, assignMissingOptionIds, assignOptionId, assignQuestionId, isClinicianOnly, registerClinicalProperties,
} from "@sj/clinical";

const licenseKey = process.env.NEXT_PUBLIC_SURVEYJS_KEY;
if (licenseKey) setLicenseKey(licenseKey);
registerClinicalProperties();

/**
 * The toolbox (DSG-01 subset), in order. "Yes / No" is a Select One with two
 * options, so each answer has a stable ID and can carry outputs.
 */
const TOOLBOX: { name: string; title: string; iconFrom: string; json: object }[] = [
  { name: "group", title: "Group", iconFrom: "panel", json: { type: "panel" } },
  {
    name: "yesno",
    title: "Yes / No",
    iconFrom: "boolean",
    json: { type: "radiogroup", choices: [{ value: "yes", text: "Yes" }, { value: "no", text: "No" }], colCount: 0 },
  },
  { name: "selectone", title: "Select One", iconFrom: "radiogroup", json: { type: "radiogroup", choices: ["Item 1", "Item 2", "Item 3"] } },
  { name: "selectmany", title: "Select Many", iconFrom: "checkbox", json: { type: "checkbox", choices: ["Item 1", "Item 2", "Item 3"] } },
  { name: "shorttext", title: "Text", iconFrom: "text", json: { type: "text" } },
  { name: "date", title: "Date", iconFrom: "text:date", json: { type: "text", inputType: "date" } },
  { name: "number", title: "Number", iconFrom: "text:number", json: { type: "text", inputType: "number" } },
  { name: "statement", title: "Statement", iconFrom: "html", json: { type: "html", html: "<p>Statement text</p>" } },
];

/**
 * The built-in special options (None, Other, Don't know, Refuse) are saved without
 * custom properties, so they can't carry outputs. "None of these" is a normal option
 * marked exclusive instead. Hiding the property also hides its placeholder on the
 * canvas. "Select all" stays (QT-03); it carries no outputs of its own.
 */
for (const name of ["showNoneItem", "showOtherItem", "showDontKnowItem", "showRefuseItem"]) {
  const prop = Serializer.findProperty("selectbase", name);
  if (prop) prop.visible = false;
}

export type ChapterCreatorOptions = {
  readOnly: boolean;
  /** Extra canvas actions for a question, e.g. the Outputs badge (step 5). */
  questionActions?: (q: Question) => object[];
};

export function createChapterCreator(opts: ChapterCreatorOptions): SurveyCreator {
  const creator = new SurveyCreator({
    showLogicTab: true,
    showJSONEditorTab: false, // it would bypass the ID rules
    showPreviewTab: false, // our own preview page has the viewer switch and outputs
    showTranslationTab: false,
    showThemeTab: false,
    useElementTitles: true, // titles, not IDs, in the Logic tab and expressions
    autoSaveEnabled: true,
  });
  creator.readOnly = opts.readOnly;
  setUpToolbox(creator);

  // Stable IDs on every new or copied element (QST-04, OPT-02, STR-07).
  // Converting a question's type keeps its ID, so conditions on it keep working.
  creator.onQuestionAdded.add((_, o) => {
    const q = o.question;
    // SurveyJS numbers default names from existing names, which are IDs here,
    // so number the default title ourselves.
    if (o.reason !== "ELEMENT_COPIED" && o.reason !== "ELEMENT_CONVERTED" && q.locTitle.isEmpty) {
      q.title = `Question ${creator.survey.getAllQuestions(false, true).length}`;
    }
    if (o.reason !== "ELEMENT_CONVERTED") assignQuestionId(q);
    assignChoiceIds(q);
    // New questions are required by default (QST-02); statements can't be.
    if (o.reason !== "ELEMENT_COPIED" && o.reason !== "ELEMENT_CONVERTED" && q.getType() !== "html") q.isRequired = true;
  });
  creator.onPanelAdded.add((_, o) => {
    if (o.reason === "ELEMENT_CONVERTED") return;
    if (o.reason !== "ELEMENT_COPIED" && o.panel.locTitle.isEmpty) {
      o.panel.title = `Group ${creator.survey.getAllPanels(false, true).length}`;
    }
    // A copied group brings copies of its questions and groups; they need their own IDs.
    const walk = (panel: PanelModel) => {
      assignGroupId(panel);
      for (const el of panel.elements) {
        if (el.isPanel) walk(el as PanelModel);
        else {
          assignQuestionId(el as Question);
          assignChoiceIds(el as Question);
        }
      }
    };
    walk(o.panel);
  });
  creator.onItemValueAdded.add((_, o) => assignOptionId(o.newItem, o.itemValues));
  // Items typed into the "fast entry" box skip onItemValueAdded.
  creator.onFastEntryFinished.add((_, o) => assignMissingOptionIds(o.items));

  // IDs are shown but never edited.
  creator.onPropertyGetReadOnly.add((_, o) => {
    if (o.property.name === "name" || (o.property.name === "value" && o.parentProperty?.name === "choices")) o.readOnly = true;
  });

  // Canvas badges (STR-09).
  creator.onElementGetActions.add((_, o) => {
    const el = o.element as Question;
    if (!el.isQuestion && !el.isPanel) return;
    o.actions.unshift({
      id: "clinician-only",
      title: "Clinician only",
      tooltip: "Only clinicians see this",
      showTitle: true,
      disableShrink: true,
      location: "start",
      enabled: false,
      visible: new ComputedUpdater(() => isClinicianOnly(el)) as unknown as boolean,
      innerCss: "sj-badge-clinician",
    });
    if (el.isQuestion && opts.questionActions) o.actions.unshift(...opts.questionActions(el));
  });

  return creator;
}

function assignChoiceIds(q: Question) {
  const choices = (q as unknown as { choices?: ItemValue[] }).choices;
  if (choices) assignMissingOptionIds(choices);
}

function setUpToolbox(creator: SurveyCreator) {
  const icons = new Map<string, string>();
  for (const item of creator.toolbox.items) {
    icons.set(item.name, item.iconName);
    for (const sub of item.items ?? []) {
      const inputType = (sub.json as { inputType?: string } | undefined)?.inputType;
      if (inputType) icons.set(`${item.name}:${inputType}`, sub.iconName);
    }
  }
  creator.toolbox.clearItems();
  for (const t of TOOLBOX) {
    creator.toolbox.addItem({ name: t.name, title: t.title, iconName: icons.get(t.iconFrom) ?? icons.get("text"), json: t.json });
  }
  creator.toolbox.showCategoryTitles = false;
}
