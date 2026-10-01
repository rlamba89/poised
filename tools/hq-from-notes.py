#!/usr/bin/env python3
"""Turns docs/hje-full-hq-content.md (transcribed from screenshots) into the same JSON shape
the Lifebox export snippet downloads, so `go run ./cmd/import-lifebox` can load it.

Disclosures can't be transcribed (the screenshots only show their count), so none are included.

    python3 tools/hq-from-notes.py docs/hje-full-hq-content.md > /tmp/hje-from-notes.json
"""
import json
import re
import sys
import uuid

TYPES = {
    "Yes/No": "QuestionBoolean",
    "Select One": "QuestionRadio",
    "Select Many": "QuestionCheckbox",
    "Text short": "QuestionText",
    "Text long": "QuestionText",
    "Date": "QuestionDate",
    "Statement": "PanelStatement",
    "Profile": "QuestionDemographic",
    "BMI": "QuestionBMI",
    "Admissions": "QuestionHospitalAdmission",
    "Medication": "QuestionMedication",
}

ITEM = re.compile(
    r'^(\d+)\. (Yes/No|Select One|Select Many|Text short|Text long|Date|Statement|Profile|BMI|Admissions|Medication(?: \((?:prescribed|non-prescribed)\))?)'
    r'\s*(?:\*\*)?\[(P|C)\](?:\*\*)?\s*(opt\s*)?(?:"(.*)")?'
)
WHEN = re.compile(r'when (?:(\d+\.\d+) )?Q(\d+) is (not )?"?([^"]+?)"?$')


def text(t):
    return {"text": t}


def new_id():
    return str(uuid.uuid4())


def parse(md):
    sets, set_, page, item = [], None, None, None
    pages_by_number = {}
    pending_page_conditions = []  # (page, "4.2", q, isNot, label)
    for raw in md.splitlines():
        line = raw.rstrip()
        m = re.match(r"^## (\d+)\. (.+?)(?: \(icon: (\w+)\))?$", line)
        if m:
            set_ = {"id": new_id(), "name": m.group(2), "description": "", "icon": m.group(3) or "Folder", "pages": []}
            sets.append(set_)
            continue
        m = re.match(r"^### (\d+\.\d+) (.+?)(?: \(page shown when (\d+\.\d+) Q(\d+) is (.+)\))?$", line)
        if m and set_ is not None:
            page = {"id": new_id(), "name": text(m.group(2)), "renderIf": None, "elements": [], "_items": {}}
            page["elements"].append({"__typename": "ClinicalPageSummary", "id": new_id(), "isClinical": True})
            set_["pages"].append(page)
            pages_by_number[m.group(1)] = page
            if m.group(3):
                pending_page_conditions.append((page, m.group(3), int(m.group(4)), False, m.group(5)))
            continue
        if page is None:
            continue
        m = ITEM.match(line)
        if m:
            kind = m.group(2)
            base = "Medication" if kind.startswith("Medication") else kind
            item = {
                "__typename": TYPES[base],
                "id": new_id(),
                "isClinical": m.group(3) == "C",
                "isRequired": not m.group(4) and base not in ("Statement", "Profile"),
                "_options": [],
                "_kind": kind,
            }
            title = m.group(5) or ""
            if base == "Statement":
                item["text"] = text(title.replace("\\n", "\n"))
            else:
                item["name"] = text(title or {"BMI": "BMI Calculator", "Profile": "Your Profile"}.get(base, ""))
            if kind == "Text long":
                item["textType"] = "MULTI_LINE"
            elif kind == "Text short":
                item["textType"] = "SINGLE_LINE"
            if base == "Date":
                item["dateType"] = "DAY_MONTH_YEAR"
            if base == "Medication":
                item["medicationType"] = "NON_PRESCRIBED" if "non-prescribed" in kind else "PRESCRIBED"
            if "same text as" in line:
                ref = re.search(r"same text as (\d+\.\d+) Q(\d+)", line)
                src = pages_by_number[ref.group(1)]["_items"][int(ref.group(2))]
                item["text"] = src["text"]
            page["elements"].append(item)
            page["_items"][int(m.group(1))] = item
            continue
        s = line.strip()
        if item is None or not s.startswith("- "):
            continue
        s = re.sub(r"\s*\([^)]*\)$", "", s[2:]) if not s[2:].startswith("desc:") else s[2:]  # drop trailing notes like "(stacked, ...)"
        if s.startswith("desc:"):
            d = s[5:].strip()
            if d.startswith("same as Q"):
                d = page["_items"][int(d[len("same as Q"):])]["description"]["text"]
            else:
                d = d.strip('"')
            item["description"] = text(d)
        elif s.startswith("when "):
            item["_when"] = s
        elif s.startswith("or: "):
            item["_options"].append(("None", s[4:].replace(" 🔗", "")))
        elif item["_kind"] in ("Yes/No", "Select One") and re.match(r"^[^,]+(, [^,]+)*$", s) and "🔗" in s or s in ("Yes, No", "Yes, No 🔗"):
            for part in s.split(", "):
                item["_options"].append(("String", part.replace(" 🔗", "").strip()))
        elif item["_kind"] in ("Select One", "Select Many"):
            item["_options"].append(("String", s.replace(" 🔗", "").strip()))
        elif s.startswith("The wording") or s.startswith("Yes"):
            pass

    # Options and logic.
    for set_ in sets:
        for page in set_["pages"]:
            for item in page["_items"].values():
                if item["__typename"] == "QuestionBoolean":
                    item["optionTrueId"], item["optionFalseId"] = new_id(), new_id()
                    item["labelTrue"], item["labelFalse"] = text("Yes"), text("No")
                    item["_byLabel"] = {"Yes": item["optionTrueId"], "No": item["optionFalseId"]}
                elif item["__typename"] in ("QuestionRadio", "QuestionCheckbox"):
                    opts = item["_options"] or [("String", "Yes"), ("String", "No")]
                    item["questionOptions"] = []
                    item["_byLabel"] = {}
                    for kind, label in opts:
                        oid = new_id()
                        item["questionOptions"].append({"__typename": "QuestionOption" + kind, "id": oid, "label": text(label)})
                        item["_byLabel"][label] = oid
            for item in page["_items"].values():
                when = item.pop("_when", None)
                if when:
                    m = WHEN.search(when)
                    src = page["_items"][int(m.group(2))]
                    item["visibleIf"] = condition(src, m.group(3), m.group(4))
    for page, page_no, q, is_not, label in pending_page_conditions:
        src = pages_by_number[page_no]["_items"][q]
        page["renderIf"] = condition(src, is_not, label)
    for set_ in sets:
        for page in set_["pages"]:
            page.pop("_items")
            for el in page["elements"]:
                for k in [k for k in el if k.startswith("_") and not k.startswith("__")]:
                    el.pop(k)
    return sets


def condition(src, is_not, label):
    oid = src["_byLabel"][label.strip()]
    return ("!" if is_not else "") + "containsAny(selectedOptionIds, [{" + oid + "}])"


if __name__ == "__main__":
    md = open(sys.argv[1]).read()
    sets = parse(md)
    json.dump({"questionnaire": {"name": "HJE Full HQ", "description": "Transcribed from screenshots of the Lifebox Training system (sets 1-4).", "questionSets": sets}, "categories": []}, sys.stdout, indent=1)
