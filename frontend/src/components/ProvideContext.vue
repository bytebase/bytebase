<template>
  <slot />
</template>

<script lang="ts">
import { computed, ComputedRef } from "vue";
import { useStore } from "vuex";
import { User } from "../types";

export default {
  name: "ProvideContext",
  async setup() {
    const store = useStore();

    const currentUser: ComputedRef<User> = computed(() =>
      store.getters["auth/currentUser"]()
    );

    await store.dispatch("principal/fetchPrincipalList");
    await store.dispatch("environment/fetchEnvironmentList");
    await store.dispatch(
      "bookmark/fetchBookmarkListByUser",
      currentUser.value.id
    );
    await store.dispatch("roleMapping/fetchRoleMappingList");
    await store.dispatch("uistate/restoreExpandState");
  },
};
</script>
