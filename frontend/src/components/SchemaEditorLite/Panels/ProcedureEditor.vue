<template>
  <CommonCodeEditor
    :db="db"
    :code="procedure.definition"
    :readonly="disallowChangeProcedure"
    :status="status"
    @update:code="handleUpdateDefinition"
  >
    <template v-if="showLastUpdater && procedureConfig" #header-suffix>
      <div class="flex justify-end items-center h-[28px]">
        <LastUpdater
          :updater="procedureConfig.updater"
          :update-time="procedureConfig.updateTime"
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
  ProcedureMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { useSchemaEditorContext } from "../context";
import type { EditStatus } from "../types";
import CommonCodeEditor from "./CommonCodeEditor.vue";
import { LastUpdater } from "./common";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  procedure: ProcedureMetadata;
}>();

const {
  readonly,
  showLastUpdater,
  markEditStatus,
  getSchemaStatus,
  getProcedureStatus,
} = useSchemaEditorContext();

const statusForSchema = () => {
  return getSchemaStatus(props.db, {
    database: props.database,
    schema: props.schema,
  });
};
const status = computed(() => {
  return getProcedureStatus(props.db, {
    database: props.database,
    schema: props.schema,
    procedure: props.procedure,
  });
});
const markStatus = (status: EditStatus) => {
  markEditStatus(
    props.db,
    {
      database: props.database,
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

const procedureConfig = computed(() => {
  const sc = props.database.schemaConfigs.find(
    (sc) => sc.name === props.schema.name
  );
  if (!sc) return undefined;
  return sc.procedureConfigs.find((pc) => pc.name === props.procedure.name);
});

const handleUpdateDefinition = (code: string) => {
  // eslint-disable-next-line vue/no-mutating-props
  props.procedure.definition = code;

  if (status.value === "normal") {
    markStatus("updated");
  }
};
</script>
