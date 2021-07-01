<template>
  <slot />
</template>

<script lang="ts">
import { ComputedRef } from "@vue/reactivity";
import { useStore } from "vuex";
import { Principal, DEFAULT_PROJECT_ID } from "../types";
import { computed } from "@vue/runtime-core";

export default {
  name: "ProvideDashboardContext",
  async setup() {
    const store = useStore();

    const currentUser: ComputedRef<Principal> = computed(() =>
      store.getters["auth/currentUser"]()
    );

    await Promise.all([
      // Fetch so MemberSelect can have the data.
      store.dispatch("member/fetchMemberList"),
      // Though fetchMemberList also return the principal info, it's possible that a principal is no longer a member.
      // since all record types have creator, updater which are associated with principal, so we need to fetch
      // the principal list as well.
      store.dispatch("principal/fetchPrincipalList"),
      store.dispatch("environment/fetchEnvironmentList"),
      // The default project hosts databases not explicitly assigned to other users project.
      store.dispatch("project/fetchProjectById", DEFAULT_PROJECT_ID),
      store.dispatch("message/fetchMessageListByUser", currentUser.value.id),
      store.dispatch("uistate/restoreState"),
    ]);
  },
};
</script>
