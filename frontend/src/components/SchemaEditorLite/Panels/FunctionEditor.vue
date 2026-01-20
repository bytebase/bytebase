<template>
  <CommonCodeEditor
    :db="db"
    :code="func.definition"
    :readonly="disallowChangeFunction"
    :status="status"
    @update:code="handleUpdateDefinition"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import type {
  Database,
  DatabaseMetadata,
  FunctionMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";
import type { EditStatus } from "../types";
import CommonCodeEditor from "./CommonCodeEditor.vue";

const props = defineProps<{
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  func: FunctionMetadata;
}>();

const { readonly, markEditStatus, getSchemaStatus, getFunctionStatus } =
  useSchemaEditorContext();

const statusForSchema = () => {
  return getSchemaStatus(props.db, {
    schema: props.schema,
  });
};
const status = computed(() => {
  return getFunctionStatus(props.db, {
    schema: props.schema,
    function: props.func,
  });
});
const markStatus = (status: EditStatus) => {
  markEditStatus(
    props.db,
    {
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

const handleUpdateDefinition = (code: string) => {
  // eslint-disable-next-line vue/no-mutating-props
  props.func.definition = code;

  if (status.value === "normal") {
    markStatus("updated");
  }
};
</script>
