import i18n from "@/react/i18n";
import { STORAGE_KEY_LANGUAGE } from "@/utils/storage-keys";
import { emitStorageChangedEvent } from "@/utils/util";

export type HeaderLocaleOption = {
  label: string;
  value: string;
};

export const HEADER_LANGUAGE_OPTIONS: HeaderLocaleOption[] = [
  { label: "English", value: "en-US" },
  { label: "简体中文", value: "zh-CN" },
  { label: "Español", value: "es-ES" },
  { label: "日本語", value: "ja-JP" },
  { label: "Tiếng việt", value: "vi-VN" },
];

export function setAppLocale(lang: string) {
  void i18n.changeLanguage(lang);
  localStorage.setItem(STORAGE_KEY_LANGUAGE, lang);
  emitStorageChangedEvent();
  window.dispatchEvent(
    new CustomEvent("bb.react-locale-change", { detail: lang })
  );
}
