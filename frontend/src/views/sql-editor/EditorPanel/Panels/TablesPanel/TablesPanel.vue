<template>
  <div
    v-if="metadata?.schema"
    class="px-2 py-2 gap-y-2 h-full overflow-hidden flex flex-col"
  >
    <template v-if="!metadata.table">
      <SchemaSelectToolbar />
      <TablesTable
        v-if="!metadata.table"
        :db="database"
        :database="metadata.database"
        :schema="metadata.schema"
        :tables="metadata.schema.tables"
        @click="select"
      />
    </template>

    <template v-if="metadata.table">
      <TableEditor
        v-if="metadata.table"
        :db="database"
        :database="metadata.database"
        :schema="metadata.schema"
        :table="metadata.table"
        class="!pt-0"
      >
        <template #toolbar-prefix>
          <NButton size="small" @click="deselect">
            <ArrowLeftIcon class="w-4 h-4" />
          </NButton>
        </template>
      </TableEditor>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ArrowLeftIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref, watch } from "vue";
import { provideSchemaEditorContext } from "@/components/SchemaEditorLite";
import TableEditor from "@/components/SchemaEditorLite/Panels/TableEditor.vue";
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
import { useEditorPanelContext } from "../../context";
import { SchemaSelectToolbar } from "../common";
import TablesTable from "./TablesTable.vue";

const editorStore = useSQLEditorStore();
const { database } = useConnectionOfCurrentSQLEditorTab();
const { selectedSchemaName } = useEditorPanelContext();
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

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
}) => {
  metadata.value = selected;
};

const deselect = () => {
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
    forceShowIndexes: true,
    forceShowPartitions: true,
  }),
});
</script>
