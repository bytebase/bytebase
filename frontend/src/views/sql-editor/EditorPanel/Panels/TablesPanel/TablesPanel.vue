<template>
  <div v-if="metadata" class="px-2 py-2 h-full overflow-hidden flex flex-col">
    <div
      v-if="showToolbar"
      class="pb-2 w-full flex flex-row justify-between items-center"
    >
      <template v-if="!metadata.table">
        <div
          v-if="showSchemaSelect"
          class="flex flex-row justify-start items-center text-sm gap-x-2"
        >
          <span>Schema:</span>
          <NSelect
            v-model:value="selectedSchemaName"
            :options="schemaSelectOptions"
            class="min-w-[8rem]"
          />
        </div>
      </template>
    </div>

    <template v-if="metadata.schema">
      <TableList
        v-if="!metadata.table"
        :db="database"
        :database="metadata.database"
        :schema="metadata.schema"
        :tables="metadata.schema.tables"
        :custom-click="true"
        @click="handleSelectTable"
      />
      <TableEditor
        v-if="metadata.table"
        :db="database"
        :database="metadata.database"
        :schema="metadata.schema"
        :table="metadata.table"
        class="!pt-0"
      >
        <template #toolbar-prefix>
          <NButton size="small" @click="deselectTable">
            <ArrowLeftIcon class="w-4 h-4" />
          </NButton>
        </template>
      </TableEditor>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ArrowLeftIcon } from "lucide-vue-next";
import { NButton, NSelect } from "naive-ui";
import { computed, ref, watch } from "vue";
import { provideSchemaEditorContext } from "@/components/SchemaEditorLite";
import TableEditor from "@/components/SchemaEditorLite/Panels/TableEditor.vue";
import TableList from "@/components/SchemaEditorLite/Panels/TableList";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
  useProjectV1Store,
  useSQLEditorStore,
} from "@/store";
import {
  DatabaseMetadata,
  DatabaseMetadataView,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { useSelectSchema } from "../../common";

const editorStore = useSQLEditorStore();
const {
  selectedSchemaName,
  options: schemaSelectOptions,
  showSchemaSelect,
} = useSelectSchema();
const { database } = useConnectionOfCurrentSQLEditorTab();
const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(
    database.value.name,
    DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
  );
});

const metadata = ref<{
  database: DatabaseMetadata;
  schema?: SchemaMetadata;
  table?: TableMetadata;
}>();

const showToolbar = computed(() => {
  if (metadata.value?.table) {
    // should show table detail
    return false;
  }

  if (showSchemaSelect.value) return true;
  return false;
});

const handleSelectTable = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
}) => {
  metadata.value = selected;
};

const deselectTable = () => {
  if (!metadata.value) return;
  metadata.value.table = undefined;
};

watch(
  [databaseMetadata, selectedSchemaName],
  ([database, schema]) => {
    metadata.value = {
      database,
      schema: database.schemas.find((s) => s.name === schema),
      table: undefined,
    };
  },
  { immediate: true }
);

provideSchemaEditorContext({
  targets: computed(() => [
    {
      database: database.value,
      metadata: databaseMetadata.value,
      baselineMetadata: databaseMetadata.value,
    },
  ]),
  project: computed(() =>
    useProjectV1Store().getProjectByName(editorStore.project)
  ),
  resourceType: ref("branch"),
  readonly: ref(true),
  selectedRolloutObjects: ref(undefined),
  showLastUpdater: ref(false),
  disableDiffColoring: ref(true),
  options: ref({
    hideSemanticTypeColumn: true,
    hideClassificationColumn: true,
  }),
});
</script>
