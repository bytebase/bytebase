<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <BannersWrapper />
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
import { defineComponent } from "vue";
import { ServerInfo } from "@/types";
import { pushNotification, useActuatorStore } from "@/store";
import ProvideDashboardContext from "@/components/ProvideDashboardContext.vue";
import BannersWrapper from "@/components/BannersWrapper.vue";

export default defineComponent({
  name: "DashboardLayout",
  components: {
    ProvideDashboardContext,
    BannersWrapper,
  },
  setup() {
    const actuatorStore = useActuatorStore();

    const ping = () => {
      actuatorStore.fetchServerInfo().then((info: ServerInfo) => {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: JSON.stringify(info),
        });
      });
    };

    return {
      ping,
    };
  },
});
</script>
