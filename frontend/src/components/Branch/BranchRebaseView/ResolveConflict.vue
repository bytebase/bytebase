<template>
  <div
    class="w-full flex-1 relative text-sm border rounded overflow-clip bb-resolve-conflict-editor"
  >
    <MonacoEditor
      v-model:content="state.editingSchema"
      :line-highlights="lineHighlightOptions"
      class="h-full"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import {
  LineHighlightOption,
  LineHighlightOverviewRulerPosition,
  MonacoEditor,
} from "@/components/MonacoEditor";
import { ComposedProject } from "@/types";
import { RebaseBranchValidationState } from "./types";

type LocalState = {
  editingSchema: string;
};
const ConflictMarkers = [
  {
    pattern: "<<<<<<",
    type: "current",
    color: "rgb(16 185 129)", // bg-emerald-500
    position: "LEFT" as LineHighlightOverviewRulerPosition,
  },
  {
    pattern: "======",
    type: "gutter",
    color: "rgb(107 114 128)", // bg-gray-500
    position: "FULL" as LineHighlightOverviewRulerPosition,
  },
  {
    pattern: ">>>>>>",
    type: "incoming",
    color: "rgb(59 130 246)", // bg-blue-500
    position: "RIGHT" as LineHighlightOverviewRulerPosition,
  },
];

const props = defineProps<{
  project: ComposedProject;
  validationState: RebaseBranchValidationState;
}>();

const state = reactive<LocalState>({
  editingSchema: "",
});

const lineHighlightOptions = computed(() => {
  const options: LineHighlightOption[] = [];
  state.editingSchema.split("\n").forEach((line, index) => {
    const marker = ConflictMarkers.find((m) => line.startsWith(m.pattern));
    if (marker) {
      options.push({
        lineNumber: index + 1,
        className: `bb-conflict-line--${marker.type}`,
        overviewRuler: {
          color: marker.color,
          position: marker.position,
        },
      });
    }
  });
  return options;
});

const validateConflictSchema = () => {
  if (props.validationState.branch) {
    // Auto rebase
    return {
      valid: true,
      schema: undefined,
    };
  }
  if (lineHighlightOptions.value.length === 0) {
    // Conflict resolved
    return {
      valid: true,
      schema: state.editingSchema,
    };
  }
  return {
    valid: false,
    schema: state.editingSchema,
  };
};
watch(
  () => props.validationState.conflictSchema,
  (conflictSchema) => {
    state.editingSchema = conflictSchema ?? "";
  },
  {
    immediate: true,
  }
);

defineExpose({ validateConflictSchema });
</script>

<style lang="postcss" scoped>
.bb-resolve-conflict-editor :deep(.bb-conflict-line--current) {
  @apply bg-emerald-300;
}
.bb-resolve-conflict-editor :deep(.bb-conflict-line--gutter) {
  @apply bg-gray-300;
}
.bb-resolve-conflict-editor :deep(.bb-conflict-line--incoming) {
  @apply bg-blue-300;
}
</style>
