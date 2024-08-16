<template>
  <div class="px-2 py-2 gap-4 h-full overflow-hidden flex flex-col">
    <div class="grid gap-2 items-center text-sm" style="grid-template-columns: max-content max-content">
      <div class="flex items-center gap-x-1">
        <DatabaseIcon class="w-4 h-4" />
        <span>{{ $t("common.database") }}:</span>
      </div>
      <DatabaseChooser style="justify-content: start" />
      <template v-if="showSchemaSelect">
        <div class="flex items-center gap-x-1">
          <SchemaIcon class="w-4 h-4" />
          <span>{{ $t("common.schema") }}:</span>
        </div>
        <NSelect
          v-model:value="selectedSchemaName"
          :options="schemaSelectOptions"
          size="small"
          class="min-w-[12rem]"
        />
      </template>
    </div>

    <div class="flex-1 overflow-auto flex flex-col gap-4">
      <DatabaseOverviewInfo :database="database" />

      <div class="flex flex-col gap-2">
        <h2 class="text-lg">{{ $t("db.tables") }}</h2>
        <div class="max-h-[20rem] overflow-x-auto overflow-y-hidden">
          <TablesTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :tables="metadata.schema.tables"
            @click="
              updateViewState({
                view: 'TABLES',
                detail: {
                  table: $event.table.name,
                },
              })
            "
          />
        </div>
      </div>

      <div class="flex flex-col gap-2">
        <h2 class="text-lg">{{ $t("db.views") }}</h2>
        <div class="max-h-[20rem] overflow-x-auto overflow-y-hidden">
          <ViewsTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :views="metadata.schema.views"
            @click="
              updateViewState({
                view: 'VIEWS',
                detail: {
                  view: $event.view.name,
                },
              })
            "
          />
        </div>
      </div>

      <div class="flex flex-col gap-2">
        <h2 class="text-lg">{{ $t("db.functions") }}</h2>
        <div class="max-h-[20rem] overflow-x-auto overflow-y-hidden">
          <FunctionsTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :funcs="metadata.schema.functions"
            @click="
              updateViewState({
                view: 'FUNCTIONS',
                detail: {
                  func: $event.func.name,
                },
              })
            "
          />
        </div>
      </div>

      <div class="flex flex-col gap-2">
        <h2 class="text-lg">{{ $t("db.procedures") }}</h2>
        <div class="max-h-[20rem] overflow-x-auto overflow-y-hidden">
          <ProceduresTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :procedures="metadata.schema.procedures"
            @click="
              updateViewState({
                view: 'PROCEDURES',
                detail: {
                  procedure: $event.procedure.name,
                },
              })
            "
          />
        </div>
      </div>

      <div
        v-if="instanceV1SupportsExternalTable(database.instanceResource)"
        class="flex flex-col gap-2"
      >
        <h2 class="text-lg">{{ $t("db.external-tables") }}</h2>
        <div class="max-h-[20rem] overflow-x-auto overflow-y-hidden">
          <ExternalTablesTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :external-tables="metadata.schema.externalTables"
            @click="
              updateViewState({
                view: 'EXTERNAL_TABLES',
                detail: {
                  externalTable: $event.externalTable.name,
                },
              })
            "
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NSelect, type SelectOption } from "naive-ui";
import { computed } from "vue";
import DatabaseOverviewInfo from "@/components/Database/DatabaseOverviewInfo.vue";
import { DatabaseIcon, SchemaIcon } from "@/components/Icon";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { hasSchemaProperty, instanceV1SupportsExternalTable } from "@/utils";
import DatabaseChooser from "@/views/sql-editor/EditorCommon/DatabaseChooser.vue";
import { useEditorPanelContext } from "../../context";
import ExternalTablesTable from "../ExternalTablesPanel/ExternalTablesTable.vue";
import FunctionsTable from "../FunctionsPanel/FunctionsTable.vue";
import ProceduresTable from "../ProceduresPanel/ProceduresTable.vue";
import TablesTable from "../TablesPanel/TablesTable.vue";
import ViewsTable from "../ViewsPanel/ViewsTable.vue";
import { SchemaSelectToolbar } from "../common";

const { instance, database } = useConnectionOfCurrentSQLEditorTab();
const { viewState, updateViewState, selectedSchemaName } =
  useEditorPanelContext();
const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(
    database.value.name,
    DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
  );
});

const metadata = computed(() => {
  const database = databaseMetadata.value;
  const schema = database.schemas.find(
    (s) => s.name === viewState.value?.schema
  );
  const table = schema?.tables.find(
    (t) => t.name === viewState.value?.detail?.table
  );
  return { database, schema, table };
});

const schemaSelectOptions = computed(() => {
  return databaseMetadata.value.schemas.map<SelectOption>((schema) => ({
    label: schema.name,
    value: schema.name,
  }));
});
const showSchemaSelect = computed(() => {
  return hasSchemaProperty(instance.value.engine);
});
</script>
