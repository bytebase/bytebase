<template>
  <div
    class="bb-schema-editor w-full h-full flex flex-col border rounded-lg overflow-hidden relative"
    v-bind="$attrs"
  >
    <MaskSpinner v-if="combinedLoading" />
    <Splitpanes
      class="default-theme w-full flex-1 flex flex-row overflow-hidden relative"
    >
      <Pane min-size="15" size="25">
        <Aside
          v-if="ready"
          @update-is-editing="$emit('update-is-editing', $event)"
        />
      </Pane>
      <Pane min-size="60" size="75">
        <Editor v-if="ready" />
      </Pane>
    </Splitpanes>
  </div>
</template>

<script lang="ts" setup>
import { Splitpanes, Pane } from "splitpanes";
import "splitpanes/dist/splitpanes.css";
import { reactive, computed, onMounted, toRef, ref, watch } from "vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useSettingV1Store } from "@/store";
import type { ComposedProject } from "@/types";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import Aside from "./Aside";
import Editor from "./Editor.vue";
import { useAlgorithm } from "./algorithm";
import { provideSchemaEditorContext } from "./context";
import type { EditTarget, RolloutObject } from "./types";

const props = defineProps<{
  project: ComposedProject;
  readonly?: boolean;
  selectedRolloutObjects?: RolloutObject[];
  targets?: EditTarget[];
  loading?: boolean;
  diffWhenReady?: boolean;
  disableDiffColoring?: boolean;
  hidePreview?: boolean;
}>();
const emit = defineEmits<{
  (event: "update:selected-rollout-objects", objects: RolloutObject[]): void;
  (event: "update-is-editing", objects: RolloutObject[]): void;
}>();

interface LocalState {
  diffing: boolean;
  initialized: boolean;
}

const settingStore = useSettingV1Store();
const state = reactive<LocalState>({
  diffing: false,
  initialized: false,
});

// Prepare schema template contexts.
onMounted(async () => {
  await settingStore.getOrFetchSettingByName(
    Setting_SettingName.SCHEMA_TEMPLATE
  );
  state.initialized = true;
});

const targets = computed(() => {
  return props.targets ?? [];
});

const ready = computed(() => {
  return state.initialized && targets.value.length > 0;
});

const combinedLoading = computed(() => {
  return props.loading || state.diffing || !ready.value;
});

const context = provideSchemaEditorContext({
  targets,
  project: toRef(props, "project"),
  classificationConfigId: ref(props.project.dataClassificationConfigId),
  readonly: toRef(props, "readonly"),
  selectedRolloutObjects: toRef(props, "selectedRolloutObjects"),
  disableDiffColoring: toRef(props, "disableDiffColoring"),
  hidePreview: toRef(props, "hidePreview"),
});
const { rebuildMetadataEdit, applyMetadataEdit } = useAlgorithm(context);

useEmitteryEventListener(context.events, "rebuild-edit-status", (params) => {
  if (ready.value) {
    targets.value.forEach((target) => {
      rebuildMetadataEdit(target, params.resets);
    });
  }
});

watch(
  [ready, () => props.diffWhenReady],
  ([ready, diffWhenReady]) => {
    if (ready && diffWhenReady) {
      targets.value.forEach((target) => {
        rebuildMetadataEdit(target);
      });
    }
  },
  { immediate: true }
);

useEmitteryEventListener(
  context.events,
  "update:selected-rollout-objects",
  (objects) => {
    emit("update:selected-rollout-objects", objects);
  }
);

const refreshPreview = () => {
  context.events.emit("refresh-preview");
};

defineExpose({
  rebuildMetadataEdit,
  applyMetadataEdit,
  refreshPreview,
});
</script>

<style lang="postcss" scoped>
/* splitpanes pane style */
.bb-schema-editor :deep(.splitpanes.default-theme .splitpanes__pane) {
  @apply bg-transparent !transition-none;
}

.bb-schema-editor :deep(.splitpanes.default-theme .splitpanes__splitter) {
  @apply bg-gray-100 border-none;
}

.bb-schema-editor :deep(.splitpanes.default-theme .splitpanes__splitter:hover) {
  @apply bg-indigo-300;
}

.bb-schema-editor
  :deep(.splitpanes.default-theme .splitpanes__splitter::before),
.bb-schema-editor
  :deep(.splitpanes.default-theme .splitpanes__splitter::after) {
  @apply bg-gray-700 opacity-50 text-white;
}

.bb-schema-editor
  :deep(.splitpanes.default-theme .splitpanes__splitter:hover::before),
.bb-schema-editor
  :deep(.splitpanes.default-theme .splitpanes__splitter:hover::after) {
  @apply bg-white opacity-100;
}
</style>
