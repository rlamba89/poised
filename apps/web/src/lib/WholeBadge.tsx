// A badge that always shows its whole text (S33, shared since S36). Mantine's Badge hides what
// doesn't fit (overflow hidden), which lets a narrow column squeeze it to "MEASUREME…" or "COMPL…"
// at phone width (S33 test reports 1 and 2; the patient's set list, S34–S35); with overflow
// visible and no shrinking, the row grows or wraps to fit it instead.
import { Badge, type BadgeProps } from "@mantine/core";

export function WholeBadge(props: BadgeProps & React.ComponentPropsWithoutRef<"div">) {
  return <Badge {...props} style={{ flexShrink: 0, overflow: "visible", ...props.style }} styles={{ label: { overflow: "visible" } }} />;
}
