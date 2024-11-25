<template>
  <div class="px-2 py-2 gap-4 h-full overflow-hidden flex flex-col">
    <div class="w-full flex flex-row gap-x-2 justify-between items-center">
      <div class="flex items-center justify-start gap-2">
        <DatabaseChooser />
        <SchemaSelectToolbar simple />
      </div>
    </div>

    <div class="flex-1 overflow-auto flex flex-col gap-4">
      <DatabaseOverviewInfo :database="database" />

      <div class="flex flex-col gap-2">
        <div class="flex items-center justify-between">
          <h2 class="text-lg">{{ $t("db.tables") }}</h2>
          <SearchBox v-model:value="state.keywords.tables" />
        </div>
        <div class="overflow-x-auto overflow-y-hidden">
          <TablesTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :tables="metadata.schema.tables"
            :keyword="state.keywords.tables"
            :max-height="230"
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
        <div class="flex items-center justify-between">
          <h2 class="text-lg">{{ $t("db.views") }}</h2>
          <SearchBox v-model:value="state.keywords.views" />
        </div>
        <div class="overflow-x-auto overflow-y-hidden">
          <ViewsTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :views="metadata.schema.views"
            :keyword="state.keywords.views"
            :max-height="230"
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
        <div class="flex items-center justify-between">
          <h2 class="text-lg">{{ $t("db.functions") }}</h2>
          <SearchBox v-model:value="state.keywords.functions" />
        </div>
        <div class="max-h-[16rem] overflow-x-auto overflow-y-hidden">
          <FunctionsTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :funcs="metadata.schema.functions"
            :keyword="state.keywords.functions"
            :max-height="230"
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
        <div class="flex items-center justify-between">
          <h2 class="text-lg">{{ $t("db.procedures") }}</h2>
          <SearchBox v-model:value="state.keywords.procedures" />
        </div>
        <div class="max-h-[16rem] overflow-x-auto overflow-y-hidden">
          <ProceduresTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :procedures="metadata.schema.procedures"
            :keyword="state.keywords.procedures"
            :max-height="230"
            @click="
              updateViewState({
                view: 'PROCEDURES',
                detail: {
                  procedure: keyWithPosition(
                    $event.procedure.name,
                    $event.position
                  ),
                },
              })
            "
          />
        </div>
      </div>

      <div
        v-if="instanceV1SupportsSequence(database.instanceResource)"
        class="flex flex-col gap-2"
      >
        <div class="flex items-center justify-between">
          <h2 class="text-lg">{{ $t("db.sequences") }}</h2>
          <SearchBox v-model:value="state.keywords.sequences" />
        </div>
        <div class="max-h-[16rem] overflow-x-auto overflow-y-hidden">
          <SequencesTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :sequences="metadata.schema.sequences"
            :keyword="state.keywords.sequences"
            :max-height="230"
            @click="
              updateViewState({
                view: 'SEQUENCES',
                detail: {
                  sequence: keyWithPosition(
                    $event.sequence.name,
                    $event.position
                  ),
                },
              })
            "
          />
        </div>
      </div>

      <div
        v-if="instanceV1SupportsTrigger(database.instanceResource)"
        class="flex flex-col gap-2"
      >
        <div class="flex items-center justify-between">
          <h2 class="text-lg">{{ $t("db.triggers") }}</h2>
          <SearchBox v-model:value="state.keywords.triggers" />
        </div>
        <div class="max-h-[16rem] overflow-x-auto overflow-y-hidden">
          <TriggersTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :triggers="metadata.schema.triggers"
            :keyword="state.keywords.triggers"
            :max-height="230"
            @click="
              updateViewState({
                view: 'TRIGGERS',
                detail: {
                  trigger: keyWithPosition(
                    $event.trigger.name,
                    $event.position
                  ),
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
        <div class="flex items-center justify-between">
          <h2 class="text-lg">{{ $t("db.external-tables") }}</h2>
          <SearchBox v-model:value="state.keywords.externalTables" />
        </div>
        <div class="max-h-[16rem] overflow-x-auto overflow-y-hidden">
          <ExternalTablesTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :external-tables="metadata.schema.externalTables"
            :keyword="state.keywords.externalTables"
            :max-height="230"
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

      <div
        v-if="instanceV1SupportsPackage(database.instanceResource)"
        class="flex flex-col gap-2"
      >
        <div class="flex items-center justify-between">
          <h2 class="text-lg">{{ $t("db.packages") }}</h2>
          <SearchBox v-model:value="state.keywords.packages" />
        </div>
        <div class="max-h-[16rem] overflow-x-auto overflow-y-hidden">
          <PackagesTable
            v-if="metadata.schema"
            :db="database"
            :database="metadata.database"
            :schema="metadata.schema"
            :packages="metadata.schema.packages"
            :keyword="state.keywords.packages"
            :max-height="230"
            @click="
              updateViewState({
                view: 'PACKAGES',
                detail: {
                  package: keyWithPosition(
                    $event.package.name,
                    $event.position
                  ),
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
import { computed, reactive } from "vue";
import DatabaseOverviewInfo from "@/components/Database/DatabaseOverviewInfo.vue";
import { SearchBox } from "@/components/v2";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import {
  instanceV1SupportsExternalTable,
  instanceV1SupportsPackage,
  instanceV1SupportsSequence,
  instanceV1SupportsTrigger,
} from "@/utils";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon";
import DatabaseChooser from "@/views/sql-editor/EditorCommon/DatabaseChooser.vue";
import { useEditorPanelContext } from "../../context";
import ExternalTablesTable from "../ExternalTablesPanel/ExternalTablesTable.vue";
import FunctionsTable from "../FunctionsPanel/FunctionsTable.vue";
import PackagesTable from "../PackagesPanel/PackagesTable.vue";
import ProceduresTable from "../ProceduresPanel/ProceduresTable.vue";
import SequencesTable from "../SequencesPanel/SequencesTable.vue";
import TablesTable from "../TablesPanel/TablesTable.vue";
import TriggersTable from "../TriggersPanel/TriggersTable.vue";
import ViewsTable from "../ViewsPanel/ViewsTable.vue";
import { SchemaSelectToolbar } from "../common";

const state = reactive({
  keywords: {
    tables: "",
    views: "",
    functions: "",
    procedures: "",
    sequences: "",
    triggers: "",
    externalTables: "",
    packages: "",
  },
});

const { database } = useConnectionOfCurrentSQLEditorTab();
const { viewState, updateViewState } = useEditorPanelContext();
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
</script>
