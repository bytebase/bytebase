<template>
  <slot />
</template>

<script lang="ts">
import { defineComponent } from "vue";
import {
  useUIStateStore,
  usePolicyV1Store,
  useDatabaseV1Store,
  useDBGroupStore,
  prepareBasicStores,
} from "@/store";

export default defineComponent({
  name: "ProvideDashboardContext",
  async setup() {
    await prepareBasicStores();
    await Promise.all([
      useDatabaseV1Store().fetchDatabaseList({
        parent: "instances/-",
      }),
      useDBGroupStore().fetchAllDatabaseGroupList(),
      usePolicyV1Store().getOrFetchPolicyByName("policies/WORKSPACE_IAM"),
      useUIStateStore().restoreState(),
    ]);
  },
});
</script>
