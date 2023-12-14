<template>
  <div
    class="flex-shrink-0 w-44 min-h-[4rem] flex items-center overflow-y-hidden"
  >
    <router-link
      to="/"
      class="w-full select-none flex flex-row justify-center items-center"
      active-class=""
      exact-active-class=""
    >
      <img
        v-if="customBrandingLogo"
        :src="customBrandingLogo"
        class="w-auto max-w-full my-3"
      />

      <img
        v-else
        class="h-8 md:h-10 w-auto"
        src="@/assets/logo-full.svg"
        alt="Bytebase"
      />
    </router-link>
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
