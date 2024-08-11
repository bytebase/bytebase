<template>
  <CommonCodeEditor
    :db="db"
    :code="view.definition"
    :readonly="disallowChangeView"
    :status="status"
    @update:code="handleUpdateDefinition"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  ViewMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { useSchemaEditorContext } from "../context";
import type { EditStatus } from "../types";
import CommonCodeEditor from "./CommonCodeEditor.vue";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  view: ViewMetadata;
}>();

const { readonly, markEditStatus, getSchemaStatus, getViewStatus } =
  useSchemaEditorContext();

const statusForSchema = () => {
  return getSchemaStatus(props.db, {
    database: props.database,
    schema: props.schema,
  });
};
const status = computed(() => {
  return getViewStatus(props.db, {
    database: props.database,
    schema: props.schema,
    view: props.view,
  });
});
const markStatus = (status: EditStatus) => {
  markEditStatus(
    props.db,
    {
      database: props.database,
      schema: props.schema,
      view: props.view,
    },
    status
  );
};

const disallowChangeView = computed(() => {
  if (readonly.value) {
    return true;
  }
  return statusForSchema() === "dropped" || status.value === "dropped";
});

const handleUpdateDefinition = (code: string) => {
  // eslint-disable-next-line vue/no-mutating-props
  props.view.definition = code;

  if (status.value === "normal") {
    markStatus("updated");
  }
};
</script>
