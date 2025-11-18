<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent
      :title="$t('database.table-detail')"
      :content-props="{
        bodyContentClass: 'relative',
      }"
    >
      <div class="focus:outline-hidden w-[calc(100vw-256px)]" tabindex="0">
        <MaskSpinner v-if="isFetchingTableMetadata" />
        <main v-if="table" class="flex-1 relative pb-8 overflow-y-auto">
          <!-- Highlight Panel -->
          <div
            class="px-4 pb-4 border-b border-block-border md:flex md:items-center md:justify-between"
          >
            <div class="flex-1 min-w-0">
              <!-- Summary -->
              <div class="flex items-center">
                <div>
                  <div class="flex items-center">
                    <h1
                      class="pt-2 pb-2.5 text-xl font-bold leading-6 text-main truncate flex items-center gap-x-3"
                    >
                      {{ getTableName(table.name) }}
                    </h1>
                  </div>
                </div>
              </div>
              <dl
                class="flex flex-col gap-y-1 md:gap-y-0 md:flex-row md:flex-wrap"
              >
                <dt class="sr-only">{{ $t("common.environment") }}</dt>
                <dd class="flex items-center text-sm md:mr-4">
                  <span class="textlabel"
                    >{{ $t("common.environment") }}&nbsp;-&nbsp;</span
                  >
                  <EnvironmentV1Name
                    :environment="database.effectiveEnvironmentEntity"
                    icon-class="textinfolabel"
                  />
                </dd>
                <dt class="sr-only">{{ $t("common.instance") }}</dt>
                <dd class="flex items-center text-sm md:mr-4">
                  <span class="ml-1 textlabel"
                    >{{ $t("common.instance") }}&nbsp;-&nbsp;</span
                  >
                  <InstanceV1Name :instance="database.instanceResource" />
                </dd>
                <dt class="sr-only">{{ $t("common.project") }}</dt>
                <dd class="flex items-center text-sm md:mr-4">
                  <span class="textlabel"
                    >{{ $t("common.project") }}&nbsp;-&nbsp;</span
                  >
                  <ProjectV1Name
                    :project="database.projectEntity"
                    hash="#databases"
                  />
                </dd>
                <dt class="sr-only">{{ $t("common.database") }}</dt>
                <dd class="flex items-center text-sm md:mr-4">
                  <span class="textlabel"
                    >{{ $t("common.database") }}&nbsp;-&nbsp;</span
                  >
                  <DatabaseV1Name :database="database" />
                </dd>
                <SQLEditorButtonV1
                  v-if="allowQuery"
                  class="text-sm md:mr-4"
                  :database="database"
                  :schema="schemaName"
                  :table="tableName"
                  :label="true"
                />
                <NPopover
                  trigger="click"
                  placement="bottom"
                  v-if="supportGetStringSchema(instanceEngineNew)"
                  @update:show="(show: boolean) => show"
                >
                  <template #trigger>
                    <NButton
                      quaternary
                      size="tiny"
                      class="px-1!"
                      v-bind="$attrs"
                    >
                      <span class="textlabel"
                        >{{ $t("database.view-definition") }}
                      </span>
                      <CodeIcon class="ml-1 w-4 h-4" />
                    </NButton>
                  </template>
                  <TableSchemaViewer
                    class="w-lg! h-80!"
                    :database="database"
                    :schema="schemaName"
                    :object="tableName"
                    :type="GetSchemaStringRequest_ObjectType.TABLE"
                  />
                </NPopover>
              </dl>
            </div>
          </div>

          <div class="mt-6">
            <div class="max-w-6xl px-6 flex flex-col gap-y-6">
              <!-- Description list -->
              <dl class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-3">
                <div
                  v-if="hasTableEngineProperty(instanceEngineNew)"
                  class="col-span-1"
                >
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.engine") }}
                  </dt>
                  <dd class="mt-1 text-semibold">
                    {{ table.engine }}
                  </dd>
                </div>

                <div class="col-span-1">
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.classification.self") }}
                  </dt>
                  <dd class="mt-1 flex flex-row items-center">
                    <ClassificationCell
                      :classification="tableCatalog.classification"
                      :classification-config="classificationConfig"
                      :engine="instanceEngineNew"
                      @apply="
                        (id: string) =>
                          $emit('apply-classification', tableName, id)
                      "
                    />
                  </dd>
                </div>

                <div class="col-span-1">
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.row-count-estimate") }}
                  </dt>
                  <dd class="mt-1 text-semibold">
                    {{ table.rowCount }}
                  </dd>
                </div>

                <div class="col-span-1">
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.data-size") }}
                  </dt>
                  <dd class="mt-1 text-semibold">
                    {{ bytesToString(Number(table.dataSize)) }}
                  </dd>
                </div>

                <div
                  v-if="hasIndexSizeProperty(instanceEngineNew)"
                  class="col-span-1"
                >
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.index-size") }}
                  </dt>
                  <dd class="mt-1 text-semibold">
                    {{ bytesToString(Number(table.indexSize)) }}
                  </dd>
                </div>

                <template
                  v-if="
                    instanceV1HasCollationAndCharacterSet(instanceEngineNew)
                  "
                >
                  <div class="col-span-1">
                    <dt class="text-sm font-medium text-control-light">
                      {{ $t("db.collation") }}
                    </dt>
                    <dd class="mt-1 text-semibold">
                      {{ table.collation }}
                    </dd>
                  </div>
                </template>
              </dl>
            </div>
          </div>

          <div v-if="shouldShowPartitionTablesDataTable" class="mt-6 px-6">
            <div class="mb-4 w-full flex flex-row justify-between items-center">
              <div class="text-lg leading-6 font-medium text-main">
                {{ $t("database.partition-tables") }}
              </div>
              <div>
                <SearchBox
                  :value="state.partitionTableNameSearchKeyword"
                  :placeholder="$t('common.filter-by-name')"
                  @update:value="state.partitionTableNameSearchKeyword = $event"
                />
              </div>
            </div>
            <PartitionTablesDataTable
              :table="table"
              :search="state.partitionTableNameSearchKeyword"
            />
          </div>

          <div
            v-if="instanceV1SupportsColumn(instanceEngineNew)"
            class="mt-6 px-6 flex flex-col gap-y-4"
          >
            <div class="w-full flex flex-row justify-between items-center">
              <div class="text-lg leading-6 font-medium text-main">
                {{ $t("database.columns") }}
              </div>
              <div>
                <SearchBox
                  :value="state.columnNameSearchKeyword"
                  :placeholder="$t('common.filter-by-name')"
                  @update:value="state.columnNameSearchKeyword = $event"
                />
              </div>
            </div>

            <ColumnDataTable
              :database="database"
              :schema="schemaName"
              :table="table"
              :is-external-table="false"
              :column-list="table.columns"
              :classification-config="classificationConfig"
              :search="state.columnNameSearchKeyword"
            />
          </div>

          <div
            v-if="instanceV1SupportsIndex(instanceEngineNew)"
            class="mt-6 px-6 flex flex-col gap-y-4"
          >
            <div class="text-lg leading-6 font-medium text-main">
              {{ $t("database.indexes") }}
            </div>
            <IndexTable :database="database" :index-list="table.indexes" />
          </div>

          <div
            v-if="instanceV1SupportsTrigger(instanceEngineNew)"
            class="mt-6 px-6 flex flex-col gap-y-4"
          >
            <div class="text-lg leading-6 font-medium text-main">
              {{ $t("db.triggers") }}
            </div>
            <TriggerDataTable
              :database="database"
              :schema-name="schemaName"
              :table-name="tableName"
              :trigger-list="table.triggers"
            />
          </div>

          <div
            v-if="instanceV1MaskingForNoSQL(instanceEngineNew)"
            class="mt-6 px-6 flex flex-col gap-y-4"
          >
            <div>
              <div class="text-lg leading-6 font-medium text-main">
                {{ $t("common.catalog") }}
              </div>
              <span class="text-sm text-gray-400 -translate-y-2">
                {{ $t("db.catalog.description") }}
                <a
                  href="https://api.bytebase.com/#tag/databasecatalogservice/PATCH/v1/instances/{instance}/databases/{database}/catalog"
                  target="__blank"
                  class="normal-link"
                >
                  {{ $t("common.view-doc") }}
                </a>
              </span>
            </div>
            <NInput
              type="textarea"
              :autosize="{
                minRows: 15,
                maxRows: 50,
              }"
              :disabled="!hasDatabaseCatalogPermission"
              v-model:value="state.tableCatalog"
            />
            <div
              v-if="state.tableCatalog !== initTableCatalog"
              class="w-full flex items-center justify-end gap-x-2 mt-2"
            >
              <NButton
                type="primary"
                :loading="state.isUploading"
                @click="onCatalogUpload"
              >
                {{ $t("common.upload") }}
              </NButton>
            </div>
          </div>
        </main>
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create, fromJsonString, toJsonString } from "@bufbuild/protobuf";
import { computedAsync } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import { CodeIcon } from "lucide-vue-next";
import { NButton, NInput, NPopover } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import ClassificationCell from "@/components/ColumnDataTable/ClassificationCell.vue";
import TableSchemaViewer from "@/components/TableSchemaViewer.vue";
import {
  DatabaseV1Name,
  Drawer,
  DrawerContent,
  EnvironmentV1Name,
  InstanceV1Name,
  ProjectV1Name,
  SearchBox,
} from "@/components/v2";
import {
  getTableCatalog,
  pushNotification,
  useDatabaseCatalog,
  useDatabaseCatalogV1Store,
  useDatabaseV1Store,
  useDBSchemaV1Store,
} from "@/store";
import { DEFAULT_PROJECT_NAME, defaultProject } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  SchemaCatalogSchema,
  TableCatalogSchema,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import { GetSchemaStringRequest_ObjectType } from "@/types/proto-es/v1/database_service_pb";
import type { DataClassificationSetting_DataClassificationConfig } from "@/types/proto-es/v1/setting_service_pb";
import {
  bytesToString,
  hasIndexSizeProperty,
  hasProjectPermissionV2,
  hasSchemaProperty,
  hasTableEngineProperty,
  instanceV1HasCollationAndCharacterSet,
  instanceV1MaskingForNoSQL,
  instanceV1SupportsColumn,
  instanceV1SupportsIndex,
  instanceV1SupportsTrigger,
  isDatabaseV1Queryable,
  supportGetStringSchema,
} from "@/utils";
import ColumnDataTable from "./ColumnDataTable/index.vue";
import { SQLEditorButtonV1 } from "./DatabaseDetail";
import IndexTable from "./IndexTable.vue";
import MaskSpinner from "./misc/MaskSpinner.vue";
import PartitionTablesDataTable from "./PartitionTablesDataTable.vue";
import TriggerDataTable from "./TriggerDataTable.vue";

interface LocalState {
  columnNameSearchKeyword: string;
  partitionTableNameSearchKeyword: string;
  tableCatalog: string;
  isUploading: boolean;
}

const props = defineProps<{
  show: boolean;
  // Format: /databases/:databaseName
  databaseName: string;
  schemaName: string;
  tableName: string;
  classificationConfig?: DataClassificationSetting_DataClassificationConfig;
}>();

defineEmits<{
  (event: "dismiss"): void;
  (event: "apply-classification", table: string, id: string): void;
}>();

const { t } = useI18n();
const databaseV1Store = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const catalogStore = useDatabaseCatalogV1Store();

const databaseCatalog = useDatabaseCatalog(props.databaseName, false);
const tableCatalog = computed(() =>
  getTableCatalog(databaseCatalog.value, props.schemaName, props.tableName)
);

const initTableCatalog = computed(() => {
  if (!tableCatalog.value) {
    return toJsonString(
      TableCatalogSchema,
      create(TableCatalogSchema, {
        name: props.tableName,
      }),
      { prettySpaces: 2 }
    );
  }

  return toJsonString(TableCatalogSchema, tableCatalog.value, {
    prettySpaces: 2,
  });
});

const state = reactive<LocalState>({
  columnNameSearchKeyword: "",
  partitionTableNameSearchKeyword: "",
  tableCatalog: "{}",
  isUploading: false,
});
const isFetchingTableMetadata = ref(false);

const table = computedAsync(
  async () => {
    const { databaseName, tableName, schemaName } = props;
    if (!tableName) {
      return undefined;
    }
    return dbSchemaStore.getOrFetchTableMetadata({
      database: databaseName,
      schema: schemaName,
      table: tableName,
      silent: false,
    });
  },
  undefined,
  {
    evaluating: isFetchingTableMetadata,
  }
);

const hasDatabaseCatalogPermission = computed(() => {
  return hasProjectPermissionV2(
    database.value.projectEntity,
    "bb.databaseCatalogs.update"
  );
});

watch(
  () => initTableCatalog.value,
  (catalog) => {
    state.tableCatalog = catalog;
  },
  { immediate: true, deep: true }
);

const onCatalogUpload = async () => {
  const catalog = fromJsonString(TableCatalogSchema, state.tableCatalog);
  if (catalog.name !== props.tableName) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: `catalog name must be ${props.tableName}`,
    });
    return;
  }
  state.isUploading = true;

  try {
    const pendingUploadCatalog = cloneDeep(databaseCatalog.value);
    const schemaCatalog = pendingUploadCatalog.schemas.find(
      (schemaCatalog) => schemaCatalog.name === props.schemaName
    );
    if (schemaCatalog) {
      const index = schemaCatalog.tables.findIndex(
        (t) => t.name === props.tableName
      );
      if (index >= 0) {
        schemaCatalog.tables[index] = catalog;
      } else {
        schemaCatalog.tables.push(catalog);
      }
    } else {
      pendingUploadCatalog.schemas.push(
        create(SchemaCatalogSchema, {
          name: props.schemaName,
          tables: [catalog],
        })
      );
    }
    await catalogStore.updateDatabaseCatalog(pendingUploadCatalog);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    state.isUploading = false;
  }
};

const database = computed(() => {
  return databaseV1Store.getDatabaseByName(props.databaseName);
});

const instanceEngine = computed(() => {
  return database.value.instanceResource.engine;
});

const instanceEngineNew = computed(() => {
  return instanceEngine.value;
});

const allowQuery = computed(() => {
  if (database.value.project === DEFAULT_PROJECT_NAME) {
    return hasProjectPermissionV2(defaultProject(), "bb.sql.select");
  }
  return isDatabaseV1Queryable(database.value);
});

const hasPartitionTables = computed(() => {
  return (
    // Only show partition tables for PostgreSQL.
    database.value.instanceResource.engine === Engine.POSTGRES &&
    table.value &&
    table.value.partitions.length > 0
  );
});

const shouldShowPartitionTablesDataTable = computed(() => {
  return hasPartitionTables.value;
});

const getTableName = (tableName: string) => {
  if (hasSchemaProperty(instanceEngine.value) && props.schemaName) {
    return `"${props.schemaName}"."${tableName}"`;
  }
  return tableName;
};
</script>
