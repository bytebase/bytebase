<template>
  <NConfigProvider
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
import { watch } from "vue";
import { nextTick } from "vue";
import { useRoute } from "vue-router";
import { themeOverrides, dateLang, generalLang } from "../naive-ui.config";
import { provideAppRootContext, restartAppRoot } from "./AppRootContext";
import { useLanguage } from "./composables/useLanguage";
import { applyCustomTheme } from "./utils/customTheme";

const route = useRoute();
const { key } = provideAppRootContext();
const { setLocale } = useLanguage();
const cachedCustomTheme = useLocalStorage<string>("bb.custom-theme", "");

watch(
  () => route.fullPath,
  () => {
    const searchParams = new URLSearchParams(window.location.search);
    // Initial custom theme.
    let customTheme = searchParams.get("customTheme") || "";
    if (!customTheme && customTheme !== cachedCustomTheme.value) {
      customTheme = cachedCustomTheme.value;
    }
    if (customTheme) {
      cachedCustomTheme.value = customTheme;
      applyCustomTheme(customTheme);
      // If custom theme is applied, we need to restart the app root to make sure
      // the theme is applied to root variables.
      nextTick(() => {
        restartAppRoot();
      });
    }
    // Initial custom language.
    const lang = searchParams.get("lang") || "";
    if (lang) {
      setLocale(lang);
    }
  },
  {
    immediate: true,
  }
);
</script>
