<template>
  <div
    class="w-full h-full border rounded-lg overflow-hidden relative"
    v-bind="$attrs"
  >
    <MaskSpinner v-if="mergedLoading" />

    <Splitpanes
      v-if="state.initialized"
      class="default-theme w-full h-full flex flex-row overflow-hidden relative"
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
import { isEqual } from "lodash-es";
import { Splitpanes, Pane } from "splitpanes";
import { onMounted, watch, reactive, computed } from "vue";
import { useSchemaEditorV1Store, useSettingV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { ComposedProject, ComposedDatabase } from "@/types";
import {
  SchemaDesign,
  SchemaDesign_Type,
} from "@/types/proto/v1/schema_design_service";
import { BranchSchema } from "@/types/v1/schemaEditor";
import { nextAnimationFrame } from "@/utils";
import MaskSpinner from "../misc/MaskSpinner.vue";
import Aside from "./Aside/index.vue";
import Editor from "./Editor.vue";
import { convertBranchToBranchSchema } from "./utils/branch";

const props = defineProps<{
  project: ComposedProject;
  resourceType: "database" | "branch";
  readonly?: boolean;
  databases?: ComposedDatabase[];
  // NOTE: we only support editing one branch for now.
  branches?: SchemaDesign[];
  loading?: boolean;
}>();

interface LocalState {
  loading: boolean;
  initialized: boolean;
}

const settingStore = useSettingV1Store();
const schemaEditorV1Store = useSchemaEditorV1Store();
const schemaDesignStore = useSchemaDesignStore();
const state = reactive<LocalState>({
  loading: false,
  initialized: false,
});

const mergedLoading = computed(() => {
  return props.loading || state.loading;
});

const prepareBranchContext = async () => {
  if (props.resourceType !== "branch" || !props.branches) {
    return;
  }
  for (const branch of props.branches) {
    if (
      branch.type === SchemaDesign_Type.PERSONAL_DRAFT &&
      branch.parentBranch
    ) {
      // Prepare parent branch for personal draft.
      await schemaDesignStore.fetchSchemaDesignByName(
        branch.parentBranch,
        true /* useCache */
      );
    }
  }
};

const batchConvertBranchSchemas = async (branches: SchemaDesign[]) => {
  try {
    state.loading = true;
    await nextAnimationFrame();
    return await Promise.all(
      branches.map<Promise<[string, BranchSchema]>>(async (branch) => {
        return [branch.name, await convertBranchToBranchSchema(branch)];
      })
    );
  } finally {
    state.loading = false;
  }
};

const initialSchemaEditorState = async () => {
  schemaEditorV1Store.setState({
    project: props.project,
    resourceType: props.resourceType,
    // NOTE: this will clear all tabs. We will restore tabs as needed later.
    tabState: {
      tabMap: new Map(),
    },
    readonly: props.readonly || false,
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
        branch: new Map(await batchConvertBranchSchemas(props.branches ?? [])),
      },
    });
  }
};

// Prepare schema template contexts.
onMounted(async () => {
  await settingStore.getOrFetchSettingByName("bb.workspace.schema-template");
  await prepareBranchContext();
  await initialSchemaEditorState();
  state.initialized = true;
});

watch(
  () => props,
  () => {
    schemaEditorV1Store.setState({
      project: props.project,
      resourceType: props.resourceType,
      readonly: props.readonly || false,
    });
  },
  {
    deep: true,
  }
);

watch(
  [() => props.databases, () => props.branches],
  async ([newDatabases, newBranches], [oldDatabases, oldBranches]) => {
    // Update editor state if needed.
    // * If we update databases/branches, we need to rebuild the editing state.
    // * If we update databases/branches, we need to clear all tabs.
    if (props.resourceType === "database") {
      if (isEqual(newDatabases, oldDatabases)) {
        return;
      }
      schemaEditorV1Store.setState({
        tabState: {
          tabMap: new Map(),
        },
        resourceMap: {
          // NOTE: we will dynamically fetch schema list for each database in database tree view.
          database: new Map(
            (newDatabases || []).map((database) => [
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
      if (isEqual(newBranches, oldBranches)) {
        return;
      }
      schemaEditorV1Store.setState({
        tabState: {
          tabMap: new Map(),
        },
        resourceMap: {
          database: new Map(),
          branch: new Map(await batchConvertBranchSchemas(newBranches ?? [])),
        },
      });
    }
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
