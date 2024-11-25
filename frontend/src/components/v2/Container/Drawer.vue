<template>
  <NDrawer
    width="auto"
    :show="show"
    :auto-focus="false"
    :trap-focus="false"
    :close-on-esc="false"
    :data-overlay-stack-id="id"
    :data-overlay-stack-upmost="upmost"
    v-bind="$attrs"
    @update:show="onUpdateShow"
  >
    <slot />
  </NDrawer>
</template>

<script setup lang="ts">
import { toRef } from "@vueuse/core";
import { NDrawer } from "naive-ui";
import { useOverlayStack } from "@/components/misc/OverlayStackManager.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";

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

const { id, upmost, events } = useOverlayStack(toRef(props, "show"));

useEmitteryEventListener(events, "esc", () => {
  if (upmost.value && props.closeOnEsc) {
    emit("update:show", false);
    emit("close");
  }
});
</script>
