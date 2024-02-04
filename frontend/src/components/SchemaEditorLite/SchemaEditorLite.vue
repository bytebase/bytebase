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
        <Aside v-if="ready" />
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
import { reactive, computed, onMounted, toRef, watch } from "vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useDatabaseV1Store, useSettingV1Store } from "@/store";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import Aside from "./Aside";
import Editor from "./Editor.vue";
import { useAlgorithm } from "./algorithm";
import { provideSchemaEditorContext } from "./context";
import { EditTarget, RolloutObject } from "./types";

const props = defineProps<{
  project: ComposedProject;
  resourceType: "database" | "branch";
  readonly?: boolean;
  selectedRolloutObjects?: RolloutObject[];
  targets?: EditTarget[];
  // NOTE: we only support editing one branch for now.
  branch?: Branch;
  loading?: boolean;
  diffWhenReady?: boolean;
  disableDiffColoring?: boolean;
}>();
const emit = defineEmits<{
  (event: "update:selected-rollout-objects", objects: RolloutObject[]): void;
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
  await settingStore.getOrFetchSettingByName("bb.workspace.schema-template");
  state.initialized = true;
});

const targets = computed(() => {
  if (props.resourceType === "database") {
    return props.targets ?? [];
  }
  if (props.resourceType === "branch") {
    const { branch } = props;
    if (!branch) return [];
    const target: EditTarget = {
      database: useDatabaseV1Store().getDatabaseByName(branch.baselineDatabase),
      metadata: branch.schemaMetadata ?? DatabaseMetadata.fromPartial({}),
      baselineMetadata:
        branch.baselineSchemaMetadata ?? DatabaseMetadata.fromPartial({}),
    };
    return [target];
  }
  return [];
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
  resourceType: toRef(props, "resourceType"),
  readonly: toRef(props, "readonly"),
  selectedRolloutObjects: toRef(props, "selectedRolloutObjects"),
  disableDiffColoring: toRef(props, "disableDiffColoring"),
});
const { rebuildMetadataEdit, applyMetadataEdit, applySelectedMetadataEdit } =
  useAlgorithm(context);

useEmitteryEventListener(context.events, "rebuild-edit-status", (params) => {
  if (
    ready.value &&
    (props.diffWhenReady || props.resourceType === "database")
  ) {
    targets.value.forEach((target) => {
      rebuildMetadataEdit(
        target.database,
        target.baselineMetadata,
        target.metadata,
        params.resets
      );
    });
  }
});

watch(
  [ready, () => props.diffWhenReady],
  ([ready, diffWhenReady]) => {
    if (ready && diffWhenReady) {
      targets.value.forEach((target) => {
        rebuildMetadataEdit(
          target.database,
          target.baselineMetadata,
          target.metadata
        );
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

defineExpose({
  rebuildMetadataEdit,
  applyMetadataEdit,
  applySelectedMetadataEdit,
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
