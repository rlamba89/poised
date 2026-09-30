"use client";
// Throwaway spike component. Spike 1: bare Creator. Spike 2: clinicalOutputs modal,
// autosave, undo/redo. Spike 3: generated stable IDs on add and copy.
import { useState } from "react";
import { Base, ComputedUpdater, ItemValue, Question, Serializer } from "survey-core";

import { SurveyCreator, SurveyCreatorComponent } from "survey-creator-react";
import "survey-core/survey-core.css";
import "survey-creator-core/survey-creator-core.css";

type Output = { id: string; note: { text: string; category: string } };

// ?fix=1 holds outputs in an immutable { items } box (JSON stays a plain array).
// Without it, survey-core overwrites an array value in place on questions, which breaks redo.
const FIX = typeof window !== "undefined" && new URLSearchParams(window.location.search).get("fix") === "1";

let registered = false;
function registerClinicalProperties() {
  if (registered) return;
  registered = true;
  for (const cls of ["question", "itemvalue"]) {
    if (!FIX) {
      Serializer.addProperty(cls, { name: "clinicalOutputs", visible: false, isLocalizable: false });
      continue;
    }
    Serializer.addProperty(cls, {
      // Non-string type: the undo manager merges rapid edits to string properties into one step.
      name: "clinicalOutputs:clinicaloutputs",
      visible: false,
      isLocalizable: false,
      onSerializeValue: (obj: Base) => obj.getPropertyValue("clinicalOutputs")?.items,
      onSetValue: (obj: Base, value: any) =>
        obj.setPropertyValue("clinicalOutputs", Array.isArray(value) ? { items: value } : value),
    });
  }
}

function setOutputs(obj: Base, items: Output[]) {
  obj.setPropertyValue("clinicalOutputs", FIX ? (items.length ? { items } : undefined) : items);
}

function shortId(prefix: string, len: number) {
  const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789";
  const bytes = crypto.getRandomValues(new Uint8Array(len));
  return prefix + "_" + Array.from(bytes, (b) => alphabet[b % alphabet.length]).join("");
}

const outputsOf = (obj: Base): Output[] =>
  (FIX ? obj.getPropertyValue("clinicalOutputs")?.items : obj.getPropertyValue("clinicalOutputs")) ?? [];

function optionsOf(q: Question): ItemValue[] {
  const anyQ = q as any;
  const items: ItemValue[] = [...(anyQ.choices ?? [])];
  if (anyQ.showNoneItem && anyQ.noneItem) items.push(anyQ.noneItem);
  return items;
}

const countFor = (q: Question) =>
  outputsOf(q).length + optionsOf(q).reduce((n, item) => n + outputsOf(item).length, 0);

const spike2Json = {
  pages: [
    {
      name: "p1",
      elements: [
        {
          type: "radiogroup",
          name: "q_smoke1",
          title: "Do you smoke?",
          choices: [
            { value: "o_yes1", text: "Yes" },
            { value: "o_no01", text: "No" },
          ],
        },
        {
          type: "text",
          inputType: "number",
          name: "q_perday",
          title: "How many per day?",
          visibleIf: "{q_smoke1} = 'o_yes1'",
        },
        {
          type: "checkbox",
          name: "q_condit",
          title: "Which conditions?",
          choices: [
            { value: "o_asth", text: "Asthma" },
            { value: "o_none", text: "None of these", isExclusive: true },
          ],
        },
      ],
    },
  ],
};

function makeCreator(spike: number, openModal: (q: Question) => void) {
  if (spike >= 2) registerClinicalProperties();

  const creator = new SurveyCreator({
    showLogicTab: true,
    showJSONEditorTab: false,
    showPreviewTab: false,
    showTranslationTab: false,
    showThemeTab: false,
    autoSaveEnabled: spike === 2,
    useElementTitles: spike >= 2,
  });

  const events: any[] = [];
  const saves: any[] = [];
  creator.onModified.add((_, o: any) => events.push({ ev: "modified", type: o.type, name: o.name }));

  if (spike === 2) {
    creator.autoSaveDelay = 300;
    creator.saveSurveyFunc = (saveNo: number, callback: (no: number, ok: boolean) => void) => {
      saves.push({ saveNo, json: creator.JSON });
      setTimeout(() => callback(saveNo, true), 100);
    };
    creator.onElementGetActions.add((_, options) => {
      const el = options.element as any;
      if (!el.isQuestion) return;
      options.actions.push({
        id: "clinical-outputs",
        title: new ComputedUpdater(() => `Outputs (${countFor(el)})`) as any,
        showTitle: true,
        disableShrink: true,
        location: "start",
        action: () => openModal(el),
      });
    });
    creator.JSON = spike2Json;
  }

  if (spike === 3) {
    creator.onQuestionAdded.add((_, o) => {
      const q = o.question;
      const before = { name: q.name, title: q.title, titleEmpty: q.locTitle.isEmpty };
      // The displayed title falls back to `name`, so pin it before renaming.
      if (q.locTitle.isEmpty) q.title = q.name;
      q.name = shortId("q", 6);
      // Default choices created with the question do not raise onItemValueAdded.
      if (o.reason !== "ELEMENT_COPIED") {
        for (const item of optionsOf(q)) {
          if (item === (q as any).noneItem) continue;
          if (!item.hasText) item.text = String(item.value);
          item.value = shortId("o", 4);
        }
      }
      events.push({ ev: "questionAdded", reason: o.reason, before, after: { name: q.name, title: q.title } });
    });
    creator.onPanelAdded.add((_, o) => {
      const p = o.panel;
      if (p.locTitle.isEmpty) p.title = p.name;
      const before = p.name;
      p.name = shortId("g", 6);
      events.push({ ev: "panelAdded", reason: o.reason, before, after: { name: p.name, title: p.title } });
    });
    creator.onItemValueAdded.add((_, o) => {
      const item = o.newItem;
      const before = { value: item.value, text: item.text, hasText: item.hasText };
      if (!item.hasText) item.text = String(item.value);
      item.value = shortId("o", 4);
      events.push({ ev: "itemValueAdded", prop: o.propertyName, before, after: { value: item.value, text: item.text } });
    });
  }

  (window as any).__spike = { creator, events, saves, outputsOf, setOutputs, countFor, optionsOf, FIX };
  return creator;
}

export default function SpikeCreator({ spike }: { spike: number }) {
  const [target, setTarget] = useState<Question | null>(null);
  const [, rerender] = useState(0);
  const [creator] = useState(() => makeCreator(spike, setTarget));

  const addOutput = (obj: Base, text: string) => {
    const next: Output[] = [...outputsOf(obj), { id: shortId("out", 4), note: { text, category: "Lifestyle" } }];
    setOutputs(obj, next);
    rerender((n) => n + 1);
  };
  const removeLast = (obj: Base) => {
    setOutputs(obj, outputsOf(obj).slice(0, -1));
    rerender((n) => n + 1);
  };

  return (
    <div style={{ height: "100vh" }}>
      <SurveyCreatorComponent creator={creator} />
      {target && (
        <div
          role="dialog"
          data-testid="outputs-modal"
          style={{ position: "fixed", inset: "10% 20%", background: "white", border: "2px solid #333", padding: 16, zIndex: 1000, overflow: "auto" }}
        >
          <h3>Outputs for “{target.title}”</h3>
          {[{ label: "Question", obj: target as Base }, ...optionsOf(target).map((i) => ({ label: `Option “${i.text}” (${i.value})`, obj: i as Base }))].map(
            (row, idx) => (
              <div key={idx} data-testid={`row-${idx}`} style={{ borderTop: "1px solid #ccc", padding: 8 }}>
                <strong>{row.label}</strong> — <span data-testid={`count-${idx}`}>{outputsOf(row.obj).length}</span> output(s)
                <ul>{outputsOf(row.obj).map((o) => <li key={o.id}>{o.note.text}</li>)}</ul>
                <button data-testid={`add-${idx}`} onClick={() => addOutput(row.obj, `Output ${outputsOf(row.obj).length + 1}`)}>Add output</button>{" "}
                <button data-testid={`remove-${idx}`} onClick={() => removeLast(row.obj)}>Remove last</button>
              </div>
            ),
          )}
          <button data-testid="close-modal" onClick={() => setTarget(null)}>Close</button>
        </div>
      )}
    </div>
  );
}
