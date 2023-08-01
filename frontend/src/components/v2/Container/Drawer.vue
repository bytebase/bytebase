<template>
  <NDrawer
    width="auto"
    :show="show"
    :close-on-esc="true"
    @update:show="onUpdateShow"
  >
    <slot />
  </NDrawer>
</template>

<script setup lang="ts">
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
</script>
