<template>
  <div v-if="state.initialized" class="w-full h-[32rem] border rounded-lg">
    <Splitpanes
      class="default-theme w-full h-full flex flex-row overflow-hidden"
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
import { onMounted, watch } from "vue";
import { reactive } from "vue";
import { useSchemaEditorV1Store, useSettingV1Store } from "@/store";
import { ComposedProject, ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import Aside from "./Aside/index.vue";
import Editor from "./Editor.vue";
import { convertBranchToBranchSchema } from "./utils/branch";

const props = defineProps<{
  readonly: boolean;
  engine: Engine;
  project: ComposedProject;
  baselineSchemaMetadata?: DatabaseMetadata;
  schemaMetadata?: DatabaseMetadata;
  resourceType: "database" | "branch";
  databases?: ComposedDatabase[];
  // NOTE: we only support editing one branch for now.
  branches?: SchemaDesign[];
}>();

interface LocalState {
  initialized: boolean;
}

const settingStore = useSettingV1Store();
const schemaEditorV1Store = useSchemaEditorV1Store();
const state = reactive<LocalState>({
  initialized: false,
});

const updateSchemaEditorState = () => {
  schemaEditorV1Store.setState({
    engine: props.engine,
    project: props.project,
    readonly: props.readonly,
    resourceType: props.resourceType,
  });

  if (props.resourceType === "database") {
    schemaEditorV1Store.setState({
      resourceMap: {
        // NOTE: we will dynamically fetch schema list for each database in database tree view.
        database: new Map(
          (props.databases || []).map((database) => [
            database.name,
            {
              database,
              schemaList: [],
              originSchemaList: [],
            },
          ])
        ),
        branch: new Map(),
      },
    });
  } else {
    schemaEditorV1Store.setState({
      resourceMap: {
        database: new Map(),
        branch: new Map(
          (props.branches || []).map((branch) => [
            branch.name,
            convertBranchToBranchSchema(branch),
          ])
        ),
      },
    });
  }
};

// Prepare schema template contexts.
onMounted(async () => {
  await settingStore.getOrFetchSettingByName("bb.workspace.schema-template");
  updateSchemaEditorState();
  state.initialized = true;
});

watch(
  () => props,
  () => {
    updateSchemaEditorState();
  },
  {
    deep: true,
  }
);
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
