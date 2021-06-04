<template>
  <slot />
</template>

<script lang="ts">
import { useStore } from "vuex";
import { Principal, DEFAULT_PROJECT_ID } from "../types";

export default {
  name: "ProvideDashboardContext",
  async setup() {
    const store = useStore();

    await store.dispatch("member/fetchMemberList");
    // fetchPrincipalList relies on fetchMemberList to find the role of the principal.
    await store.dispatch("principal/fetchPrincipalList");
    await store.dispatch("environment/fetchEnvironmentList");

    await Promise.all([
      // The default project hosts databases not explicitly assigned to other users project.
      store.dispatch("project/fetchProjectById", DEFAULT_PROJECT_ID),
      store.dispatch("uistate/restoreState"),
    ]);

    // Refresh the user after fetchMemberList / fetchPrincipalList.
    // The user info may change remotely and thus we need to refresh both in-memory and local cache state.
    store.dispatch("auth/refreshUser").then((currentUser: Principal) => {
      // In order to display the empty/non-empty inbox icon on the sidebar properly.
      store.dispatch("message/fetchMessageListByUser", currentUser.id);
    });
  },
};
</script>
