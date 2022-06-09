<template>
  <div v-if="show" class="bg-accent">
    <div class="mx-auto py-3 px-3">
      <div class="flex items-center justify-between flex-wrap">
        <div class="w-0 flex-1 flex items-center">
          <p class="ml-3 font-medium text-white truncate">
            <span>
              {{ $t(`banner.demo-intro${isReadonly ? "-readonly" : ""}`) }}
            </span>
          </p>
        </div>
        <div
          class="order-3 mt-2 flex-shrink-0 w-full sm:order-2 sm:mt-0 sm:w-auto"
        >
          <a
            href="https://www.bytebase.com/docs/install/install-with-docker?source=console"
            target="_blank"
            class="flex items-center justify-center pl-4 pr-2 py-2 border border-transparent rounded-md shadow-sm text-base font-medium text-accent bg-white hover:bg-indigo-50"
          >
            {{ $t("banner.action") }}
            <heroicons-outline:lightning-bolt class="ml-1 w-6 h-6" />
          </a>
        </div>
        <div class="order-2 flex-shrink-0 sm:order-3 sm:ml-3">
          <button
            type="button"
            class="-mr-1 flex p-2 rounded-md hover:bg-accent-hover focus:outline-none focus:ring-2 focus:ring-white sm:-mr-2"
            @click.prevent="show = false"
          >
            <span class="sr-only">{{ $t("common.dismiss") }}</span>
            <!-- Heroicon name: outline/x -->
            <heroicons-outline:x class="h-6 w-6 text-white" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { useActuatorStore } from "@/store";
import { storeToRefs } from "pinia";
import { defineComponent, ref } from "vue";

export default defineComponent({
  name: "BannerDemo",
  setup() {
    const actuatorStore = useActuatorStore();
    const show = ref(true);

    const { isReadonly } = storeToRefs(actuatorStore);

    return { show, isReadonly };
  },
});
</script>
