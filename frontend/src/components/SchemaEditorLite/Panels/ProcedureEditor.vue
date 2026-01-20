<template>
  <CommonCodeEditor
    :db="db"
    :code="procedure.definition"
    :readonly="disallowChangeProcedure"
    :status="status"
    @update:code="handleUpdateDefinition"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import type {
  Database,
  DatabaseMetadata,
  ProcedureMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";
import type { EditStatus } from "../types";
import CommonCodeEditor from "./CommonCodeEditor.vue";

const props = defineProps<{
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  procedure: ProcedureMetadata;
}>();

const { readonly, markEditStatus, getSchemaStatus, getProcedureStatus } =
  useSchemaEditorContext();

const statusForSchema = () => {
  return getSchemaStatus(props.db, {
    schema: props.schema,
  });
};
const status = computed(() => {
  return getProcedureStatus(props.db, {
    schema: props.schema,
    procedure: props.procedure,
  });
});
const markStatus = (status: EditStatus) => {
  markEditStatus(
    props.db,
    {
      schema: props.schema,
      procedure: props.procedure,
    },
    status
  );
};

const disallowChangeProcedure = computed(() => {
  if (readonly.value) {
    return true;
  }
  return statusForSchema() === "dropped" || status.value === "dropped";
});

const handleUpdateDefinition = (code: string) => {
  // eslint-disable-next-line vue/no-mutating-props
  props.procedure.definition = code;

  if (status.value === "normal") {
    markStatus("updated");
  }
};
</script>
