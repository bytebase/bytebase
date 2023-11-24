<template>
  <slot />
</template>

<script lang="ts">
import { defineComponent } from "vue";
import {
  useUIStateStore,
  useDatabaseV1Store,
  useDBGroupStore,
  initCommonModelStores,
} from "@/store";

export default defineComponent({
  name: "ProvideDashboardContext",
  async setup() {
    await initCommonModelStores();
    await Promise.all([
      useDatabaseV1Store().fetchDatabaseList({
        parent: "instances/-",
      }),
      useDBGroupStore().fetchAllDatabaseGroupList(),
      useUIStateStore().restoreState(),
    ]);
  },
});
</script>
