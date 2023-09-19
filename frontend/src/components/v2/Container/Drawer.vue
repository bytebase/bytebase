<template>
  <NDrawer
    width="auto"
    :show="show"
    :auto-focus="false"
    :trap-focus="false"
    :close-on-esc="true"
    v-bind="$attrs"
    @update:show="onUpdateShow"
  >
    <slot />
  </NDrawer>
</template>

<script setup lang="ts">
import { useEventListener } from "@vueuse/core";
import { NDrawer } from "naive-ui";

withDefaults(
  defineProps<{
    show?: boolean;
  }>(),
  {
    show: true,
  }
);
const emit = defineEmits<{
  (event: "update:show", show: boolean): void;
  (event: "close"): void;
}>();

const onUpdateShow = (show: boolean) => {
  emit("update:show", show);

  if (!show) {
    emit("close");
  }
};

useEventListener("keydown", (e) => {
  if (e.code == "Escape") {
    emit("update:show", false);
    emit("close");
  }
});
</script>
