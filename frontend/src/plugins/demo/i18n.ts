import { head } from "lodash-es";
import * as storage from "./storage";
import { I18NText } from "./types";

export const nextI18NText = {
  "en-US": "Next",
  "zh-CN": "下一步",
  "es-ES": "Siguiente",
};

export const getStringFromI18NText = (text: string | I18NText) => {
  if (typeof text === "string") {
    return text;
  }

  const values = Object.values(text);
  const firstValue = head(values) || "";
  const { bytebase_options: BBOptions } = storage.get(["bytebase_options"]);

  if (BBOptions && BBOptions.appearance) {
    const language = BBOptions.appearance.language;
    return text[language] || firstValue;
  }

  return firstValue;
};
