<template>
  <slot />
</template>

<script lang="ts">
import { provide, computed } from "vue";
import { useStore } from "vuex";
import { User } from "../types";

// We use symbols as unique identifiers.
export const UserStateSymbol = Symbol("user");

export default {
  name: "ProvideUser",
  setup() {
    const store = useStore();

    const currentUser: User = computed(() =>
      store.getters["auth/currentUser"]()
    )?.value;
    provide(UserStateSymbol, currentUser);
  },
};
</script>
