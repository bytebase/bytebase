<template>
  <div v-if="!state.isLoading" class="w-full h-[32rem] border rounded-lg">
    <Splitpanes
      class="default-theme w-full h-full flex flex-row overflow-hidden"
    >
      <Pane size="25">
        <AsidePanel />
      </Pane>
      <Pane min-size="60" size="75">
        <Designer />
      </Pane>
    </Splitpanes>
  </div>
</template>

<script lang="ts" setup>
import { Splitpanes, Pane } from "splitpanes";
import { onMounted, reactive, ref } from "vue";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { Engine } from "@/types/proto/v1/common";
import { provideSchemaDesignerContext } from "./common";
import { SchemaDesignerTabState } from "./common/type";
import AsidePanel from "./AsidePanel.vue";
import Designer from "./Designer.vue";
import { Schema, convertSchemaMetadataList } from "@/types";
import { cloneDeep } from "lodash-es";

interface LocalState {
  isLoading: boolean;
}

const props = defineProps<{
  readonly: boolean;
  engine: Engine;
  schemaDesign: SchemaDesign;
}>();

const state = reactive<LocalState>({
  isLoading: true,
});

const metadata = ref<DatabaseMetadata>(DatabaseMetadata.fromPartial({}));
const editableSchemas = ref<Schema[]>([]);
const baselineMetadata = ref<DatabaseMetadata>(
  DatabaseMetadata.fromPartial({})
);
const tabState = ref<SchemaDesignerTabState>({
  tabMap: new Map(),
});

onMounted(async () => {
  baselineMetadata.value =
    cloneDeep(props.schemaDesign?.baselineSchemaMetadata) ||
    DatabaseMetadata.fromPartial({});
  metadata.value =
    cloneDeep(props.schemaDesign?.schemaMetadata) ||
    DatabaseMetadata.fromPartial({});
  editableSchemas.value = convertSchemaMetadataList(metadata.value.schemas);
  state.isLoading = false;
});

provideSchemaDesignerContext({
  readonly: props.readonly,
  baselineMetadata: baselineMetadata.value,
  engine: props.engine,
  metadata: metadata,
  tabState: tabState,
  originalSchemas: cloneDeep(editableSchemas.value),
  editableSchemas: editableSchemas,
});

defineExpose({
  metadata,
  baselineMetadata,
  editableSchemas,
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
