// Package lifebox converts a questionnaire exported from the Lifebox Author tool
// (tools/export-lifebox-hq.js) into our questionnaire: one SurveyJS JSON per Question Set.
// See docs/plan-redesign.md (step 7) for the mapping.
package lifebox

// Export is the file the export snippet downloads. Only the fields we use are listed;
// encoding/json matches the camelCase names case-insensitively.
type Export struct {
	Questionnaire struct {
		Name         string
		Description  string
		QuestionSets []QuestionSet
	}
	Categories []struct {
		ID   string
		Name string
	}
}

// Text is a Lifebox text record; we keep only the English text.
type Text struct {
	Text string
}

type QuestionSet struct {
	ID          string
	Name        string
	Description string
	Icon        string
	Pages       []Page
}

type Page struct {
	ID       string
	RenderIf *string
	Name     *Text
	Elements []Node
}

// Node is any page element. Typename says which fields apply.
type Node struct {
	Typename         string `json:"__typename"`
	ID               string
	Name             *Text
	Description      *Text
	Text             *Text // PanelStatement
	LabelTrue        *Text // QuestionBoolean
	LabelFalse       *Text
	OptionTrueID     string
	OptionFalseID    string
	IsRequired       bool
	IsClinical       bool
	RenderIf         *string
	VisibleIf        *string
	TextType         string // SINGLE_LINE | MULTI_LINE
	DateType         string // DAY_MONTH_YEAR | DAY_MONTH_YEAR_MULTIPLE
	MedicationType   string // PRESCRIBED | NON_PRESCRIBED | RECREATIONAL
	GroupSectionType *string
	ShowGroupHeading *bool
	QuestionOptions  []Option
	Rules            []Rule
	Elements         []Node
}

type Option struct {
	Typename string `json:"__typename"`
	ID       string
	Label    *Text
}

// Rule is a Lifebox disclosure: when its expression holds, it adds codes and a note.
type Rule struct {
	Expression            string
	DisclosureCodes       []Code
	EpisodeNoteCategoryID *string
	EpisodeNoteTemplate   *string
}

type Code struct {
	Code    string
	CodeSet string
	Synonym string
}

func (t *Text) String() string {
	if t == nil {
		return ""
	}
	return t.Text
}
