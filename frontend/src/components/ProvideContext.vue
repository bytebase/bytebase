<template>
  <slot />
</template>

<script lang="ts">
import { useStore } from "vuex";
import { Principal } from "../types";

export default {
  name: "ProvideContext",
  async setup() {
    const store = useStore();

    // Group the await to prepare the context in one batch.
    // If we split the await one by one, the execution flow will be interrupted
    // by router.beforeEach, which will pre-maturely fetches data relying on the
    // context here.
    await Promise.all([
      store.dispatch("environment/fetchEnvironmentList"),
      store.dispatch("roleMapping/fetchRoleMappingList"),
      // fetchPrincipalList relies on fetchRoleMappingList to find the role of the principal.
      store.dispatch("principal/fetchPrincipalList"),
      store.dispatch("instance/fetchInstanceList"),
      // [TODO] Fetches all databases. This may cause performance issue if the entire fleet
      // has a lot of databases.
      // The purpose it serves is our task list in the home view displays the database name column.
      // User with the Developer role may subscribe to the task, while she is not granted to that task's
      // related database. In such case, user won't fetch that task by her user id, so we need to do
      // a separate fetch somewhere. Another way to do it is to do on-demand fetch for each task whose
      // database hasn't been fetched yet, that solution is more complex and might not bring better
      // performance because it requires to do more round-trips because of the N+1 nature. Thus,
      // we choose to do a full fetch when initializing the context here.
      // [NOTE] This only fetches the database info itself, won't fetch the instance and especially
      // the data source info which contains connection credentials might not be granted to the current
      // user.
      store.dispatch("database/fetchDatabaseList"),
      store.dispatch("uistate/restoreState"),
    ]);

    // Refresh the user after fetchRoleMappingList / fetchPrincipalList.
    // The user info may change remotely and thus we need to refresh both in-memory and local cache state.
    store.dispatch("auth/refreshUser").then((currentUser: Principal) => {
      // In order to display the empty/non-empty inbox icon on the sidebar properly.
      store.dispatch("message/fetchMessageListByUser", currentUser.id);
    });
  },
};
</script>
