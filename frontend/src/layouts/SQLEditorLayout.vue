<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <BannersWrapper />
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
import { useLocalStorage } from "@vueuse/core";
import { onMounted } from "vue";
import { useRoute } from "vue-router";
import BannersWrapper from "@/components/BannersWrapper.vue";
import ProvideSQLEditorContext from "@/components/ProvideSQLEditorContext.vue";
import {
  pushNotification,
  useActuatorV1Store,
  useSQLEditorStore,
} from "@/store";
import { SQLEditorMode } from "@/types";

const actuatorStore = useActuatorV1Store();
const sqlEditorStore = useSQLEditorStore();
const route = useRoute();

onMounted(() => {
  let mode = route.query.mode as SQLEditorMode;
  const storage = useLocalStorage<SQLEditorMode>(
    "bb.sql-editor.mode",
    "BUNDLED"
  );
  if (mode != "BUNDLED" && mode != "STANDALONE") {
    mode = storage.value;
  }

  storage.value = mode;
  sqlEditorStore.setSQLEditorState({
    mode,
  });
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
