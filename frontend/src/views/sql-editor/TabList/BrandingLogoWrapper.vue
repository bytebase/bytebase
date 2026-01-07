<template>
  <div
    class="flex justify-center items-center bg-gray-100 rounded-3xl"
    :class="customBrandingLogo ? 'md:px-1 md:py-0.5' : ''"
  >
    <img
      v-if="customBrandingLogo"
      class="hidden md:block h-6 mr-2 ml-1 bg-no-repeat bg-contain bg-center"
      :src="customBrandingLogo"
    />
    <slot />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useActuatorV1Store } from "@/store";

const customBrandingLogo = computed((): string | undefined => {
  return useActuatorV1Store().brandingLogo;
});
</script>
