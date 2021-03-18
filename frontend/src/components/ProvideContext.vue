<template>
  <slot />
</template>

<script lang="ts">
import { useStore } from "vuex";

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
      store.dispatch("principal/fetchPrincipalList"),
      store.dispatch("roleMapping/fetchRoleMappingList"),
      store.dispatch("instance/fetchInstanceList"),
      store.dispatch("uistate/restoreState"),
    ]);
  },
};
</script>
