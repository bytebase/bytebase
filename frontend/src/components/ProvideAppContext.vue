<template>
  <slot />
</template>

<script lang="ts">
import { useStore } from "vuex";

export default {
  name: "ProvideAppContext",
  async setup() {
    const store = useStore();

    // Restore login user first.
    await store.dispatch("auth/restoreUser");

    await Promise.all([
      store.dispatch("actuator/info"),
      store.dispatch("plan/fetchCurrentPlan"),
    ]);
  },
};
</script>
