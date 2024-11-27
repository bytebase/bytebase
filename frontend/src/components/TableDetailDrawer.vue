<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent
      :title="$t('database.table-detail')"
      :content-props="{
        bodyContentClass: 'relative',
      }"
    >
      <div class="focus:outline-none w-[calc(100vw-256px)]" tabindex="0">
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
                      <BBBadge
                        v-if="isGhostTable(table)"
                        text="gh-ost"
                        :can-remove="false"
                        class="text-xs"
                      />
                    </h1>
                  </div>
                </div>
              </div>
              <dl
                class="flex flex-col space-y-1 md:space-y-0 md:flex-row md:flex-wrap"
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
                  class="text-sm md:mr-4"
                  :database="database"
                  :schema="schemaName"
                  :table="tableName"
                  :label="true"
                  :disabled="!allowQuery"
                />
                <NPopover
                  trigger="click"
                  placement="bottom"
                  @update:show="(show: boolean) => show"
                >
                  <template #trigger>
                    <NButton
                      quaternary
                      size="tiny"
                      class="!px-1"
                      v-bind="$attrs"
                    >
                      <span class="textlabel"
                        >{{ $t("database.view-definition") }}
                      </span>
                      <CodeIcon class="ml-1 w-4 h-4" />
                    </NButton>
                  </template>
                  <TableSchemaViewer
                    class="!w-[32rem] !h-[20rem]"
                    :database="database"
                    :schema="schemaName"
                    :table="tableName"
                  />
                </NPopover>
              </dl>
            </div>
          </div>

          <div class="mt-6">
            <div class="max-w-6xl px-6 space-y-6 divide-y divide-block-border">
              <!-- Description list -->
              <dl class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-3">
                <div
                  v-if="hasTableEngineProperty(instanceEngine)"
                  class="col-span-1"
                >
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.engine") }}
                  </dt>
                  <dd class="mt-1 text-semibold">
                    {{ table.engine }}
                  </dd>
                </div>

                <div v-if="classificationConfig" class="col-span-1">
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.classification.self") }}
                  </dt>
                  <dd class="mt-1 font-semibold">
                    <ClassificationLevelBadge
                      :classification="tableConfig.classificationId"
                      :classification-config="classificationConfig"
                      placeholder="-"
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
                    {{ bytesToString(table.dataSize.toNumber()) }}
                  </dd>
                </div>

                <div
                  v-if="hasIndexSizeProperty(instanceEngine)"
                  class="col-span-1"
                >
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.index-size") }}
                  </dt>
                  <dd class="mt-1 text-semibold">
                    {{ bytesToString(table.indexSize.toNumber()) }}
                  </dd>
                </div>

                <template v-if="hasCollationProperty(instanceEngine)">
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

          <div v-if="shouldShowColumnTable" class="mt-6 px-6">
            <div class="mb-4 w-full flex flex-row justify-between items-center">
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
              :column-list="table.columns"
              :mask-data-list="sensitiveDataList"
              :classification-config="classificationConfig"
              :search="state.columnNameSearchKeyword"
            />
          </div>

          <div v-if="instanceEngine !== Engine.SNOWFLAKE" class="mt-6 px-6">
            <div class="text-lg leading-6 font-medium text-main mb-4">
              {{ $t("database.indexes") }}
            </div>
            <IndexTable :database="database" :index-list="table.indexes" />
          </div>
        </main>
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { CodeIcon } from "lucide-vue-next";
import { NButton, NPopover } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { BBBadge } from "@/bbkit";
import ClassificationLevelBadge from "@/components/SchemaTemplate/ClassificationLevelBadge.vue";
import TableSchemaViewer from "@/components/TableSchemaViewer.vue";
import {
  DatabaseV1Name,
  InstanceV1Name,
  Drawer,
  DrawerContent,
  EnvironmentV1Name,
  ProjectV1Name,
  SearchBox,
} from "@/components/v2";
import { useDatabaseV1Store, useDBSchemaV1Store } from "@/store";
import { usePolicyByParentAndType } from "@/store/modules/v1/policy";
import { DEFAULT_PROJECT_NAME, defaultProject } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { MaskData } from "@/types/proto/v1/org_policy_service";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import type { DataClassificationSetting_DataClassificationConfig } from "@/types/proto/v1/setting_service";
import {
  bytesToString,
  hasCollationProperty,
  hasIndexSizeProperty,
  hasProjectPermissionV2,
  hasSchemaProperty,
  hasTableEngineProperty,
  isDatabaseV1Queryable,
  isGhostTable,
} from "@/utils";
import ColumnDataTable from "./ColumnDataTable/index.vue";
import { SQLEditorButtonV1 } from "./DatabaseDetail";
import IndexTable from "./IndexTable.vue";
import PartitionTablesDataTable from "./PartitionTablesDataTable.vue";
import MaskSpinner from "./misc/MaskSpinner.vue";

interface LocalState {
  columnNameSearchKeyword: string;
  partitionTableNameSearchKeyword: string;
}

const props = defineProps<{
  show: boolean;
  // Format: /databases/:databaseName
  databaseName: string;
  schemaName: string;
  tableName: string;
  classificationConfig?: DataClassificationSetting_DataClassificationConfig;
}>();

defineEmits(["dismiss"]);

const databaseV1Store = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const state = reactive<LocalState>({
  columnNameSearchKeyword: "",
  partitionTableNameSearchKeyword: "",
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
      skipCache: true,
      silent: false,
    });
  },
  undefined,
  {
    evaluating: isFetchingTableMetadata,
  }
);

const tableConfig = computed(() =>
  dbSchemaStore.getTableConfig(
    props.databaseName,
    props.schemaName,
    props.tableName
  )
);

const database = computed(() => {
  return databaseV1Store.getDatabaseByName(props.databaseName);
});

const instanceEngine = computed(() => {
  return database.value.instanceResource.engine;
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

const shouldShowColumnTable = computed(() => {
  return instanceEngine.value !== Engine.MONGODB;
});

const getTableName = (tableName: string) => {
  if (hasSchemaProperty(instanceEngine.value) && props.schemaName) {
    return `"${props.schemaName}"."${tableName}"`;
  }
  return tableName;
};

const sensitiveDataPolicy = usePolicyByParentAndType(
  computed(() => ({
    parentPath: database.value.name,
    policyType: PolicyType.MASKING,
  }))
);

const sensitiveDataList = computed((): MaskData[] => {
  const policy = sensitiveDataPolicy.value;
  if (!policy || !policy.maskingPolicy) {
    return [];
  }

  return policy.maskingPolicy.maskData;
});
</script>
