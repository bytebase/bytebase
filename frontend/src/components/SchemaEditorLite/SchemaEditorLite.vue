<template>
  <div
    class="bb-schema-editor w-full h-full flex flex-col border rounded-lg overflow-hidden relative"
    v-bind="$attrs"
  >
    <MaskSpinner v-if="combinedLoading" />
    <NSplit
      :min="0.15"
      :max="0.4"
      :default-size="0.25"
      :resize-trigger-size="1"
    >
      <template #1>
        <Aside
          v-if="ready"
          @update-is-editing="$emit('update-is-editing', $event)"
        />
      </template>
      <template #2>
        <Editor v-if="ready" />
      </template>
    </NSplit>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { NSplit } from "naive-ui";
import { computed, toRef } from "vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import Aside from "./Aside";
import { useAlgorithm } from "./algorithm";
import { provideSchemaEditorContext } from "./context";
import Editor from "./Editor.vue";
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

const targets = computed(() => {
  return props.targets ?? [];
});

const ready = computed(() => {
  return targets.value.length > 0;
});

const combinedLoading = computed(() => {
  return props.loading || !ready.value;
});

const context = provideSchemaEditorContext({
  targets,
  project: toRef(props, "project"),
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
