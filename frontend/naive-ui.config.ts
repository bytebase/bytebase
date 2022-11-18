import {
  GlobalThemeOverrides,
  zhCN,
  dateZhCN,
  NLocale,
  NDateLocale,
} from "naive-ui";

import { curLocale } from "./src/plugins/i18n";

import { computed } from "vue";

const callVar = (css: string) => {
  return getComputedStyle(document.documentElement).getPropertyValue(css);
};

export const themeOverrides = computed((): GlobalThemeOverrides => {
  return {
    common: {
      primaryColor: callVar("--color-accent"),
      primaryColorHover: callVar("--color-accent-hover"),
      primaryColorPressed: callVar("--color-accent"),

      successColor: callVar("--color-success"),
      successColorHover: callVar("--color-success-hover"),
      successColorPressed: callVar("--color-success"),

      warningColor: callVar("--color-warning"),
      warningColorHover: callVar("--color-warning-hover"),
      warningColorPressed: callVar("--color-warning"),

      errorColor: callVar("--color-error"),
      errorColorHover: callVar("--color-error-hover"),
      errorColorPressed: callVar("--color-error"),
    },
    Button: {
      colorInfo: callVar("--color-accent"),
      colorHoverInfo: callVar("--color-accent-hover"),
      colorPressedInfo: callVar("--color-accent-disabled"),
      colorFocusInfo: callVar("--color-accent"),
      colorDisabledInfo: callVar("--color-accent-disabled"),
      borderInfo: callVar("--color-accent"),
      borderHoverInfo: callVar("--color-accent-hover"),
      borderFocusInfo: callVar("--color-accent"),
    },
    Dialog: {
      iconColorInfo: callVar("--color-accent"),
    },
  };
});

export const darkThemeOverrides = computed((): GlobalThemeOverrides => {
  return {
    common: {
      primaryColor: callVar("--color-dark-green"),
      primaryColorHover: callVar("--color-dark-green-hover"),
      primaryColorPressed: callVar("--color-dark-green"),
    },
    Button: {
      colorInfo: callVar("--color-dark-green"),
      colorHoverInfo: callVar("--color-dark-green-hover"),
      colorFocusInfo: callVar("--color-dark-green"),
      borderInfo: callVar("--color-dark-green"),
      borderHoverInfo: callVar("--color-dark-green-hover"),
      borderFocusInfo: callVar("--color-dark-green"),
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
