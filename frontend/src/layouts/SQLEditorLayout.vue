<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <BannersWrapper v-if="showBanners" />
    <!-- Suspense is experimental, be aware of the potential change -->
    <Suspense>
      <template #default>
        <ProvideSQLEditorContext>
          <router-view />
        </ProvideSQLEditorContext>
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
import { computed } from "vue";
import BannersWrapper from "@/components/BannersWrapper.vue";
import ProvideSQLEditorContext from "@/components/ProvideSQLEditorContext.vue";
import { pushNotification, useActuatorV1Store } from "@/store";

const actuatorStore = useActuatorV1Store();
const { pageMode } = storeToRefs(actuatorStore);

const showBanners = computed(() => {
  return pageMode.value === "BUNDLED";
});

const ping = () => {
  actuatorStore.fetchServerInfo().then((info) => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: JSON.stringify(info),
    });
  });
};
</script>
