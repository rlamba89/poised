// How SurveyJS draws an Information block (S35, INF-03), and every other `html` element: old
// statements, the Profile block and the clinician's Clinical Summary lists. It replaces SurveyJS's
// own drawing, which puts the HTML on the page as it is: here it always goes through
// `informationHtml` (@poised/clinical, information.ts), which turns the author's marks into HTML and
// then cleans it (only a few safe tags; links to https, mailto and tel; pictures from https). So no
// script, event attribute or javascript: link reaches a patient's or clinician's screen, whoever
// wrote the JSON. Importing this file registers it, so every page that shows a survey imports it once.
import { createElement } from "react";
import { ReactQuestionFactory, SurveyQuestionElementBase } from "survey-react-ui";
import type { QuestionHtmlModel } from "survey-core";
import { informationHtml } from "@poised/clinical";

export class SurveyQuestionInformation extends SurveyQuestionElementBase {
  protected get question(): QuestionHtmlModel {
    return this.questionBase as unknown as QuestionHtmlModel;
  }

  // Redraw when the text changes (a Clinical Summary's list) or the survey's language does.
  componentDidMount() {
    super.componentDidMount();
    this.watch();
  }
  componentDidUpdate(prevProps: unknown, prevState: unknown) {
    super.componentDidUpdate(prevProps, prevState);
    this.watch();
  }
  componentWillUnmount() {
    super.componentWillUnmount();
    this.question.locHtml.onChanged = () => {};
    const md = this.question.getLocalizableString("markdown");
    if (md) md.onChanged = () => {};
  }
  private watch() {
    const redraw = () => this.setState({ changed: ((this.state as { changed?: number } | null)?.changed ?? 0) + 1 });
    this.question.locHtml.onChanged = redraw;
    const md = this.question.getLocalizableString("markdown");
    if (md) md.onChanged = redraw;
  }

  protected renderElement() {
    const html = informationHtml(this.question);
    if (!html) return null;
    return <div className={["sj-info", this.question.renderCssRoot].filter(Boolean).join(" ")} dangerouslySetInnerHTML={{ __html: html }} />;
  }
}

ReactQuestionFactory.Instance.registerQuestion("html", (props) => createElement(SurveyQuestionInformation, props));
