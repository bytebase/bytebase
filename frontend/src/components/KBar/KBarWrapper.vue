<template>
  <KBarProvider
    :actions="globalActions"
    :options="{ placeholder, disabled, compare }"
  >
    <KBarPortal>
      <KBarPositioner class="bb-kbar-mask">
        <KBarAnimator class="bb-kbar-container">
          <KBarSearch class="bb-kbar-searchbox" />
          <RenderResults />
          <KBarFooter />
        </KBarAnimator>
      </KBarPositioner>
    </KBarPortal>

    <KBarHelper />
    <slot />
  </KBarProvider>
</template>

<script lang="ts">
import {
  KBarProvider,
  KBarPortal,
  KBarPositioner,
  KBarAnimator,
  KBarSearch,
  defineAction,
} from "@bytebase/vue-kbar";
import { defineComponent, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useCurrentUserV1 } from "@/store";
import { UNKNOWN_USER_NAME } from "@/types";
import { useModalStackStatus } from "../../bbkit/BBModalStack.vue";
import KBarFooter from "./KBarFooter.vue";
import KBarHelper from "./KBarHelper.vue";
import RenderResults from "./RenderResults.vue";
import { compareAction as compare } from "./utils";

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
    const { t } = useI18n();
    const router = useRouter();
    const modalStack = useModalStackStatus();

    const placeholder = computed(() => t("kbar.options.placeholder"));

    const disabled = computed(() => {
      if (modalStack.value.length > 0) {
        // Disable kbar when any modal dialog is shown
        // We don't want to show modal dialogs and kbar at the same time
        // This also avoids navigating through kbar, which may
        // cause unexpected results
        return true;
      }

      const currentUserV1 = useCurrentUserV1();
      // totally disable kbar when not logged in
      // since there is no point to show it on signin/signup pages
      return currentUserV1.value.name === UNKNOWN_USER_NAME;
    });

    const globalActions = computed(() => [
      defineAction({
        id: "bb.navigation.home",
        name: "Home",
        shortcut: ["g", "h"],
        section: t("kbar.navigation"),
        keywords: "navigation",
        perform: () => router.push({ name: "workspace.home" }),
      }),
      defineAction({
        id: "bb.navigation.anomaly-center",
        name: "Anomaly Center",
        shortcut: ["g", "a", "c"],
        section: t("kbar.navigation"),
        keywords: "navigation",
        perform: () => router.push({ name: "workspace.anomaly-center" }),
      }),
    ]);

    return {
      globalActions,
      placeholder,
      disabled,
      compare,
    };
  },
});
</script>

<style scoped>
.bb-kbar-mask {
  @apply bg-gray-300 bg-opacity-80;
}
.bb-kbar-container {
  @apply bg-white shadow-lg rounded-lg w-128 overflow-hidden divide-y;
}
.bb-kbar-searchbox {
  @apply px-4 py-4 text-lg w-full box-border outline-none border-none;
}
</style>
