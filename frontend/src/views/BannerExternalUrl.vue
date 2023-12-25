<template>
  <div v-if="show" class="bg-accent">
    <div class="mx-auto py-3 px-3">
      <div class="flex items-center justify-between flex-wrap">
        <div class="w-0 flex-1 flex items-center">
          <p class="ml-3 font-medium text-white truncate">
            <span>
              {{ $t("banner.external-url") }}
            </span>
          </p>
        </div>
        <div
          class="order-3 mt-2 flex-shrink-0 w-full sm:order-2 sm:mt-0 sm:w-auto"
        >
          <router-link
            to="/setting/general"
            class="flex items-center justify-center pl-4 pr-2 py-2 border border-transparent rounded-md shadow-sm text-base font-medium text-accent bg-white hover:bg-indigo-50"
          >
            {{ $t("banner.configure-now") }}
            <heroicons-outline:wrench-screwdriver class="ml-1 w-5 h-5" />
          </router-link>
        </div>
        <div class="order-2 flex-shrink-0 sm:order-3 sm:ml-3 -mr-1">
          <NButton @click.prevent="show = false">
            <span class="sr-only">{{ $t("common.dismiss") }}</span>
            <!-- Heroicon name: outline/x -->
            <heroicons-outline:x class="h-6 w-6 text-white" />
          </NButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref } from "vue";

export default defineComponent({
  name: "BannerExternalUrl",
  setup() {
    const show = ref(true);

    return { show };
  },
});
</script>
