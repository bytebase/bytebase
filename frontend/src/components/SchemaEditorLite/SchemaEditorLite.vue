<template>
  <div
    class="w-full h-full flex flex-col border rounded-lg overflow-hidden relative"
    v-bind="$attrs"
  >
    <MaskSpinner v-if="mergedLoading" />
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
import { reactive, computed, onMounted, toRef, watch } from "vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { useDatabaseV1Store, useSettingV1Store } from "@/store";
import { ComposedProject } from "@/types";
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
  targets?: EditTarget[];
  // NOTE: we only support editing one branch for now.
  branch?: Branch;
  loading?: boolean;
  diffWhenReady?: boolean;
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

const mergedLoading = computed(() => {
  return props.loading || state.diffing || !ready.value;
});

const context = provideSchemaEditorContext({
  targets,
  project: toRef(props, "project"),
  resourceType: toRef(props, "resourceType"),
  readonly: toRef(props, "readonly"),
});
const { rebuildMetadataEdit, applyMetadataEdit } = useAlgorithm(context);

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
