<template>
  <button
    class="p-0.5 rounded hover:bg-gray-200"
    :class="[isFocused && focusedClass, isFocused ? '!visible' : 'invisible']"
    @click="toggleFocus"
  >
    <!-- TODO(Jim): replace this raw SVG with heroicons-outline:viewfinder-circle when supported -->
    <FocusIcon class="w-4 h-4" />
  </button>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import type { TableMetadata } from "@/types/proto/store/database";
import FocusIcon from "./FocusIcon.vue";
import { expectedZoomRange, useSchemaDiagramContext } from ".";
import { DEFAULT_PADDINGS } from "./const";
import { VueClass } from "@/utils";

const props = withDefaults(
  defineProps<{
    table: TableMetadata;
    focusedClass?: VueClass;
  }>(),
  {
    focusedClass: "",
  }
);

const emits = defineEmits<{
  (name: "toggle", on: boolean, e: Event): void;
}>();

const { zoom, focusedTables, events } = useSchemaDiagramContext();

const isFocused = computed(() => {
  return focusedTables.value.has(props.table);
});

const toggleFocus = (e: Event) => {
  const oldValue = isFocused.value;
  if (oldValue) {
    focusedTables.value.delete(props.table);
  } else {
    focusedTables.value.add(props.table);
    events.emit("set-center", {
      type: "table",
      target: props.table,
      padding: DEFAULT_PADDINGS,
      zooms: expectedZoomRange(zoom.value, 0.5, 1),
    });
  }
  emits("toggle", !oldValue, e);
};
</script>
