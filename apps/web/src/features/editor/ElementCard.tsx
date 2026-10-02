"use client";
// One element on the page canvas: a read-only lookalike of what the patient sees, with a
// PATIENT / CLINICIAN badge, disclosure chips and the "Displays when" box (STR-09, LOG-06).
// The selected card's title can be edited in place.
import { Fragment, useEffect, useState } from "react";
import { ActionIcon, Badge, Button, Menu, Text, TextInput, Tooltip } from "@mantine/core";
import {
  IconArrowDown, IconArrowUp, IconCalculator, IconCopy, IconDots, IconFileArrowRight, IconGitBranch, IconInfoCircle, IconLink,
  IconFolderMinus, IconFolderPlus, IconLock, IconPlus, IconRepeat, IconSignature, IconTrash, IconUpload,
} from "@tabler/icons-react";
import {
  bandsOf, calcFormOf, choiceDisplayOf, copyElement, countDisclosures, dateSettingsOf, describeLogic, enableIfToReadOnly,
  findElement, groupsToMoveInto, hasQuestionOutputs, isLocked, kindOf, moveElement, moveElementTo, moveOutOfGroup, numberSettingsOf, optionsOf, pageTitle, pagesOf, plainText,
  repeatSettingsOf, textFormatOf, textOf, updateElement, DATE_FORMATS, TEXT_FORMATS, type ChoiceJson, type ElementJson, type Part,
  type Target,
} from "@poised/clinical";
import { AddContentBar } from "./AddContentBar";
import { useEditor } from "./context";
import css from "./editor.module.css";

export function ElementList({ elements, target, pageElements }: { elements: ElementJson[]; target: Omit<Target, "index">; pageElements: ElementJson[] }) {
  return (
    <>
      {elements.map((el, i) => (
        <ElementCard key={el.name} el={el} index={i} count={elements.length} target={target} pageElements={pageElements} />
      ))}
    </>
  );
}

function ElementCard(props: { el: ElementJson; index: number; count: number; target: Omit<Target, "index">; pageElements: ElementJson[] }) {
  const { el, index, count, target, pageElements } = props;
  const ed = useEditor();
  const [dropHere, setDropHere] = useState(false);
  const kind = kindOf(el);
  const locked = isLocked(kind);
  const selected = ed.selected === el.name;
  const isGroup = kind === "group" || kind === "section";
  const shown = describeLogic(ed.doc, el.visibleIf);
  const required = describeLogic(ed.doc, el.requiredIf as string | undefined);
  const readOnlyWhen = describeLogic(ed.doc, enableIfToReadOnly(el.enableIf as string | undefined));
  const otherPages = pagesOf(ed.doc).map((p, i) => ({ p, i })).filter(({ p }) => p.name !== ed.pageName);
  const intoGroups = groupsToMoveInto(ed.doc, el.name);
  const parent = findElement(ed.doc, el.name)?.parent;

  const classes = [css.card];
  if (selected) classes.push(css.cardSelected);
  if (el.visibleIf) classes.push(css.conditional);
  if (kind === "section") classes.push(css.section);
  if (dropHere) classes.push(css.dropBefore);

  const act = (e: React.MouseEvent, fn: () => void) => {
    e.stopPropagation();
    fn();
  };
  const openLogic = (e: React.MouseEvent) => act(e, () => ed.openLogic(el.name));

  return (
    <div
      className={classes.join(" ")}
      data-testid={`card-${el.name}`}
      onClick={(e) => {
        e.stopPropagation();
        if (!locked) ed.select(el.name);
      }}
      draggable={!ed.readOnly}
      onDragStart={(e) => {
        e.stopPropagation();
        ed.setDragging(el.name);
      }}
      onDragEnd={() => ed.setDragging(null)}
      onDragOver={(e) => {
        if (!ed.dragging || ed.dragging === el.name) return;
        e.preventDefault();
        e.stopPropagation();
        setDropHere(true);
      }}
      onDragLeave={() => setDropHere(false)}
      onDrop={(e) => {
        e.preventDefault();
        e.stopPropagation();
        setDropHere(false);
        const dragged = ed.dragging;
        ed.setDragging(null);
        if (dragged && dragged !== el.name) ed.change((d) => moveElementTo(d, dragged, { ...target, index }));
      }}
    >
      <div className={css.cardTop}>
        <span style={{ display: "flex", gap: 6, alignItems: "center", flexWrap: "wrap" }}>
          {isGroup ? (
            <Badge size="sm" variant="light" color="gray" radius="sm">{kind === "section" ? "Section" : "Group"}</Badge>
          ) : el.clinicianOnly ? (
            <span className={css.badgeClinician}>Clinician</span>
          ) : (
            <span className={css.badgePatient}>Patient</span>
          )}
          {isGroup && el.clinicianOnly && <span className={css.badgeClinician}>Clinician</span>}
          {el.type === "paneldynamic" && (
            <Badge size="sm" variant="light" color="blue" radius="sm" leftSection={<IconRepeat size={10} />}>
              Repeats {rangeText(repeatSettingsOf(el).min, repeatSettingsOf(el).max)}
            </Badge>
          )}
          {locked && (
            <Badge size="sm" variant="outline" color="gray" radius="sm" leftSection={<IconLock size={10} />}>Locked</Badge>
          )}
        </span>
        {!ed.readOnly && (
          <span className={css.cardActions}>
            {!locked && (
              <>
                <Tooltip label="Copy">
                  <ActionIcon variant="subtle" color="gray" size="sm" aria-label="Copy" onClick={(e) => act(e, () => ed.change((d) => copyElement(d, el.name).doc))}>
                    <IconCopy size={15} />
                  </ActionIcon>
                </Tooltip>
                <Tooltip label="Move up">
                  <ActionIcon variant="subtle" color="gray" size="sm" aria-label="Move up" disabled={index === 0} onClick={(e) => act(e, () => ed.change((d) => moveElement(d, el.name, -1)))}>
                    <IconArrowUp size={15} />
                  </ActionIcon>
                </Tooltip>
                <Tooltip label="Move down">
                  <ActionIcon variant="subtle" color="gray" size="sm" aria-label="Move down" disabled={index === count - 1} onClick={(e) => act(e, () => ed.change((d) => moveElement(d, el.name, 1)))}>
                    <IconArrowDown size={15} />
                  </ActionIcon>
                </Tooltip>
                {(otherPages.length > 0 || intoGroups.length > 0 || parent) && (
                  <Menu position="bottom-end" withinPortal>
                    <Menu.Target>
                      <ActionIcon variant="subtle" color="gray" size="sm" aria-label="More actions" onClick={(e) => e.stopPropagation()}>
                        <IconDots size={15} />
                      </ActionIcon>
                    </Menu.Target>
                    <Menu.Dropdown onClick={(e) => e.stopPropagation()}>
                      {intoGroups.length > 0 && <Menu.Label>Move into</Menu.Label>}
                      {intoGroups.map((g) => (
                        <Menu.Item
                          key={g.name}
                          leftSection={<IconFolderPlus size={14} />}
                          onClick={() => ed.change((d) => moveElementTo(d, el.name, { page: ed.pageName, parent: g.name }))}
                        >
                          {textOf(g.title) || g.name} <Text span size="xs" c="dimmed">({g.patientPage ? "Section" : "Group"})</Text>
                        </Menu.Item>
                      ))}
                      {parent && (
                        <Menu.Item leftSection={<IconFolderMinus size={14} />} onClick={() => ed.change((d) => moveOutOfGroup(d, el.name))}>
                          Move out of “{textOf(parent.title) || parent.name}”
                        </Menu.Item>
                      )}
                      {otherPages.length > 0 && <Menu.Label>Move to page</Menu.Label>}
                      {otherPages.map(({ p, i }) => (
                        <Menu.Item key={p.name} leftSection={<IconFileArrowRight size={14} />} onClick={() => ed.change((d) => moveElementTo(d, el.name, { page: p.name }))}>
                          {pageTitle(p, i)}
                        </Menu.Item>
                      ))}
                    </Menu.Dropdown>
                  </Menu>
                )}
              </>
            )}
            <Tooltip label="Delete">
              <ActionIcon variant="subtle" color="red" size="sm" aria-label="Delete" onClick={(e) => act(e, () => ed.remove(el.name))}>
                <IconTrash size={15} />
              </ActionIcon>
            </Tooltip>
          </span>
        )}
      </div>

      {kind === "statement" ? (
        <>
          {el.visibleIf && <IconGitBranch size={16} className={css.branch} />}
          <div className={css.statementText} style={{ marginTop: el.visibleIf ? 8 : 0 }}>
            {plainText(textOf(el.html)).split("\n").map((line, i) => (
              <p key={i}>{line}</p>
            ))}
          </div>
        </>
      ) : kind === "bmi" ? (
        <BmiBody el={el} />
      ) : (
        <>
          <div className={css.title}>
            {el.visibleIf && <IconGitBranch size={16} className={css.branch} />}
            {selected && !ed.readOnly && kind !== "profile" ? (
              <InlineTitle el={el} />
            ) : (
              <span>
                {textOf(el.title) || el.name}
                {!isGroup && !["profile", "calculation"].includes(kind) && !el.isRequired && !el.requiredIf && <span className={css.optional}> (optional)</span>}
              </span>
            )}
          </div>
          {textOf(el.description) && <Text size="sm" className={css.description}>{textOf(el.description)}</Text>}
          {!isGroup && <Body el={el} />}
        </>
      )}

      {isGroup && <GroupBody el={el} pageElements={pageElements} target={target} />}

      {shown && (
        <div className={css.displaysWhen} onClick={openLogic}>
          <IconGitBranch size={15} className={css.branch} />
          <DisplaysWhen condition={shown} />
        </div>
      )}
      {required && (
        <div className={css.displaysWhen} onClick={openLogic}>
          <DisplaysWhen condition={required} label="Required when" />
        </div>
      )}
      {readOnlyWhen && (
        <div className={css.displaysWhen} onClick={openLogic}>
          <DisplaysWhen condition={readOnlyWhen} label="Read-only when" />
        </div>
      )}
    </div>
  );
}

const rangeText = (min: number, max: number | undefined) => (max ? `${min}–${max}` : min > 1 ? `${min}+` : "");

/** The selected card's title, edited in place; saved when it loses focus or on Enter. */
function InlineTitle({ el }: { el: ElementJson }) {
  const ed = useEditor();
  const [draft, setDraft] = useState(textOf(el.title));
  useEffect(() => setDraft(textOf(el.title)), [el.title]);
  const save = () => {
    const v = draft.trim();
    if (v && v !== textOf(el.title)) ed.change((d) => updateElement(d, el.name, { title: v }));
  };
  return (
    <TextInput
      aria-label="Question text on the card"
      variant="unstyled"
      value={draft}
      onChange={(e) => setDraft(e.currentTarget.value)}
      onBlur={save}
      onKeyDown={(e) => e.key === "Enter" && e.currentTarget.blur()}
      onClick={(e) => e.stopPropagation()}
      style={{ flex: 1 }}
      styles={{ input: { fontWeight: 700, fontSize: 14, minHeight: 0, height: "auto", borderBottom: "1px dashed var(--mantine-color-gray-5)" } }}
    />
  );
}

/** A condition in plain words (LOG-06), with a lead such as "Required when". */
export function DisplaysWhen({ condition, label = "Displays when", compact }: { condition: { parts: Part[] } | { raw: string }; label?: string; compact?: boolean }) {
  return (
    <span style={{ display: "inline-flex", gap: 5, flexWrap: "wrap", alignItems: "baseline" }}>
      {!compact && <b style={{ whiteSpace: "nowrap" }}>{label}</b>}
      {"raw" in condition ? (
        <span>{condition.raw}</span>
      ) : (
        condition.parts.map((p, i) =>
          p.strong ? (
            <b key={i} style={{ whiteSpace: "nowrap" }}>{p.text}</b>
          ) : (
            <span key={i}>{p.text}</span>
          ),
        )
      )}
    </span>
  );
}

/** The `🔗 n` chip. Shown only when there are disclosures (Lifebox DisclosureIndicator). */
function Chip({ count, onClick }: { count: number; onClick: () => void }) {
  if (count === 0) return null;
  return (
    <span
      className={css.chip}
      role="button"
      aria-label={`${count} disclosures`}
      onClick={(e) => {
        e.stopPropagation();
        onClick();
      }}
    >
      <IconLink size={12} /> {count}
    </span>
  );
}

/** An option's score (OPT-03) and its own condition (OPT-05). */
function OptionExtras({ o }: { o: ChoiceJson }) {
  return (
    <>
      {o.score !== undefined && <span className={css.score}>{o.score} pt{o.score === 1 ? "" : "s"}</span>}
      {o.visibleIf && (
        <Tooltip label="Shown only when a condition is met (see Logic)">
          <IconGitBranch size={14} className={css.branch} />
        </Tooltip>
      )}
    </>
  );
}

function Body({ el }: { el: ElementJson }) {
  const ed = useEditor();
  const kind = kindOf(el);
  const chipFor = (o: ChoiceJson) => <Chip count={o.clinicalOutputs?.length ?? 0} onClick={() => ed.openDisclosures(el.name, o.value)} />;
  const questionChip = hasQuestionOutputs(kind) && <Chip count={countDisclosures(el)} onClick={() => ed.openDisclosures(el.name)} />;
  const options = optionsOf(el);
  const listView = choiceDisplayOf(el) === "list";
  const other = el.showOtherItem ? (
    <div className={css.optionRow}>
      <span className={kind === "selectmany" ? css.checkboxBox : css.radioCircle} />
      <Text size="sm">{textOf(el.otherText) || "Other (please specify)"}</Text>
      <span className={css.fakeInput} style={{ width: 120, height: 22 }} />
    </div>
  ) : null;

  switch (kind) {
    case "yesno":
      return (
        <div className={css.control} style={{ display: "flex", gap: 14, flexWrap: "wrap" }}>
          {options.map((o) => (
            <span key={o.value} style={{ display: "inline-flex", gap: 6, alignItems: "center" }}>
              <span className={css.radioBlock}>{o.text}<span className={css.radioCircle} /></span>
              <OptionExtras o={o} />
              {chipFor(o)}
            </span>
          ))}
        </div>
      );
    case "selectone":
      return (
        <div className={css.control}>
          {listView && <ListBox placeholder={textOf(el.placeholder) || "Select…"} />}
          {options.map((o) => (
            <div key={o.value} className={css.optionRow}>
              {listView ? <Text size="sm">· {o.text}</Text> : <span className={css.radioBlock}>{o.text}<span className={css.radioCircle} /></span>}
              <OptionExtras o={o} />
              {chipFor(o)}
            </div>
          ))}
          {other}
        </div>
      );
    case "selectmany": {
      const normal = options.filter((o) => !o.isExclusive);
      const none = options.filter((o) => o.isExclusive);
      const row = (o: ChoiceJson) => (
        <div key={o.value} className={css.optionRow}>
          {listView ? <Text size="sm">·</Text> : <span className={css.checkboxBox} />}
          <Text size="sm">{o.text}</Text>
          <OptionExtras o={o} />
          {chipFor(o)}
        </div>
      );
      const min = el.minSelectedChoices as number | undefined;
      const max = el.maxSelectedChoices as number | undefined;
      return (
        <div className={css.control}>
          {listView && <ListBox placeholder={textOf(el.placeholder) || "Search and select…"} />}
          {el.showSelectAllItem ? (
            <div className={css.optionRow}>
              <span className={css.checkboxBox} />
              <Text size="sm" fs="italic">{textOf(el.selectAllText) || "Select all"}</Text>
            </div>
          ) : null}
          {normal.map(row)}
          {other}
          {none.length > 0 && (
            <Fragment>
              <Text size="sm" my={8}>or</Text>
              {none.map(row)}
            </Fragment>
          )}
          {(min || max) && (
            <Text size="xs" c="dimmed" mt={4}>Ticks: {min && max ? `${min}–${max}` : min ? `at least ${min}` : `at most ${max}`}</Text>
          )}
        </div>
      );
    }
    case "text": {
      const f = textFormatOf(el);
      return (
        <div className={css.control} style={{ display: "flex", alignItems: "center", gap: 6, flexWrap: "wrap" }}>
          <span className={el.type === "comment" ? css.fakeTextarea : css.fakeInput}>{placeholderOf(el)}</span>
          {f.format !== "none" && <Badge size="xs" variant="light" color="gray">{TEXT_FORMATS[f.format].label}</Badge>}
          {el.maxLength ? <Badge size="xs" variant="light" color="gray">max {String(el.maxLength)}</Badge> : null}
          {questionChip}
        </div>
      );
    }
    case "number": {
      const s = numberSettingsOf(el);
      const range = s.min !== undefined || s.max !== undefined ? `${s.min ?? "…"} to ${s.max ?? "…"}` : "";
      return (
        <div className={css.control} style={{ display: "flex", alignItems: "center", gap: 6, flexWrap: "wrap" }}>
          <span className={css.fakeInput} style={{ width: 110 }}>{placeholderOf(el)}</span>
          {s.unit && <Text size="sm" fw={600}>{s.unit}</Text>}
          {range && <Badge size="xs" variant="light" color="gray">{range}</Badge>}
          {s.decimals !== undefined && <Badge size="xs" variant="light" color="gray">{s.decimals === 0 ? "whole numbers" : `${s.decimals} dp`}</Badge>}
          {(s.warnMin !== undefined || s.warnMax !== undefined) && <Badge size="xs" variant="light" color="yellow">soft warning</Badge>}
          {questionChip}
        </div>
      );
    }
    case "date": {
      const s = dateSettingsOf(el);
      return (
        <div className={css.control}>
          <div style={{ display: "flex", alignItems: "center", gap: 6, flexWrap: "wrap" }}>
            <span className={css.fakeInput} style={{ width: 130, color: "var(--mantine-color-gray-6)", fontSize: 12, padding: "6px 8px" }}>
              {s.format === "dmy" ? "dd/mm/yyyy" : s.format === "my" ? "mm/yyyy" : "yyyy"}
            </span>
            {s.format !== "dmy" && <Badge size="xs" variant="light" color="gray">{DATE_FORMATS[s.format]}</Badge>}
            {s.range !== "any" && <Badge size="xs" variant="light" color="gray">{s.range} only</Badge>}
            {questionChip}
          </div>
          {s.multiple && (
            <Button size="xs" variant="default" mt="xs" leftSection={<IconPlus size={14} />} tabIndex={-1}>Add another date</Button>
          )}
        </div>
      );
    }
    case "rating": {
      const min = Number(el.rateMin ?? 1);
      const max = Number(el.rateMax ?? 5);
      const values: (number | string)[] = max - min <= 12 ? Array.from({ length: max - min + 1 }, (_, i) => min + i) : [min, "…", max];
      return (
        <div className={css.control}>
          <div style={{ display: "flex", gap: 4, flexWrap: "wrap", alignItems: "center" }}>
            {values.map((v, i) => (
              <span key={i} className={css.ratingBox}>{v}</span>
            ))}
            {questionChip}
          </div>
          <div style={{ display: "flex", justifyContent: "space-between", maxWidth: 34 * values.length }}>
            <Text size="xs" c="dimmed">{textOf(el.minRateDescription)}</Text>
            <Text size="xs" c="dimmed">{textOf(el.maxRateDescription)}</Text>
          </div>
        </div>
      );
    }
    case "grid":
      return <GridBody el={el} />;
    case "calculation": {
      const f = calcFormOf(el.expression as string | undefined, el.calcKind as string | undefined);
      const what = !el.expression
        ? "No formula yet. Choose one in Settings."
        : f.kind === "score"
          ? `Total of the scores of ${f.questions.length} question${f.questions.length === 1 ? "" : "s"}`
          : f.kind === "years"
            ? "Years since a date"
            : f.kind === "bmi"
              ? "BMI from height and weight"
              : String(el.expression);
      const bands = bandsOf(el);
      return (
        <div className={css.control}>
          <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
            <IconCalculator size={16} />
            <Text size="sm" ff={f.kind === "custom" ? "monospace" : undefined}>{what}</Text>
            {!bands.length && questionChip}
          </div>
          {bands.length > 0 && (
            <div style={{ display: "flex", gap: 6, flexWrap: "wrap", marginTop: 8 }}>
              {bands.map((b, i) => (
                <span key={b.id} style={{ display: "inline-flex", gap: 4, alignItems: "center" }}>
                  <Badge size="sm" variant="light" color={["green", "yellow", "red", "grape"][i % 4]}>
                    {b.label} {b.max !== undefined ? `≤ ${b.max}` : bands.length > 1 ? "above" : ""}
                  </Badge>
                  <Chip count={b.clinicalOutputs?.length ?? 0} onClick={() => ed.openDisclosures(el.name)} />
                </span>
              ))}
            </div>
          )}
        </div>
      );
    }
    case "upload":
      return (
        <div className={`${css.control} ${css.dropZone}`}>
          <IconUpload size={18} />
          <Text size="sm">Drag a file here or choose one</Text>
          <Text size="xs" c="dimmed">
            {el.acceptedTypes === "image/*" ? "Images" : el.acceptedTypes === "application/pdf" ? "PDFs" : el.acceptedTypes ? "Images and PDFs" : "Any file"}
            {el.maxSize ? `, up to ${Math.round(Number(el.maxSize) / 1048576)} MB` : ""}
            {el.allowMultiple ? ", several files" : ""}
          </Text>
        </div>
      );
    case "signature":
      return (
        <div className={`${css.control} ${css.dropZone}`} style={{ height: 80 }}>
          <IconSignature size={18} />
          <Text size="xs" c="dimmed">Sign here</Text>
        </div>
      );
    case "medication":
    case "admissions":
      return (
        <div className={`${css.control} ${css.greyPanel}`}>
          <Text size="sm" mb="xs">{kind === "medication" ? "No medication added yet" : "No admission added yet"}</Text>
          <Button size="xs" variant="default" leftSection={<IconPlus size={14} />} tabIndex={-1}>
            {kind === "medication" ? "Add medication" : "Add admission"}
          </Button>
        </div>
      );
    case "profile":
      return (
        <div className={css.control}>
          <div className={css.greyPanel} style={{ textAlign: "left" }}>
            {["Name", "Date of birth", "Email address", "Mobile phone number", "Gender"].map((f) => (
              <Text key={f} size="sm" fw={700}>{f}</Text>
            ))}
          </div>
          <Text size="xs" c="dimmed" mt="xs" style={{ display: "flex", gap: 6 }}>
            <IconInfoCircle size={14} style={{ flex: "none" }} />
            If any of these details are incorrect, please contact your hospital to make changes to your profile.
          </Text>
        </div>
      );
    default:
      return null;
  }
}

const placeholderOf = (el: ElementJson) =>
  textOf(el.placeholder) ? <Text span size="xs" c="dimmed" px={8} lh="28px">{textOf(el.placeholder)}</Text> : null;

function ListBox({ placeholder }: { placeholder: string }) {
  return (
    <div className={css.fakeInput} style={{ width: 240, display: "flex", alignItems: "center", justifyContent: "space-between", padding: "0 8px", marginBottom: 6 }}>
      <Text size="xs" c="dimmed">{placeholder}</Text>
      <Text size="xs" c="dimmed">▾</Text>
    </div>
  );
}

/** Rows by columns, with a disclosure chip in each cell that has disclosures (QT-10). */
function GridBody({ el }: { el: ElementJson }) {
  const ed = useEditor();
  const items = (list: unknown) =>
    (Array.isArray(list) ? list : []).map((c: { value: string; text: unknown } | string) =>
      typeof c === "string" ? { value: c, text: c } : { value: c.value, text: textOf(c.text) || c.value },
    );
  const cells = (el.cellOutputs ?? {}) as Record<string, Record<string, unknown[]>>;
  const columns = items(el.columns);
  return (
    <table className={`${css.control} ${css.grid}`}>
      <thead>
        <tr>
          <th />
          {columns.map((c) => (
            <th key={c.value}>{c.text}</th>
          ))}
        </tr>
      </thead>
      <tbody>
        {items(el.rows).map((r) => (
          <tr key={r.value}>
            <td>{r.text}</td>
            {columns.map((c) => (
              <td key={c.value} style={{ textAlign: "center" }}>
                <span className={css.radioCircle} style={{ display: "inline-block", verticalAlign: "middle" }} />{" "}
                <Chip count={cells[r.value]?.[c.value]?.length ?? 0} onClick={() => ed.openDisclosures(el.name)} />
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function BmiBody({ el }: { el: ElementJson }) {
  return (
    <>
      <Text fw={700} size="lg">{textOf(el.title) || "BMI Calculator"}</Text>
      <Text size="sm" mt={4}>
        {textOf(el.description) || "Please tell us your height and weight so we can calculate your Body Mass Index (BMI)"}
      </Text>
      {[["What is your height?", "cm"], ["What is your weight?", "kg"]].map(([label, unit]) => (
        <div key={unit} style={{ marginTop: 14 }}>
          <Text size="sm" fw={600}>{label}</Text>
          <span style={{ display: "inline-flex", alignItems: "center", gap: 6, marginTop: 4 }}>
            <span className={css.fakeInput} style={{ width: 70 }} /> <Text size="sm" fw={600}>{unit}</Text>
          </span>
        </div>
      ))}
    </>
  );
}

function GroupBody({ el, pageElements, target }: { el: ElementJson; pageElements: ElementJson[]; target: Omit<Target, "index"> }) {
  const ed = useEditor();
  const children = (el.type === "paneldynamic" ? el.templateElements : el.elements) ?? [];
  const selected = ed.selected === el.name;
  const inner: Omit<Target, "index"> = { page: target.page, parent: el.name };
  const body = (
    <div className={css.groupBody}>
      <ElementList elements={children} target={inner} pageElements={pageElements} />
      {children.length === 0 && (
        <div
          className={css.emptyGroup}
          onDragOver={(e) => {
            if (ed.dragging && ed.dragging !== el.name) {
              e.preventDefault();
              e.stopPropagation();
            }
          }}
          onDrop={(e) => {
            e.preventDefault();
            e.stopPropagation();
            const dragged = ed.dragging;
            ed.setDragging(null);
            if (dragged) ed.change((d) => moveElementTo(d, dragged, inner));
          }}
        >
          Add content or drag and drop existing content in to this {el.patientPage ? "Section" : "Group"}
        </div>
      )}
      {selected && <AddContentBar target={inner} pageElements={pageElements} inside={el} />}
      {el.type === "paneldynamic" && (
        <Button size="xs" variant="default" leftSection={<IconPlus size={14} />} tabIndex={-1} w="fit-content">
          {textOf(el.panelAddText) || "Add another"}
        </Button>
      )}
    </div>
  );
  return el.patientPage ? <div className={css.sectionBody} style={{ marginTop: 12 }}>{body}</div> : body;
}
