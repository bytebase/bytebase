<template>
  <div class="w-full h-[32rem] border rounded-lg">
    <Splitpanes
      class="default-theme w-full h-full flex flex-row overflow-hidden"
    >
      <Pane min-size="15" size="25">
        <AsidePanel />
      </Pane>
      <Pane min-size="60" size="75">
        <Designer />
      </Pane>
    </Splitpanes>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { Splitpanes, Pane } from "splitpanes";
import { onMounted, ref, watch } from "vue";
import { useSettingV1Store } from "@/store";
import { Schema, convertSchemaMetadataList } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import AsidePanel from "./AsidePanel.vue";
import Designer from "./Designer.vue";
import { provideSchemaDesignerContext } from "./common";
import { SchemaDesignerTabState } from "./common/type";
import { rebuildEditableSchemas } from "./common/util";

const props = defineProps<{
  readonly: boolean;
  engine: Engine;
  schemaDesign: SchemaDesign;
}>();

const settingStore = useSettingV1Store();
const readonly = ref(props.readonly);
const engine = ref(props.engine);
const metadata = ref<DatabaseMetadata>(DatabaseMetadata.fromPartial({}));
const originalSchemas = ref<Schema[]>([]);
const editableSchemas = ref<Schema[]>([]);
const baselineMetadata = ref<DatabaseMetadata>(
  DatabaseMetadata.fromPartial({})
);
const tabState = ref<SchemaDesignerTabState>({
  tabMap: new Map(),
});

// Prepare schema template contexts.
onMounted(async () => {
  await settingStore.getOrFetchSettingByName("bb.workspace.schema-template");
});

const rebuildEditingState = () => {
  originalSchemas.value = convertSchemaMetadataList(
    baselineMetadata.value.schemas
  );
  editableSchemas.value = rebuildEditableSchemas(
    originalSchemas.value,
    metadata.value.schemas
  );
  tabState.value = {
    tabMap: new Map(),
  };
};

provideSchemaDesignerContext({
  readonly: readonly,
  baselineMetadata: baselineMetadata,
  engine: engine,
  metadata: metadata,
  tabState: tabState,
  originalSchemas: originalSchemas,
  editableSchemas: editableSchemas,
});

watch(
  () => props,
  () => {
    baselineMetadata.value =
      cloneDeep(props.schemaDesign?.baselineSchemaMetadata) ||
      DatabaseMetadata.fromPartial({});
    metadata.value =
      cloneDeep(props.schemaDesign?.schemaMetadata) ||
      DatabaseMetadata.fromPartial({});
    readonly.value = props.readonly;
    engine.value = props.engine;
  },
  {
    immediate: true,
    deep: true,
  }
);

watch(
  () => metadata.value,
  (value, oldValue) => {
    // NOTE: regenerate editing state in the following cases:
    // * change baseline schema.
    // * change selected schema design.
    if (!isEqual(value, oldValue)) {
      rebuildEditingState();
    }
  },
  {
    immediate: true,
    deep: true,
  }
);

defineExpose({
  metadata,
  baselineMetadata,
  editableSchemas,
  rebuildEditingState,
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
