<template>
  <slot />
</template>

<script lang="ts">
import { useMemberStore, usePrincipalStore, useUIStateStore } from "@/store";
import { defineComponent } from "vue";
import { useStore } from "vuex";
import { DEFAULT_PROJECT_ID } from "../types";

export default defineComponent({
  name: "ProvideDashboardContext",
  async setup() {
    const store = useStore();

    await Promise.all([
      store.dispatch("setting/fetchSetting"),
      // Fetch so MemberSelect can have the data.
      useMemberStore().fetchMemberList(),
      // Though fetchMemberList also return the principal info, it's possible that a principal is no longer a member.
      // since all record types have creator, updater which are associated with principal, so we need to fetch
      // the principal list as well.
      // We also need this to render the proper inbox and activity entry.
      usePrincipalStore().fetchPrincipalList(),
      store.dispatch("environment/fetchEnvironmentList"),
      // The default project hosts databases not explicitly assigned to other users project.
      store.dispatch("project/fetchProjectById", DEFAULT_PROJECT_ID),
      useUIStateStore().restoreState(),
    ]);
  },
});
</script>
