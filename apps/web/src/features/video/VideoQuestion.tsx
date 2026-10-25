// How a Video element (S34, INF-01/02) is drawn in a survey: its length, the video site's own player
// (with its own captions), and the "I have watched this video" tick when the author asked for one.
// The rules (the link check, the player's address, required, read-only for clinicians) are in
// @poised/clinical (video.ts); this only draws them. Importing this file registers it with SurveyJS,
// so every page that shows a survey imports it once.
//
// The player is an iframe whose address is built only from a recognised YouTube or Vimeo video ID:
// youtube-nocookie.com, or Vimeo with dnt=1. It sends the page's origin as the referrer even where
// the page itself says no-referrer (QA's web pages do), because YouTube's player needs to know which
// site embeds it (YouTube's "Required minimum functionality", which recommends exactly this policy).
import { createElement } from "react";
import { ReactQuestionFactory, SurveyQuestionElementBase } from "survey-react-ui";
import type { QuestionVideoModel } from "@poised/clinical";

export class SurveyQuestionVideo extends SurveyQuestionElementBase {
  protected get question(): QuestionVideoModel {
    return this.questionBase as unknown as QuestionVideoModel;
  }

  protected renderElement() {
    const q = this.question;
    const src = q.embedUrl;
    const ticked = q.value === true;
    return (
      <div className="sj-video" data-testid={`video-${q.name}`}>
        {q.lengthText && <p className="sj-video-length">Length {q.lengthText}</p>}
        {src ? (
          <div className="sj-video-frame">
            <iframe
              src={src}
              title={q.title}
              loading="lazy"
              allow="encrypted-media; picture-in-picture; fullscreen"
              allowFullScreen
              referrerPolicy="strict-origin-when-cross-origin"
            />
          </div>
        ) : (
          <p className="sj-video-missing">This video can't be shown.</p>
        )}
        {q.watchTick && (
          <label className="sj-video-tick">
            <input
              type="checkbox"
              checked={ticked}
              disabled={q.isReadOnly}
              aria-required={q.isRequired && !q.forClinician}
              // Unticking clears the answer, so a required tick is missing again.
              onChange={(e) => {
                q.value = e.currentTarget.checked ? true : undefined;
              }}
            />
            {/* Its own words, in the survey's language (the editor stores them when the tick is turned on). */}
            <span>{q.locTickText.isEmpty ? q.tickText : this.renderLocString(q.locTickText)}</span>
          </label>
        )}
        {q.watchTick && q.forClinician && (
          <p className="sj-video-ticked" data-testid={`video-ticked-${q.name}`}>
            {ticked ? "The patient ticked this." : "The patient hasn't ticked this."}
          </p>
        )}
      </div>
    );
  }
}

ReactQuestionFactory.Instance.registerQuestion("video", (props) => createElement(SurveyQuestionVideo, props));
