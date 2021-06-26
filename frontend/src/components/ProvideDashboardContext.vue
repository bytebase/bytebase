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

    await store.dispatch("environment/fetchEnvironmentList");

    await Promise.all([
      // The default project hosts databases not explicitly assigned to other users project.
      store.dispatch("project/fetchProjectById", DEFAULT_PROJECT_ID),
      store.dispatch("uistate/restoreState"),
    ]);

    store.dispatch("message/fetchMessageListByUser", currentUser.value.id);
  },
};
</script>
