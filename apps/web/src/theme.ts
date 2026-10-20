import { createTheme, type MantineColorsTuple } from "@mantine/core";

// Preopsis's teal (docs/brand/). Shade 7, #0f766e, is the logo's colour and the primary one:
// white text on it has a contrast ratio of about 5.5:1 (WCAG AA).
const brand: MantineColorsTuple = [
  "#f0fdfa", "#ccfbf1", "#99f6e4", "#5eead4", "#2dd4bf",
  "#14b8a6", "#0d9488", "#0f766e", "#115e59", "#134e4a",
];

export const theme = createTheme({
  colors: { brand },
  primaryColor: "brand",
  primaryShade: 7,
});
