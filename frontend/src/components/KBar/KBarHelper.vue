<template>
  <slot />
</template>

<script lang="ts">
import { useKBarHandler } from "@bytebase/vue-kbar";
import { defineComponent, watch } from "vue";
import { useRoute } from "vue-router";
import { useRecentVisitActions } from "./useRecentVisit";

export default defineComponent({
  name: "KBarFooter",
  setup() {
    const handler = useKBarHandler();
    const route = useRoute();
    watch(
      () => route.fullPath,
      () => {
        // force hide kbar when page navigated
        handler.value.hide();
      }
    );

    // dynamically create Recent Visit kbar actions
    // recorded by `useRecentVisitHistory`
    useRecentVisitActions();
  },
});
</script>
