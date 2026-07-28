import * as stylex from "@stylexjs/stylex";

export const controlSize = {
  xs: {
    height: 24,
    minHeight: 24,
    paddingInline: 6,
    paddingBlock: 4,
    fontSize: 12,
    lineHeight: "16px",
    gap: 6,
    iconSize: 14,
  },
  sm: {
    height: 28,
    minHeight: 28,
    paddingInline: 8,
    paddingBlock: 6,
    fontSize: 12,
    lineHeight: "16px",
    gap: 6,
    iconSize: 16,
  },
  md: {
    height: 36,
    minHeight: 36,
    paddingInline: 12,
    paddingBlock: 8,
    fontSize: 14,
    lineHeight: "20px",
    gap: 8,
    iconSize: 16,
  },
  lg: {
    height: 40,
    minHeight: 40,
    paddingInline: 16,
    paddingBlock: 8,
    fontSize: 14,
    lineHeight: "20px",
    gap: 8,
    iconSize: 20,
  },
} as const;

export type ControlSize = keyof typeof controlSize;

interface ControlSizeOptions {
  gap?: boolean;
  paddingInline?: boolean;
}

const controlHeightStyles = stylex.create({
  xs: {
    height: 24,
  },
  sm: {
    height: 28,
  },
  md: {
    height: 36,
  },
  lg: {
    height: 40,
  },
});

const controlMinHeightStyles = stylex.create({
  xs: {
    minHeight: 24,
  },
  sm: {
    minHeight: 28,
  },
  md: {
    minHeight: 36,
  },
  lg: {
    minHeight: 40,
  },
});

const controlPaddingInlineStyles = stylex.create({
  xs: {
    paddingInline: 6,
  },
  sm: {
    paddingInline: 8,
  },
  md: {
    paddingInline: 12,
  },
  lg: {
    paddingInline: 16,
  },
});

const controlTextStyles = stylex.create({
  xs: {
    fontSize: 12,
    lineHeight: "16px",
  },
  sm: {
    fontSize: 12,
    lineHeight: "16px",
  },
  md: {
    fontSize: 14,
    lineHeight: "20px",
  },
  lg: {
    fontSize: 14,
    lineHeight: "20px",
  },
});

const controlGapStyles = stylex.create({
  xs: {
    columnGap: 6,
  },
  sm: {
    columnGap: 6,
  },
  md: {
    columnGap: 8,
  },
  lg: {
    columnGap: 8,
  },
});

const buttonGapStyles = stylex.create({
  xs: {
    columnGap: 4,
  },
  sm: {
    columnGap: 4,
  },
  md: {
    columnGap: 6,
  },
  lg: {
    columnGap: 6,
  },
});

const controlMultilinePaddingStyles = stylex.create({
  xs: {
    paddingBlock: 4,
    paddingInline: 6,
  },
  sm: {
    paddingBlock: 6,
    paddingInline: 8,
  },
  md: {
    paddingBlock: 8,
    paddingInline: 12,
  },
  lg: {
    paddingBlock: 8,
    paddingInline: 16,
  },
});

export function controlSizeStyle(
  size: ControlSize,
  { gap, paddingInline = true }: ControlSizeOptions = {}
) {
  return [
    controlHeightStyles[size],
    paddingInline && controlPaddingInlineStyles[size],
    controlTextStyles[size],
    gap && controlGapStyles[size],
  ];
}

export function buttonGapStyle(size: ControlSize) {
  return buttonGapStyles[size];
}

export function controlMinHeightStyle(
  size: ControlSize,
  { gap, paddingInline = true }: ControlSizeOptions = {}
) {
  return [
    controlMinHeightStyles[size],
    paddingInline && controlPaddingInlineStyles[size],
    controlTextStyles[size],
    gap && controlGapStyles[size],
  ];
}

export function controlMultilineSizeStyle(size: ControlSize) {
  return [controlMultilinePaddingStyles[size], controlTextStyles[size]];
}

const formStyles = stylex.create({
  controlGroup: {
    display: "flex",
    flexDirection: "column",
    rowGap: 8,
  },
  controlRow: {
    alignItems: "center",
    columnGap: 8,
    display: "flex",
    width: "100%",
  },
  field: {
    display: "flex",
    flexDirection: "column",
    rowGap: 6,
  },
  fieldGroup: {
    display: "flex",
    flexDirection: "column",
    rowGap: 24,
  },
  fieldHeader: {
    display: "flex",
    flexDirection: "column",
    rowGap: 0,
  },
  fieldTitle: {
    color: "rgb(var(--color-main))",
    fontSize: 16,
    fontWeight: 600,
    lineHeight: "24px",
  },
  fieldDescription: {
    color: "rgb(var(--color-control-placeholder))",
    fontSize: 14,
    lineHeight: "20px",
  },
  label: {
    alignItems: "center",
    color: "rgb(var(--color-control))",
    columnGap: 4,
    display: "inline-flex",
    fontSize: 14,
    fontWeight: 500,
    lineHeight: "20px",
  },
  error: {
    color: "rgb(var(--color-error))",
    fontSize: 12,
    lineHeight: "16px",
  },
  section: {
    paddingBlock: 24,
    rowGap: 16,
    "@media (min-width: 1024px)": {
      display: "flex",
      rowGap: 0,
    },
  },
  sectionContent: {
    flex: 1,
    marginTop: 16,
    "@media (min-width: 1024px)": {
      marginTop: 0,
      paddingInline: 16,
    },
  },
  sectionHeader: {
    textAlign: "left",
    "@media (min-width: 1024px)": {
      width: "25%",
    },
  },
  sectionTitle: {
    color: "rgb(var(--color-main))",
    fontSize: 24,
    fontWeight: 700,
    lineHeight: "32px",
  },
});

export function formFieldStyle() {
  return formStyles.field;
}

export function formLabelStyle() {
  return formStyles.label;
}

export function formErrorStyle() {
  return formStyles.error;
}

export function formControlGroupStyle() {
  return formStyles.controlGroup;
}

export function formControlRowStyle() {
  return formStyles.controlRow;
}

export function formFieldGroupStyle() {
  return formStyles.fieldGroup;
}

export function formFieldHeaderStyle() {
  return formStyles.fieldHeader;
}

export function formFieldTitleStyle() {
  return formStyles.fieldTitle;
}

export function formFieldDescriptionStyle() {
  return formStyles.fieldDescription;
}

export function formSectionStyle() {
  return formStyles.section;
}

export function formSectionHeaderStyle() {
  return formStyles.sectionHeader;
}

export function formSectionTitleStyle() {
  return formStyles.sectionTitle;
}

export function formSectionContentStyle() {
  return formStyles.sectionContent;
}

const stickyActionFooterStyles = stylex.create({
  root: {
    width: "100%",
  },
  content: {
    alignItems: "center",
    display: "flex",
    justifyContent: "space-between",
  },
  side: {
    alignItems: "center",
    display: "flex",
  },
  right: {
    columnGap: 8,
  },
});

export function stickyActionFooterStyle() {
  return stickyActionFooterStyles.root;
}

export function stickyActionFooterContentStyle() {
  return stickyActionFooterStyles.content;
}

export function stickyActionFooterSideStyle() {
  return stickyActionFooterStyles.side;
}

export function stickyActionFooterRightStyle() {
  return stickyActionFooterStyles.right;
}

export const overlaySurfaceClassName = [
  "max-h-60",
  "overflow-y-auto",
  "overflow-x-hidden",
  "rounded-sm",
  "border",
  "border-control-border",
  "bg-background",
  "py-1",
  "shadow-md",
  "focus:outline-hidden",
].join(" ");

const rowBaseStyles = stylex.create({
  base: {
    alignItems: "center",
    color: "rgb(var(--color-control))",
    cursor: "pointer",
    display: "flex",
    lineHeight: "20px",
    minWidth: 0,
    position: "relative",
    textAlign: "start",
    transitionDuration: "150ms",
    transitionProperty:
      "color, background-color, border-color, text-decoration-color, fill, stroke",
    transitionTimingFunction: "cubic-bezier(0.4, 0, 0.2, 1)",
    userSelect: "none",
    width: "100%",
    ":disabled": {
      cursor: "not-allowed",
      opacity: 0.5,
    },
    ":focus": {
      backgroundColor: "rgb(var(--color-control-bg))",
      outlineStyle: "none",
    },
    ":hover": {
      backgroundColor: "rgb(var(--color-control-bg))",
    },
  },
});

export const menuRowStateClassName = [
  "data-highlighted:bg-control-bg",
  "data-selected:bg-accent/5",
  "aria-selected:bg-accent/5",
  "disabled:pointer-events-none",
  "disabled:opacity-50",
  "aria-disabled:pointer-events-none",
  "aria-disabled:opacity-50",
  "data-disabled:pointer-events-none",
  "data-disabled:opacity-50",
].join(" ");

export const listRowStateClassName = [
  "hover:bg-control-bg",
  "data-selected:bg-accent/5",
  "aria-selected:bg-accent/5",
  "disabled:pointer-events-none",
  "disabled:opacity-50",
  "aria-disabled:pointer-events-none",
  "aria-disabled:opacity-50",
  "data-disabled:pointer-events-none",
  "data-disabled:opacity-50",
].join(" ");

const rowSizeStyles = stylex.create({
  sm: {
    columnGap: 8,
    fontSize: 14,
    minHeight: 32,
    paddingBlock: 6,
    paddingInline: 8,
  },
  md: {
    columnGap: 8,
    fontSize: 14,
    minHeight: 36,
    paddingBlock: 8,
    paddingInline: 12,
  },
});

const listRowStyles = stylex.create({
  base: {
    textAlign: "start",
    width: "100%",
  },
  icon: {
    color: "rgb(var(--color-control-light))",
    flexShrink: 0,
    height: 16,
    width: 16,
  },
  primaryText: {
    color: "rgb(var(--color-control))",
    minWidth: 0,
  },
  secondaryText: {
    color: "rgb(var(--color-control-light))",
    fontSize: 12,
    lineHeight: "16px",
    minWidth: 0,
  },
});

export type RowSize = keyof typeof rowSizeStyles;

export function menuRowStyle(size: RowSize = "sm") {
  return [rowBaseStyles.base, rowSizeStyles[size]];
}

export function listRowStyle(size: RowSize = "sm") {
  return [rowBaseStyles.base, rowSizeStyles[size], listRowStyles.base];
}

export function listRowIconStyle() {
  return listRowStyles.icon;
}

export function listRowPrimaryTextStyle() {
  return listRowStyles.primaryText;
}

export function listRowSecondaryTextStyle() {
  return listRowStyles.secondaryText;
}

export function interactiveRowStyle(size: RowSize = "sm") {
  return listRowStyle(size);
}
