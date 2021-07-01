<template>
  <slot />
</template>

<script lang="ts">
import { useStore } from "vuex";

export default {
  name: "ProvideAppContext",
  async setup() {
    const store = useStore();

    await Promise.all([store.dispatch("actuator/info")]);
  },
};
</script>
