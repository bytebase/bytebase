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
import { cloneDeep } from "lodash-es";
import { Splitpanes, Pane } from "splitpanes";
import "splitpanes/dist/splitpanes.css";
import { computed, onMounted, toRef } from "vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useSettingV1Store } from "@/store";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import Aside from "./Aside";
import Editor from "./Editor.vue";
import { useAlgorithm } from "./algorithm";
import { provideSchemaEditorContext } from "./context";
import type { EditTarget, RolloutObject } from "./types";

const props = defineProps<{
  project: Project;
  readonly?: boolean;
  selectedRolloutObjects?: RolloutObject[];
  targets?: EditTarget[];
  loading?: boolean;
  hidePreview?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:selected-rollout-objects", objects: RolloutObject[]): void;
  (event: "update-is-editing", objects: RolloutObject[]): void;
}>();

const settingStore = useSettingV1Store();

// Prepare schema template contexts.
onMounted(async () => {
  await settingStore.getOrFetchSettingByName(
    Setting_SettingName.SCHEMA_TEMPLATE
  );
});

const targets = computed(() => {
  return props.targets ?? [];
});

const ready = computed(() => {
  return targets.value.length > 0;
});

const combinedLoading = computed(() => {
  return props.loading || !ready.value;
});

const classificationConfig = computed(() => {
  if (!props.project.dataClassificationConfigId) {
    return;
  }
  return settingStore.getProjectClassification(
    props.project.dataClassificationConfigId
  );
});

const context = provideSchemaEditorContext({
  targets,
  project: toRef(props, "project"),
  classificationConfig,
  readonly: toRef(props, "readonly"),
  selectedRolloutObjects: toRef(props, "selectedRolloutObjects"),
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

useEmitteryEventListener(
  context.events,
  "update:selected-rollout-objects",
  (objects) => {
    emit("update:selected-rollout-objects", objects);
  }
);

useEmitteryEventListener(context.events, "merge-metadata", (metadatas) => {
  for (const metadata of metadatas) {
    const target = props.targets?.find(
      (t) => t.metadata.name === metadata.name
    );
    if (!target) {
      continue;
    }

    mergeTableMetadataToTarge({
      metadata,
      mergeTo: target.metadata,
    });
    mergeTableMetadataToTarge({
      metadata,
      mergeTo: target.baselineMetadata,
    });
  }
});

const mergeTableMetadataToTarge = ({
  metadata,
  mergeTo,
}: {
  metadata: DatabaseMetadata;
  mergeTo: DatabaseMetadata;
}) => {
  for (const schema of metadata.schemas) {
    if (schema.tables.length === 0) {
      continue;
    }
    const targetSchema = mergeTo.schemas.find((s) => s.name === schema.name);
    if (!targetSchema) {
      continue;
    }

    for (const table of schema.tables) {
      if (!targetSchema.tables.find((t) => t.name === table.name)) {
        targetSchema.tables.push(cloneDeep(table));
      }
    }
  }
};

const refreshPreview = () => {
  context.events.emit("refresh-preview");
};

defineExpose({
  applyMetadataEdit,
  refreshPreview,
  isDirty: context.isDirty,
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
