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
import { applyCustomTheme } from "./utils/customTheme";

const { key } = provideAppRootContext();
const initialized = ref(false);

onMounted(() => {
  const searchParams = new URLSearchParams(window.location.search);
  let customTheme = searchParams.get("customTheme") || "";
  const cachedCustomTheme = useLocalStorage<string>("bb.custom-theme", "");
  if (!customTheme && cachedCustomTheme.value) {
    customTheme = cachedCustomTheme.value;
  }
  if (customTheme) {
    cachedCustomTheme.value = customTheme;
    applyCustomTheme(customTheme);
  }
  initialized.value = true;
});
</script>
