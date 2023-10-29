import { useLocalStorage } from "@vueuse/core";
import { useI18n } from "vue-i18n";
import { hexToRgb } from "./css";

export const applyCustomTheme = (theme: string) => {
  const rootElement = document.documentElement;
  if (theme === "lixiang") {
    rootElement.style.setProperty(
      "--color-accent",
      hexToRgb("#00665f").join(" ")
    );
    rootElement.style.setProperty(
      "--color-accent-disabled",
      hexToRgb("#b8c3c3").join(" ")
    );
    rootElement.style.setProperty(
      "--color-accent-hover",
      hexToRgb("#00554f").join(" ")
    );
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
