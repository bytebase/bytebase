<template>
  <div
    class="flex-shrink-0 w-44 min-h-[4rem] flex items-center overflow-y-hidden"
  >
    <router-link
      to="/"
      class="select-none"
      active-class=""
      exact-active-class=""
    >
      <img
        v-if="customBrandingLogo"
        :src="customBrandingLogo"
        class="w-full max-h-full"
      />

      <img
        v-if="!customBrandingLogo"
        class="h-12 w-auto"
        src="../assets/logo-full.svg"
        alt="Bytebase"
      />
    </router-link>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useSettingV1Store } from "@/store/modules/v1/setting";

const customBrandingLogo = computed((): string | undefined => {
  return useSettingV1Store().brandingLogo;
});
</script>
