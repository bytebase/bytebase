<template>
  <div
    class="flex justify-center items-center bg-gray-100 rounded-3xl"
    :class="customBrandingLogo ? 'p-2' : ''"
  >
    <img
      v-if="customBrandingLogo"
      class="h-6 mr-4 ml-1 bg-no-repeat bg-contain bg-center"
      src="@/assets/logo-full.svg"
    />
    <slot />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useSettingStore } from "@/store";

const settingStore = useSettingStore();

const customBrandingLogo = computed((): string | undefined => {
  const brandingLogoSetting = settingStore.getSettingByName("bb.branding.logo");
  return brandingLogoSetting?.value;
});
</script>
