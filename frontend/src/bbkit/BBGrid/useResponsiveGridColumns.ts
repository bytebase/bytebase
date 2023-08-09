import { useStyleTag } from "@vueuse/core";
import { uniqueId } from "lodash-es";
import { computed, onUnmounted, unref } from "vue";
import { MaybeRef } from "@/types";

const breakpoints = {
  default: "0",
  sm: "640px",
  md: "768px",
  lg: "1024px",
  xl: "1280px",
  "2xl": "1536px",
  "3xl": "1920px",
  "4xl": "2160px",
} as const;

export type ScreenSize = keyof typeof breakpoints;
export type ColumnWidth = string | Partial<{ [S in ScreenSize]: string }>;

const screenSizes = Object.keys(breakpoints) as ScreenSize[];

export const useResponsiveGridColumns = (widths: MaybeRef<ColumnWidth[]>) => {
  const className = `${uniqueId("bb-responsive-grid-columns-")}`;
  const css = computed(() => {
    const _widths = unref(widths);
    const rules = screenSizes
      .map((screen) => {
        if (_widths.some((width) => isDefinedScreen(width, screen))) {
          return generateCSS(className, _widths, screen);
        }
        return "";
      })
      .join("");

    return rules;
  });
  const tag = useStyleTag(css);
  onUnmounted(() => tag.unload());
  return className;
};

const isDefinedScreen = (width: ColumnWidth, screen: ScreenSize) => {
  if (typeof width === "string") {
    return screen === "default";
  }

  return typeof width[screen] === "string";
};

const generateGridTemplateColumns = (
  widths: ColumnWidth[],
  screen: ScreenSize
) => {
  return widths
    .map((width) => getColumnWidthByScreen(width, screen))
    .filter((value) => value !== "")
    .join(" ");
};

const generateCSS = (
  className: string,
  widths: ColumnWidth[],
  screen: ScreenSize
) => {
  const styleValue = generateGridTemplateColumns(widths, screen);
  const rule = `.${className}{grid-template-columns:${styleValue};}`;
  if (screen === "default") {
    return rule;
  } else {
    const size = breakpoints[screen];
    return `@media (min-width: ${size}) { ${rule} }`;
  }
};

const getColumnWidthByScreen = (target: ColumnWidth, screen: ScreenSize) => {
  if (typeof target === "string") {
    return target;
  }
  const index = screenSizes.indexOf(screen);
  if (index < 0) {
    throw new Error(`unexpected screen size "${screen}"`);
  }

  for (let i = index; i >= 0; i--) {
    const s = screenSizes[i];
    const value = target[s];
    if (typeof value === "string") {
      return value;
    }
  }

  return "";
};
