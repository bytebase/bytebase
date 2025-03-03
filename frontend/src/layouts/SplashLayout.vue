<template>
  <div class="min-h-screen overflow-hidden flex">
    <div
      class="relative mx-auto flex-1 flex flex-col justify-center py-12 pb-24 px-4 sm:px-6 lg:flex-none lg:px-20 lg:w-1/2 xl:px-24"
    >
      <ProvideSetupContext v-if="route?.toString().startsWith('setup')">
        <router-view />
      </ProvideSetupContext>
      <router-view v-else />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import ProvideSetupContext from "@/views/Setup/ProvideSetupContext.vue";

const router = useRouter();

const route = computed(() => {
  return router.currentRoute.value.name;
});
</script>
