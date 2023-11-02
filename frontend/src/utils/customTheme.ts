import { useLocalStorage } from "@vueuse/core";
import { useI18n } from "vue-i18n";
import { hexToRgb } from "./css";

export const customTheme = useLocalStorage<string>("bb.custom-theme", "");

export const applyCustomTheme = () => {
  const rootElement = document.documentElement;
  if (customTheme.value === "lixiang") {
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
  } else {
    rootElement.style.removeProperty("--color-accent");
    rootElement.style.removeProperty("--color-accent-disabled");
    rootElement.style.removeProperty("--color-accent-hover");
  }
};

export const getCustomProjectTitle = () => {
  const { t } = useI18n();
  if (customTheme.value === "lixiang") {
    return t("common.tenant");
  }
  return t("common.project");
};
