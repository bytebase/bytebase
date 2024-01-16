<template>
  <div class="absolute left-0 bottom-0 mb-8 text-center w-full">
    <p class="block text-sm text-gray-400 space-x-2">
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
];
const localeLabel =
  locale.value === "en-US" || locale.value === "en"
    ? "English"
    : locale.value === "es-ES" || locale.value === "es"
    ? "Español"
    : locale.value === "ja-JP" || locale.value === "ja"
    ? "日本語"
    : "简体中文";
const selectedLanguage = ref(localeLabel);
const year = new Date().getFullYear();

const changeLanguage = (item: any) => {
  setLocale(item.value);
  selectedLanguage.value = item.label;
};
</script>
