import { useLocalStorage } from "@vueuse/core";
import { useI18n } from "vue-i18n";

export const applyCustomTheme = (theme: string) => {
  const rootElement = document.documentElement;
  if (theme === "lixiang") {
    rootElement.style.setProperty("--color-accent", "#00665f");
    rootElement.style.setProperty("--color-accent-disabled", "#ecf3f3");
    rootElement.style.setProperty("--color-accent-hover", "#00554f");
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
