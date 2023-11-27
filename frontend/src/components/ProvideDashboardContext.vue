<template>
  <slot />
</template>

<script lang="ts">
import { defineComponent } from "vue";
import {
  useEnvironmentV1Store,
  useInstanceV1Store,
  usePolicyV1Store,
  useProjectV1Store,
  useRoleStore,
  useSettingV1Store,
  useUserStore,
  useUIStateStore,
  useDatabaseV1Store,
  useDBGroupStore,
} from "@/store";

export default defineComponent({
  name: "ProvideDashboardContext",
  async setup() {
    await Promise.all([
      useUserStore().fetchUserList(),
      useSettingV1Store().fetchSettingList(),
      useRoleStore().fetchRoleList(),
      useEnvironmentV1Store().fetchEnvironments(),
      useInstanceV1Store().fetchInstanceList(),
      useProjectV1Store().fetchProjectList(true),
      usePolicyV1Store().getOrFetchPolicyByName("policies/WORKSPACE_IAM"),
    ]);
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
