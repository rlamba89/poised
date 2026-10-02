"use client";
// "Add content:" buttons (Lifebox catalogue, plus Number from DSG-01 and the SurveyJS types).
import { Button, Text } from "@mantine/core";
import {
  IconBuildingHospital, IconCalculator, IconCalendar, IconCircleDot, IconGridDots, IconLayoutList, IconLetterT, IconMessage,
  IconNumbers, IconPill, IconScale, IconSeparatorHorizontal, IconSignature, IconSquareCheck, IconStar, IconToggleLeft, IconUpload,
  IconUser, type Icon,
} from "@tabler/icons-react";
import { KINDS, KIND_LABEL, kindOf, type ElementJson, type Kind, type Target } from "@poised/clinical";
import { useEditor } from "./context";
import css from "./editor.module.css";

export const KIND_ICON: Record<Kind, Icon> = {
  group: IconLayoutList,
  section: IconSeparatorHorizontal,
  yesno: IconToggleLeft,
  selectone: IconCircleDot,
  selectmany: IconSquareCheck,
  text: IconLetterT,
  date: IconCalendar,
  number: IconNumbers,
  rating: IconStar,
  grid: IconGridDots,
  calculation: IconCalculator,
  upload: IconUpload,
  signature: IconSignature,
  medication: IconPill,
  admissions: IconBuildingHospital,
  bmi: IconScale,
  profile: IconUser,
  statement: IconMessage,
};

/**
 * Lifebox rules: a Section can't go inside a group or section; BMI must be alone on its
 * page, so it is offered only on an empty page and, once added, nothing else is.
 */
export function AddContentBar({ target, pageElements, inside }: { target: Target; pageElements: ElementJson[]; inside?: ElementJson }) {
  const { add, readOnly } = useEditor();
  if (readOnly) return null;
  const pageHasBmi = pageElements.some((el) => kindOf(el) === "bmi");
  const enabled = (k: Kind) => {
    if (pageHasBmi) return false;
    if (k === "bmi") return pageElements.length === 0;
    if (k === "section") return !inside;
    return true;
  };
  return (
    <div className={inside ? undefined : css.addBar}>
      <Text size="sm" fw={600}>Add content:</Text>
      <div className={css.addButtons}>
        {KINDS.map((k) => {
          const Icon = KIND_ICON[k];
          return (
            <Button
              key={k}
              size="xs"
              variant="default"
              leftSection={<Icon size={14} />}
              disabled={!enabled(k)}
              onClick={(e) => {
                e.stopPropagation();
                add(k, target);
              }}
            >
              {KIND_LABEL[k]}
            </Button>
          );
        })}
      </div>
    </div>
  );
}
