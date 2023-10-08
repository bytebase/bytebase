<template>
  <NDrawer
    width="auto"
    :show="show"
    :auto-focus="false"
    :trap-focus="false"
    :close-on-esc="closeOnEsc"
    v-bind="$attrs"
    @update:show="onUpdateShow"
  >
    <slot />
  </NDrawer>
</template>

<script setup lang="ts">
import { useEventListener } from "@vueuse/core";
import { NDrawer } from "naive-ui";

const props = withDefaults(
  defineProps<{
    show?: boolean;
    closeOnEsc?: boolean;
  }>(),
  {
    show: true,
    closeOnEsc: true,
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
    if (!props.closeOnEsc) return;
    emit("update:show", false);
    emit("close");
  }
});
</script>
