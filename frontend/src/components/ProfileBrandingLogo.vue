<template>
  <div
    class="flex justify-center items-center bg-gray-100 rounded-3xl"
    :class="customBrandingLogo ? 'md:px-2 md:py-1.5' : ''"
  >
    <img
      v-if="customBrandingLogo"
      class="hidden md:block h-6 mr-4 ml-1 bg-no-repeat bg-contain bg-center"
      src="@/assets/logo-full.svg"
    />
    <slot />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useSettingSWRStore } from "@/store/modules/v1/setting";

const brandingLogoSetting = useSettingSWRStore().useSettingByName(
  "bb.branding.logo",
  /* silent */ true
);

const customBrandingLogo = computed(() => {
  return brandingLogoSetting.data.value?.value?.stringValue;
});
</script>
