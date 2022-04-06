<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <template v-if="isDemo">
      <BannerDemo />
    </template>
    <template v-if="showDebugBanner">
      <BannerDebug />
    </template>
    <template v-else-if="isNearTrialExpireTime">
      <BannerTrial />
    </template>
    <template v-else-if="isReadonly">
      <div
        class="h-12 w-full text-lg font-medium bg-yellow-500 text-white flex justify-center items-center"
      >
        Server is in readonly mode. You can still view the console, but any
        change attempt will fail.
      </div>
    </template>
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
import BannerTrial from "../views/BannerTrial.vue";
import BannerDebug from "../views/BannerDebug.vue";
import { ServerInfo } from "../types";
import { isDBAOrOwner } from "../utils";
import { computed, defineComponent } from "vue";
import { useActuatorStore, useDebugStore, useNotificationStore } from "@/store";
import { storeToRefs } from "pinia";

export default defineComponent({
  name: "DashboardLayout",
  components: {
    ProvideDashboardContext,
    DashboardHeader,
    BannerDemo,
    BannerTrial,
    BannerDebug,
  },
  setup() {
    const store = useStore();
    const notificationStore = useNotificationStore();
    const actuatorStore = useActuatorStore();
    const debugStore = useDebugStore();

    const ping = () => {
      actuatorStore.fetchInfo().then((info: ServerInfo) => {
        notificationStore.pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: info,
        });
      });
    };

    const { isDemo, isReadonly } = storeToRefs(actuatorStore);
    const isNearTrialExpireTime = computed(() =>
      store.getters["subscription/isNearTrialExpireTime"]()
    );

    const { isDebug } = storeToRefs(debugStore);

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    // For now, debug mode is a global setting and will affect all users.
    // So we only allow DBA and Owner to toggle it and thus show a banner
    // reminding them to turn off
    const showDebugBanner = computed(() => {
      return isDebug.value && isDBAOrOwner(currentUser.value.role);
    });

    return {
      ping,
      isDemo,
      isReadonly,
      isNearTrialExpireTime,
      showDebugBanner,
    };
  },
});
</script>
