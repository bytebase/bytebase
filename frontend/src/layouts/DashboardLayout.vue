<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <BannerDemo v-if="isDemo" />
    <nav class="bg-white border-b border-block-border">
      <div class="max-w-full mx-auto px-4">
        <DashboardHeader />
      </div>
    </nav>
    <!-- Suspense is experimental, be aware of the potential change -->
    <Suspense>
      <template #default>
        <ProvideDashboardContext>
          <router-view name="body" />
        </ProvideDashboardContext>
      </template>
      <template #fallback>
        <div class="flex flex-row justify-between p-4 space-x-2">
          <span class="items-center flex">Loading...</span>
          <button
            class="items-center flex justify-center btn-normal"
            @click.prevent="ping"
          >
            Ping
          </button>
        </div>
      </template>
    </Suspense>
  </div>
</template>

<script lang="ts">
import { useStore } from "vuex";
import ProvideDashboardContext from "../components/ProvideDashboardContext.vue";
import DashboardHeader from "../views/DashboardHeader.vue";
import BannerDemo from "../views/BannerDemo.vue";
import { ServerInfo } from "../types";
import { computed } from "@vue/runtime-core";

export default {
  name: "DashboardLayout",
  components: { ProvideDashboardContext, DashboardHeader, BannerDemo },
  setup(props, ctx) {
    const store = useStore();

    const ping = () => {
      store.dispatch("actuator/info").then((info: ServerInfo) => {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "SUCCESS",
          title: info,
        });
      });
    };

    const isDemo = computed(() => store.getters["actuator/isDemo"]());

    return {
      ping,
      isDemo,
    };
  },
};
</script>
