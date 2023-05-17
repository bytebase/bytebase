<template>
  <slot />
</template>

<script lang="ts">
import {
  useEnvironmentV1Store,
  useMemberStore,
  useRoleStore,
  usePrincipalStore,
  useUIStateStore,
  useLegacyProjectStore,
  useDebugStore,
  useUserStore,
} from "@/store";
import { defineComponent } from "vue";
import { DEFAULT_PROJECT_ID } from "../types";
import { useSettingV1Store } from "@/store/modules/v1/setting";

export default defineComponent({
  name: "ProvideDashboardContext",
  async setup() {
    await Promise.all([
      useSettingV1Store().fetchSettingList(),
      // Fetch so MemberSelect can have the data.
      useMemberStore().fetchMemberList(),
      useRoleStore().fetchRoleList(),
      useUserStore().fetchUserList(),
      // Though fetchMemberList also return the principal info, it's possible that a principal is no longer a member.
      // since all record types have creator, updater which are associated with principal, so we need to fetch
      // the principal list as well.
      // We also need this to render the proper inbox and activity entry.
      usePrincipalStore().fetchPrincipalList(),
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
