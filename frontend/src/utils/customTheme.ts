import { watch, type Ref } from "vue";
import { hexToRgb } from "./css";

export const useCustomTheme = (
  theme: Ref<Record<string, string> | undefined>
) => {
  watch(
    theme,
    (to, from) => {
      const rootElement = document.documentElement;
      for (const cssVar in from) {
        rootElement.style.removeProperty(cssVar);
      }
      for (const cssVar in to) {
        rootElement.style.setProperty(cssVar, hexToRgb(to[cssVar]).join(" "));
      }
    },
    {
      immediate: true,
    }
  );
};
