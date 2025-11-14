<template>
  <button
    class="p-0.5 rounded-sm hover:bg-gray-200"
    :class="[isFocused && focusedClass, isFocused ? 'visible!' : 'invisible']"
    @click="toggleFocus"
  >
    <!-- TODO(Jim): replace this raw SVG with heroicons-outline:viewfinder-circle when supported -->
    <FocusIcon class="w-4 h-4" />
  </button>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { TableMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { VueClass } from "@/utils";
import { expectedZoomRange, useSchemaDiagramContext } from ".";
import { DEFAULT_PADDINGS } from "./const";
import FocusIcon from "./FocusIcon.vue";

const props = withDefaults(
  defineProps<{
    table: TableMetadata;
    focusedClass?: VueClass;
    setCenter?: boolean;
  }>(),
  {
    focusedClass: "",
    setCenter: true,
  }
);

const { zoom, focusedTables, events } = useSchemaDiagramContext();

const isFocused = computed(() => {
  return focusedTables.value.has(props.table);
});

const toggleFocus = (e: Event) => {
  e.stopPropagation();
  const on = !isFocused.value;
  if (on) {
    focusedTables.value.add(props.table);
  } else {
    focusedTables.value.delete(props.table);
  }
  if (props.setCenter) {
    events.emit("set-center", {
      type: "table",
      target: props.table,
      padding: DEFAULT_PADDINGS,
      zooms: on ? expectedZoomRange(zoom.value, 0.5, 1) : undefined,
    });
  }
};
</script>
