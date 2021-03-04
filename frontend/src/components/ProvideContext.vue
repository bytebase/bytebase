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
    await store.dispatch("environment/fetchEnvironmentList");
  },
};
</script>
