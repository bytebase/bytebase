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
      store.dispatch("plan/fetchCurrentPlan"),
      // Fetch member and principal list so PrincipalSelect can have the data.
      store.dispatch("member/fetchMemberList"),
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
