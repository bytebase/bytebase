<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <template v-if="isDemo">
      <BannerDemo />
    </template>
    <template v-else-if="isExpired || isTrialing">
      <BannerSubscription />
    </template>

    <nav class="bg-white border-b border-block-border">
      <div class="max-w-full mx-auto px-4">
        <EditorHeader />
      </div>
    </nav>

    <!-- Suspense is experimental, be aware of the potential change -->
    <Suspense>
      <template #default>
        <router-view />
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

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import {
  pushNotification,
  useActuatorStore,
  useSubscriptionStore,
} from "@/store";
import { ServerInfo } from "@/types";
import BannerDemo from "@/views/BannerDemo.vue";
import BannerSubscription from "@/views/BannerSubscription.vue";
import EditorHeader from "@/views/sql-editor/EditorHeader.vue";

const actuatorStore = useActuatorStore();
const subscriptionStore = useSubscriptionStore();

const ping = () => {
  actuatorStore.fetchInfo().then((info: ServerInfo) => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: JSON.stringify(info, null, 4),
    });
  });
};

const { isDemo } = storeToRefs(actuatorStore);
const { isExpired, isTrialing } = storeToRefs(subscriptionStore);
</script>
