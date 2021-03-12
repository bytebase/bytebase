<template>
  <slot />
</template>

<script lang="ts">
import { useStore } from "vuex";

export default {
  name: "ProvideContext",
  async setup() {
    const store = useStore();

    await store.dispatch("principal/fetchPrincipalList");
    // This depends on principal/fetchPrincipalList
    await store.dispatch("roleMapping/fetchRoleMappingList");

    await store.dispatch("environment/fetchEnvironmentList");

    await store.dispatch("uistate/restoreState");
  },
};
</script>
