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
