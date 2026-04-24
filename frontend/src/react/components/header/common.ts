import i18n from "@/plugins/i18n";
import { router } from "@/router";
import type { useUIStateStore } from "@/store";
import { emitStorageChangedEvent, setDocumentTitle } from "@/utils";
import { STORAGE_KEY_LANGUAGE } from "@/utils/storage-keys";

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
  i18n.global.locale.value = lang;
  localStorage.setItem(STORAGE_KEY_LANGUAGE, JSON.stringify(lang));
  emitStorageChangedEvent();

  const route = router.currentRoute.value;
  if (route.meta.title) {
    setDocumentTitle(route.meta.title(route));
  }
}

export function resetQuickstartProgress(
  uiStateStore: ReturnType<typeof useUIStateStore>
) {
  const keys = [
    "hidden",
    "issue.visit",
    "project.visit",
    "environment.visit",
    "instance.visit",
    "database.visit",
    "member.visit",
    "data.query",
  ];

  for (const key of keys) {
    void uiStateStore.saveIntroStateByKey({
      key,
      newState: false,
    });
  }
}
