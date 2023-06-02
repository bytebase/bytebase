<template>
  <slot />
</template>

<script lang="ts">
import { defineComponent } from "vue";
import {
  useEnvironmentV1Store,
  useRoleStore,
  useUIStateStore,
  useUserStore,
  useProjectV1Store,
} from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { useInstanceV1Store } from "@/store/modules/v1/instance";

export default defineComponent({
  name: "ProvideDashboardContext",
  async setup() {
    await Promise.all([
      useSettingV1Store().fetchSettingList(),
      useRoleStore().fetchRoleList(),
      useUserStore().fetchUserList(),
      useEnvironmentV1Store().fetchEnvironments(),
      useInstanceV1Store().fetchInstanceList(),
      useProjectV1Store().fetchProjectList(true),
      useUIStateStore().restoreState(),
    ]);
  },
});
</script>
