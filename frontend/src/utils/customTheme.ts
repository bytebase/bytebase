import { useLocalStorage } from "@vueuse/core";
import { useI18n } from "vue-i18n";

export const applyCustomTheme = (theme: string) => {
  const rootElement = document.documentElement;
  if (theme === "lixiang") {
    // TODO: update the color variables.
    rootElement.style.setProperty("--color-accent", "#52ab7c");
  }
};

export const getCustomProjectTitle = () => {
  const { t } = useI18n();
  const theme = useLocalStorage<string>("bb.custom-theme", "");
  if (theme.value === "lixiang") {
    return t("common.tenant");
  }
  return t("common.project");
};
