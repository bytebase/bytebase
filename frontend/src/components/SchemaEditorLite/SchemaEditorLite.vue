<template>
  <div
    class="w-full h-full flex flex-col border rounded-lg overflow-hidden relative"
    v-bind="$attrs"
  >
    <MaskSpinner v-if="mergedLoading" />
    <Splitpanes
      v-if="state.initialized"
      class="default-theme w-full flex-1 flex flex-row overflow-hidden relative"
    >
      <Pane min-size="15" size="25">
        <Aside />
      </Pane>
      <Pane min-size="60" size="75">
        <Editor />
      </Pane>
    </Splitpanes>
  </div>
</template>

<script lang="ts" setup>
import { Splitpanes, Pane } from "splitpanes";
import { reactive, computed, onMounted, toRef } from "vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { useDatabaseV1Store, useSettingV1Store } from "@/store";
import { ComposedProject, ComposedDatabase } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import Aside from "./Aside";
import Editor from "./Editor.vue";
import { useAlgorithm } from "./algorithm";
import { provideSchemaEditorContext } from "./context";
import { EditTarget } from "./types";

const props = defineProps<{
  project: ComposedProject;
  resourceType: "database" | "branch";
  readonly?: boolean;
  databases?: ComposedDatabase[];
  // NOTE: we only support editing one branch for now.
  branch: Branch;
  loading?: boolean;
}>();

interface LocalState {
  loading: boolean;
  initialized: boolean;
}

const settingStore = useSettingV1Store();
const state = reactive<LocalState>({
  loading: false,
  initialized: false,
});

const mergedLoading = computed(() => {
  return props.loading || state.loading;
});

// Prepare schema template contexts.
onMounted(async () => {
  await settingStore.getOrFetchSettingByName("bb.workspace.schema-template");

  // TODO: generate initial diff state
  state.initialized = true;
});

const targets = computed(() => {
  if (props.resourceType === "database") {
    return (props.databases ?? []).map<EditTarget>((database) => ({
      database,
      metadata: DatabaseMetadata.fromPartial({}), // TODO,
      baselineMetadata: DatabaseMetadata.fromPartial({}),
    }));
  }
  if (props.resourceType === "branch") {
    const { branch } = props;
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

const context = provideSchemaEditorContext({
  targets,
  project: toRef(props, "project"),
  resourceType: toRef(props, "resourceType"),
  readonly: toRef(props, "readonly"),
});
const { rebuildMetadataEdit, applyMetadataEdit } = useAlgorithm(context);

defineExpose({
  rebuildMetadataEdit,
  applyMetadataEdit,
});
</script>

<style>
@import "splitpanes/dist/splitpanes.css";

/* splitpanes pane style */
.splitpanes.default-theme .splitpanes__pane {
  @apply bg-transparent !transition-none;
}

.splitpanes.default-theme .splitpanes__splitter {
  @apply bg-gray-100 border-none;
}

.splitpanes.default-theme .splitpanes__splitter:hover {
  @apply bg-indigo-300;
}

.splitpanes.default-theme .splitpanes__splitter::before,
.splitpanes.default-theme .splitpanes__splitter::after {
  @apply bg-gray-700 opacity-50 text-white;
}

.splitpanes.default-theme .splitpanes__splitter:hover::before,
.splitpanes.default-theme .splitpanes__splitter:hover::after {
  @apply bg-white opacity-100;
}
</style>
