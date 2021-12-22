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

      successColor: callVar("--color-success"),
      successColorHover: callVar("--color-success-hover"),

      warningColor: callVar("--color-warning"),
      warningColorHover: callVar("--color-warning-hover"),

      errorColor: callVar("--color-error"),
      errorColorHover: callVar("--color-error-hover"),
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
