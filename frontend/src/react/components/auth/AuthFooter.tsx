import i18n from "@/plugins/i18n";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { emitStorageChangedEvent, setDocumentTitle } from "@/utils";
import { STORAGE_KEY_LANGUAGE } from "@/utils/storage-keys";

type LanguageOption = {
  readonly label: string;
  readonly value: string;
};

const LANGUAGE_LIST: readonly LanguageOption[] = [
  { label: "English", value: "en-US" },
  { label: "简体中文", value: "zh-CN" },
  { label: "Español", value: "es-ES" },
  { label: "日本語", value: "ja-JP" },
  { label: "Tiếng việt", value: "vi-VN" },
];

function resolveLabel(locale: string): string {
  if (locale === "zh-CN" || locale === "zh") return "简体中文";
  if (locale === "es-ES" || locale === "es") return "Español";
  if (locale === "ja-JP" || locale === "ja") return "日本語";
  if (locale === "vi-VN" || locale === "vi") return "Tiếng việt";
  return "English";
}

function setAppLocale(lang: string) {
  i18n.global.locale.value = lang;
  localStorage.setItem(STORAGE_KEY_LANGUAGE, JSON.stringify(lang));
  emitStorageChangedEvent();
  const route = router.currentRoute.value;
  if (route.meta.title) {
    setDocumentTitle(route.meta.title(route));
  }
}

export function AuthFooter() {
  const currentLocale = useVueState(() => i18n.global.locale.value as string);
  const activeLabel = resolveLabel(currentLocale);
  const year = new Date().getFullYear();

  return (
    <div className="absolute left-0 bottom-0 mb-8 text-center w-full">
      <p className="block text-sm text-control-light flex justify-center gap-x-2">
        {LANGUAGE_LIST.map((item) => (
          <a
            key={item.label}
            href="#"
            className={`hover:text-control ${
              item.label === activeLabel ? "text-main" : ""
            }`}
            onClick={(e) => {
              e.preventDefault();
              setAppLocale(item.value);
            }}
          >
            {item.label}
          </a>
        ))}
      </p>
      <p className="text-sm text-control-light mt-1">
        &copy; {year} Bytebase. All rights reserved.
      </p>
    </div>
  );
}
