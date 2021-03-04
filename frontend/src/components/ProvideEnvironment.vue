<template>
  <slot />
</template>

<script lang="ts">
import { useStore } from "vuex";

export default {
  name: "ProvideEnvironment",
  async setup() {
    const store = useStore();

    await store.dispatch("environment/fetchEnvironmentList");
  },
};
</script>
