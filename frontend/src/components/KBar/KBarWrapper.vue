<template>
  <KBarProvider :actions="globalActions" :options="{ disabled }">
    <KBarPortal>
      <KBarPositioner class="mask">
        <KBarAnimator class="container">
          <KBarSearch class="searchbox" />
          <KBarHelper />
          <RenderResults />
          <KBarFooter />
        </KBarAnimator>
      </KBarPositioner>
    </KBarPortal>

    <slot />
  </KBarProvider>
</template>

<script lang="ts">
import { defineComponent, computed } from "vue";
import {
  KBarProvider,
  KBarPortal,
  KBarPositioner,
  KBarAnimator,
  KBarSearch,
  defineAction,
} from "@bytebase/vue-kbar";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import RenderResults from "./RenderResults.vue";
import KBarHelper from "./KBarHelper.vue";
import KBarFooter from "./KBarFooter.vue";

export default defineComponent({
  name: "KBarWrapper",
  components: {
    KBarProvider,
    KBarPortal,
    KBarPositioner,
    KBarAnimator,
    KBarSearch,
    RenderResults,
    KBarHelper,
    KBarFooter,
  },
  setup() {
    const store = useStore();
    const router = useRouter();

    const disabled = computed(() => {
      const currentUser = store.getters["auth/currentUser"]();
      // totally disable kbar when not logged in
      // since there is no point to show it on signin/signup pages
      return !currentUser || currentUser.id < 0;
    });

    const globalActions = [
      defineAction({
        id: "bb.navigation.home",
        name: "Home",
        shortcut: ["g", "h"],
        section: "Navigation",
        perform: () => router.push({ name: "workspace.home" }),
      }),
      defineAction({
        id: "bb.navigation.anomaly-center",
        name: "Anomaly Center",
        shortcut: ["g", "a", "c"],
        section: "Navigation",
        perform: () => router.push({ name: "workspace.anomaly-center" }),
      }),
    ];

    return {
      globalActions,
      disabled,
    };
  },
});
</script>

<style scoped>
.mask {
  @apply bg-gray-300 bg-opacity-80;
}
.container {
  @apply bg-white shadow-lg rounded-lg w-128 overflow-hidden divide-y;
}
.searchbox {
  @apply px-4 py-4 text-lg w-full box-border outline-none border-none;
}
</style>
