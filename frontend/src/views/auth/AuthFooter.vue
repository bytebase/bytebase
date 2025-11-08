<template>
  <div class="absolute left-0 bottom-0 mb-8 text-center w-full">
    <p class="block text-sm text-gray-400 flex justify-center gap-x-2">
      <a
        v-for="item in languageList"
        :key="item.label"
        href="#"
        class="hover:text-gray-600"
        :class="{ 'text-gray-800': item.label === selectedLanguage }"
        @click.prevent="changeLanguage(item)"
      >
        {{ item.label }}
      </a>
    </p>
    <p class="text-sm text-gray-400 mt-1">
      &copy; {{ year }} Bytebase. All rights reserved.
    </p>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { useLanguage } from "../../composables/useLanguage";

const { locale, setLocale } = useLanguage();

const languageList = [
  {
    label: "English",
    value: "en-US",
  },
  {
    label: "简体中文",
    value: "zh-CN",
  },
  {
    label: "Español",
    value: "es-ES",
  },
  {
    label: "日本語",
    value: "ja-JP",
  },
  {
    label: "Tiếng việt",
    value: "vi-VN",
  },
];
const localeLabel =
  locale.value === "zh-CN" || locale.value === "zh"
    ? "简体中文"
    : locale.value === "es-ES" || locale.value === "es"
      ? "Español"
      : locale.value === "ja-JP" || locale.value === "ja"
        ? "日本語"
        : locale.value === "vi-VN" || locale.value === "vi"
          ? "Tiếng việt"
          : "English";
const selectedLanguage = ref(localeLabel);
const year = new Date().getFullYear();

const changeLanguage = (item: { label: string; value: string }) => {
  setLocale(item.value);
  selectedLanguage.value = item.label;
};
</script>
