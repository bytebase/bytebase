<template>
  <slot />
</template>

<script lang="ts">
import { defineComponent } from "vue";
import {
  useEnvironmentV1Store,
  useRoleStore,
  useUIStateStore,
  useLegacyProjectStore,
  useDebugStore,
  useUserStore,
} from "@/store";
import { DEFAULT_PROJECT_ID } from "../types";
import { useSettingV1Store } from "@/store/modules/v1/setting";

export default defineComponent({
  name: "ProvideDashboardContext",
  async setup() {
    await Promise.all([
      useSettingV1Store().fetchSettingList(),
      useRoleStore().fetchRoleList(),
      useUserStore().fetchUserList(),
      useEnvironmentV1Store().fetchEnvironments(),
      // The default project hosts databases not explicitly assigned to other users project.
      useLegacyProjectStore().fetchProjectById(DEFAULT_PROJECT_ID),
      useLegacyProjectStore().fetchAllProjectList(), // TODO(Jim): For legacy API support only. Remove this after refactored
      useUIStateStore().restoreState(),
      useDebugStore().fetchDebug(),
    ]);
  },
});
</script>
