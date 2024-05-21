<template>
  <CommonCodeEditor
    :db="db"
    :code="func.definition"
    :readonly="disallowChangeFunction"
    :status="status"
    @update:code="handleUpdateDefinition"
  >
    <template v-if="showLastUpdater && functionConfig" #header-suffix>
      <div class="flex justify-end items-center h-[28px]">
        <LastUpdater
          :updater="functionConfig.updater"
          :update-time="functionConfig.updateTime"
        />
      </div>
    </template>
  </CommonCodeEditor>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  FunctionMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { useSchemaEditorContext } from "../context";
import type { EditStatus } from "../types";
import CommonCodeEditor from "./CommonCodeEditor.vue";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  func: FunctionMetadata;
}>();

const {
  readonly,
  showLastUpdater,
  markEditStatus,
  getSchemaStatus,
  getFunctionStatus,
} = useSchemaEditorContext();

const statusForSchema = () => {
  return getSchemaStatus(props.db, {
    database: props.database,
    schema: props.schema,
  });
};
const status = computed(() => {
  return getFunctionStatus(props.db, {
    database: props.database,
    schema: props.schema,
    function: props.func,
  });
});
const markStatus = (status: EditStatus) => {
  markEditStatus(
    props.db,
    {
      database: props.database,
      schema: props.schema,
      function: props.func,
    },
    status
  );
};

const disallowChangeFunction = computed(() => {
  if (readonly.value) {
    return true;
  }
  return statusForSchema() === "dropped" || status.value === "dropped";
});

const functionConfig = computed(() => {
  const sc = props.database.schemaConfigs.find(
    (sc) => sc.name === props.schema.name
  );
  if (!sc) return undefined;
  return sc.functionConfigs.find((pc) => pc.name === props.func.name);
});

const handleUpdateDefinition = (code: string) => {
  // eslint-disable-next-line vue/no-mutating-props
  props.func.definition = code;

  if (status.value === "normal") {
    markStatus("updated");
  }
};
</script>
