<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <template v-if="isDemo">
      <BannerDemo />
    </template>
    <template v-else-if="isReadonly">
      <div
        class="h-12 w-full text-lg font-medium bg-yellow-500 text-white flex justify-center items-center"
      >
        The server is in read-only mode. You can Execute SQL queries us the
        "SELECT" statement but not modify the database.
      </div>
    </template>
    <nav class="bg-white border-b border-block-border">
      <div class="max-w-full mx-auto px-4">
        <EditorHeader />
      </div>
    </nav>
    <!-- Suspense is experimental, be aware of the potential change -->
    <Suspense>
      <template #default>
        <ProvideSqlEditorContext>
          <router-view />
        </ProvideSqlEditorContext>
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
import { useStore } from "vuex";
import { computed } from "vue";

import ProvideSqlEditorContext from "@/components/ProvideSqlEditorContext.vue";
import EditorHeader from "@/views/SqlEditor/EditorHeader.vue";
import BannerDemo from "@/views/BannerDemo.vue";
import { ServerInfo } from "../types";

const store = useStore();

const ping = () => {
  store.dispatch("actuator/fetchInfo").then((info: ServerInfo) => {
    store.dispatch("notification/pushNotification", {
      module: "bytebase",
      style: "SUCCESS",
      title: info,
    });
  });
};

const isDemo = computed(() => store.getters["actuator/isDemo"]());

const isReadonly = computed(() => store.getters["actuator/isReadonly"]());
</script>
