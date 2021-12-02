<template>
  <slot />
</template>

<script lang="ts">
import { useKBarHandler } from "@bytebase/vue-kbar";
import { defineComponent, watch } from "vue";
import { useRoute } from "vue-router";

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
  },
});
</script>
