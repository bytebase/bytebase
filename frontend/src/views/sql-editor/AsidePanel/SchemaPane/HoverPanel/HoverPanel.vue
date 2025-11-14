<template>
  <div
    ref="popoverRef"
    v-zindexable="{ enabled: true }"
    class="fixed border border-gray-100 rounded-sm bg-white p-2 shadow-sm transition-[top] text-sm"
    :class="[show ? 'visible' : ' invisible pointer-events-none']"
    :style="{
      left: `${displayPosition.x}px`,
      top: `${displayPosition.y}px`,
    }"
  >
    <template v-if="state">
      <ColumnInfo
        v-if="state.column && state.table"
        :database="state.database"
        :schema="state.schema"
        :table="state.table"
        :column="state.column"
      />
      <TablePartitionInfo
        v-else-if="state.table && state.partition"
        :database="state.database"
        :schema="state.schema"
        :table="state.table"
        :partition="state.partition"
      />
      <TableInfo
        v-else-if="state.table"
        :database="state.database"
        :schema="state.schema"
        :table="state.table"
      />
      <ExternalTableInfo
        v-else-if="state.externalTable"
        :database="state.database"
        :schema="state.schema"
        :external-table="state.externalTable"
      />
      <ViewInfo
        v-else-if="state.view"
        :database="state.database"
        :schema="state.schema"
        :view="state.view"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { onClickOutside, useElementSize, useEventListener } from "@vueuse/core";
import { zindexable as vZindexable } from "vdirs";
import { computed, ref } from "vue";
import type { Position } from "@/types";
import { minmax } from "@/utils";
import ColumnInfo from "./ColumnInfo.vue";
import ExternalTableInfo from "./ExternalTableInfo.vue";
import { useHoverStateContext } from "./hover-state";
import TableInfo from "./TableInfo.vue";
import TablePartitionInfo from "./TablePartitionInfo.vue";
import ViewInfo from "./ViewInfo.vue";

const props = defineProps<{
  offsetX: number;
  offsetY: number;
  margin: number;
}>();

const emit = defineEmits<{
  (event: "click-outside", e: MouseEvent): void;
}>();

const { state, position, update } = useHoverStateContext();

const popoverRef = ref<HTMLDivElement>();
onClickOutside(popoverRef, (e) => {
  emit("click-outside", e);
});
const { height: popoverHeight } = useElementSize(popoverRef, undefined, {
  box: "border-box",
});

const show = computed(() => {
  const show =
    state.value !== undefined &&
    position.value.x !== 0 &&
    position.value.y !== 0;
  return show;
});

const displayPosition = computed(() => {
  const p: Position = {
    x: position.value.x + props.offsetX,
    y: position.value.y + props.offsetY,
  };
  const yMin = props.margin;
  const yMax = window.innerHeight - popoverHeight.value - props.margin;
  p.y = minmax(p.y, yMin, yMax);

  return p;
});

useEventListener(popoverRef, "mouseenter", () => {
  // Keep the hover panel visible with a small delay to prevent flicker
  update(state.value, "before", 50);
});
useEventListener(popoverRef, "mouseleave", () => {
  update(undefined, "after");
});
</script>
