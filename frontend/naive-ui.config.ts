import { GlobalThemeOverrides } from "naive-ui";
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
