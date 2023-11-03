<template>
  <div
    ref="popoverRef"
    v-zindexable="{ enabled: true }"
    class="fixed border border-gray-100 rounded bg-white p-2 shadow transition-[top] text-sm"
    :class="[show ? 'visible' : ' invisible pointer-events-none']"
    :style="{
      left: `${position.x - popoverWidth + offsetX}px`,
      top: `${position.y + offsetY}px`,
    }"
  >
    <template v-if="state">
      <ColumnInfo
        v-if="state.column"
        :db="state.db"
        :database="state.database"
        :schema="state.schema"
        :table="state.table"
        :column="state.column"
      />
      <TableInfo
        v-else
        :db="state.db"
        :database="state.database"
        :schema="state.schema"
        :table="state.table"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { onClickOutside, useElementSize, useEventListener } from "@vueuse/core";
import { zindexable as vZindexable } from "vdirs";
import { computed, ref } from "vue";
import ColumnInfo from "./ColumnInfo.vue";
import TableInfo from "./TableInfo.vue";
import { useHoverStateContext } from "./hover-state";

defineProps<{
  offsetX: number;
  offsetY: number;
}>();

const emit = defineEmits<{
  (event: "click-outside", e: MouseEvent): void;
}>();

const { state, position, update } = useHoverStateContext();

const popoverRef = ref<HTMLDivElement>();
onClickOutside(popoverRef, (e) => {
  emit("click-outside", e);
});
const { width: popoverWidth } = useElementSize(popoverRef, undefined, {
  box: "border-box",
});

const show = computed(
  () =>
    state.value !== undefined &&
    position.value.x !== 0 &&
    position.value.y !== 0
);

useEventListener(popoverRef, "mouseenter", () => {
  // Reset the value immediately to cancel other pending setting values
  update(state.value, "before", 0 /* overrideDelay */);
});
useEventListener(popoverRef, "mouseleave", () => {
  update(undefined, "before");
});
</script>
