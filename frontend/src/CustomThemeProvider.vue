<template>
  <NConfigProvider
    v-if="initialized"
    :key="key"
    :locale="generalLang"
    :date-locale="dateLang"
    :theme-overrides="themeOverrides"
  >
    <slot></slot>
  </NConfigProvider>
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import { NConfigProvider } from "naive-ui";
import { onMounted, ref } from "vue";
import { themeOverrides, dateLang, generalLang } from "../naive-ui.config";
import { provideAppRootContext } from "./AppRootContext";
import { useLanguage } from "./composables/useLanguage";
import { applyCustomTheme } from "./utils/customTheme";

const { key } = provideAppRootContext();
const { setLocale } = useLanguage();
const initialized = ref(false);

onMounted(() => {
  const searchParams = new URLSearchParams(window.location.search);
  // Initial custom theme.
  let customTheme = searchParams.get("customTheme") || "";
  const cachedCustomTheme = useLocalStorage<string>("bb.custom-theme", "");
  if (!customTheme && cachedCustomTheme.value) {
    customTheme = cachedCustomTheme.value;
  }
  if (customTheme) {
    cachedCustomTheme.value = customTheme;
    applyCustomTheme(customTheme);
  }
  // Initial custom language.
  const lang = searchParams.get("lang") || "";
  if (lang) {
    setLocale(lang);
  }
  initialized.value = true;
});
</script>
