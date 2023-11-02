import {
  GlobalThemeOverrides,
  zhCN,
  dateZhCN,
  NLocale,
  NDateLocale,
} from "naive-ui";
import { computed } from "vue";
import { callCssVariable } from "@/utils";
import { customTheme } from "@/utils/customTheme";
import { curLocale } from "./src/plugins/i18n";

export const themeOverrides = computed((): GlobalThemeOverrides => {
  // Touch customTheme.value here, so when customTheme changes, this computed
  // value will be re-evaluated reactively
  console.debug(customTheme.value);
  return {
    common: {
      primaryColor: callCssVariable("--color-accent"),
      primaryColorHover: callCssVariable("--color-accent-hover"),
      primaryColorPressed: callCssVariable("--color-accent"),

      successColor: callCssVariable("--color-success"),
      successColorHover: callCssVariable("--color-success-hover"),
      successColorPressed: callCssVariable("--color-success"),

      warningColor: callCssVariable("--color-warning"),
      warningColorHover: callCssVariable("--color-warning-hover"),
      warningColorPressed: callCssVariable("--color-warning"),

      infoColor: callCssVariable("--color-info"),
      infoColorHover: callCssVariable("--color-info-hover"),
      infoColorPressed: callCssVariable("--color-info"),

      errorColor: callCssVariable("--color-error"),
      errorColorHover: callCssVariable("--color-error-hover"),
      errorColorPressed: callCssVariable("--color-error"),
    },
    Button: {
      color: "white",
      colorHover: "white",
      colorFocus: "white",
      colorPressed: "white",
    },
  };
});

export const darkThemeOverrides = computed((): GlobalThemeOverrides => {
  // Touch customTheme.value here, so when customTheme changes, this computed
  // value will be re-evaluated reactively
  console.debug(customTheme.value);
  return {
    common: {
      primaryColor: callCssVariable("--color-matrix-green"),
      primaryColorHover: callCssVariable("--color-matrix-green-hover"),
      primaryColorPressed: callCssVariable("--color-matrix-green"),
    },
    Button: {
      color: "transparent",
      colorHover: "transparent",
      colorFocus: "transparent",
      colorPressed: "transparent",
      colorInfo: callCssVariable("--color-matrix-green"),
      colorHoverInfo: callCssVariable("--color-matrix-green-hover"),
      colorFocusInfo: callCssVariable("--color-matrix-green"),
      borderInfo: callCssVariable("--color-matrix-green"),
      borderHoverInfo: callCssVariable("--color-matrix-green-hover"),
      borderFocusInfo: callCssVariable("--color-matrix-green"),
    },
    Tabs: {
      tabTextColorCard: callCssVariable("--color-control-placeholder"),
    },
  };
});

const isZhCn = (): boolean => {
  return curLocale.value === "zh-CN";
};

export const dateLang = computed((): NDateLocale | null => {
  return isZhCn() ? dateZhCN : null;
});

export const generalLang = computed((): NLocale | null => {
  return isZhCn() ? zhCN : null;
});
