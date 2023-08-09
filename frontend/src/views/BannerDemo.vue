<template>
  <div v-if="show" class="bg-accent">
    <div class="mx-auto py-3 px-3">
      <div class="flex items-center justify-between flex-wrap">
        <div class="w-0 flex-1 flex items-center">
          <p class="ml-3 font-medium text-white truncate">
            <a
              href="https://cal.com/adela-bytebase/30min"
              target="_blank"
              class="flex underline"
            >
              <heroicons-outline:calendar class="mr-1 w-6 h-6" />
              {{ $t("banner.request-demo") }}
            </a>
          </p>
        </div>
        <div
          class="flex flex-row space-x-4 order-3 mt-2 flex-shrink-0 w-full sm:order-2 sm:mt-0 sm:w-auto"
        >
          <a
            href="https://bytebase.com/docs/get-started/install/deploy-with-docker?source=demo"
            target="_blank"
            class="flex items-center justify-center pl-4 pr-2 py-2 border border-transparent rounded-md shadow-sm text-base font-medium text-accent bg-white hover:bg-indigo-50"
          >
            {{ $t("banner.deploy") }}
            <heroicons-outline:arrow-down-tray class="ml-1 w-6 h-6" />
          </a>
          <a
            href="https://hub.bytebase.com/workspace?source=demo"
            target="_blank"
            class="flex items-center justify-center pl-4 pr-2 py-2 border border-transparent rounded-md shadow-sm text-base font-medium text-accent bg-white hover:bg-indigo-50"
          >
            {{ $t("banner.cloud") }}
            <heroicons-outline:cloud class="ml-1 w-6 h-6" />
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
import { storeToRefs } from "pinia";
import { defineComponent, ref } from "vue";
import { useActuatorV1Store } from "@/store";

export default defineComponent({
  name: "BannerDemo",
  setup() {
    const actuatorStore = useActuatorV1Store();
    const show = ref(true);

    const { isReadonly } = storeToRefs(actuatorStore);

    return { show, isReadonly };
  },
});
</script>
