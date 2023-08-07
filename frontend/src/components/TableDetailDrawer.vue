<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    :close-on-esc="true"
    @update:show="(show: boolean) => !show && emit('dismiss')"
  >
    <NDrawerContent
      class="w-[calc(100vw-256px)]"
      :title="$t('database.table-detail')"
      :closable="true"
    >
      <div
        v-if="table"
        class="flex-1 overflow-auto focus:outline-none"
        tabindex="0"
      >
        <main class="flex-1 relative pb-8 overflow-y-auto">
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
                    :environment="database.instanceEntity.environmentEntity"
                    icon-class="textinfolabel"
                  />
                </dd>
                <dt class="sr-only">{{ $t("common.instance") }}</dt>
                <dd class="flex items-center text-sm md:mr-4">
                  <span class="ml-1 textlabel"
                    >{{ $t("common.instance") }}&nbsp;-&nbsp;</span
                  >
                  <InstanceV1Name :instance="database.instanceEntity" />
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
                  :label="true"
                  :disabled="!allowQuery"
                />
              </dl>
            </div>
          </div>

          <div class="mt-6">
            <div
              class="max-w-6xl mx-auto px-6 space-y-6 divide-y divide-block-border"
            >
              <!-- Description list -->
              <dl class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-2">
                <div class="col-span-1 col-start-1">
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.engine") }}
                  </dt>
                  <dd class="mt-1 text-sm text-main">
                    {{
                      instanceEngine === Engine.POSTGRES ||
                      instanceEngine === Engine.SNOWFLAKE
                        ? "n/a"
                        : table.engine
                    }}
                  </dd>
                </div>

                <div class="col-span-1">
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.row-count-estimate") }}
                  </dt>
                  <dd class="mt-1 text-sm text-main">{{ table.rowCount }}</dd>
                </div>

                <div class="col-span-1 col-start-1">
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.data-size") }}
                  </dt>
                  <dd class="mt-1 text-sm text-main">
                    {{ bytesToString(table.dataSize) }}
                  </dd>
                </div>

                <div class="col-span-1">
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.index-size") }}
                  </dt>
                  <dd class="mt-1 text-sm text-main">
                    {{
                      instanceEngine === Engine.CLICKHOUSE ||
                      instanceEngine === Engine.SNOWFLAKE
                        ? "n/a"
                        : bytesToString(table.indexSize)
                    }}
                  </dd>
                </div>

                <template
                  v-if="
                    instanceEngine !== Engine.CLICKHOUSE &&
                    instanceEngine !== Engine.SNOWFLAKE
                  "
                >
                  <div class="col-span-1">
                    <dt class="text-sm font-medium text-control-light">
                      {{ $t("db.collation") }}
                    </dt>
                    <dd class="mt-1 text-sm text-main">
                      {{
                        instanceEngine === Engine.POSTGRES
                          ? "n/a"
                          : table.collation
                      }}
                    </dd>
                  </div>
                </template>
              </dl>
            </div>
          </div>

          <div v-if="shouldShowColumnTable" class="mt-6 px-6">
            <div class="text-lg leading-6 font-medium text-main mb-4">
              {{ $t("database.columns") }}
            </div>
            <ColumnTable
              :database="database"
              :schema="schemaName"
              :table="table"
              :column-list="table.columns"
              :sensitive-data-list="sensitiveDataList"
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
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { NDrawer, NDrawerContent } from "naive-ui";
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import {
  bytesToString,
  hasWorkspacePermissionV1,
  isDatabaseV1Queryable,
  isGhostTable,
} from "@/utils";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useDBSchemaV1Store,
} from "@/store";
import { DEFAULT_PROJECT_V1_NAME, EMPTY_PROJECT_NAME } from "@/types";
import { TableMetadata } from "@/types/proto/store/database";
import ColumnTable from "./ColumnTable.vue";
import IndexTable from "./IndexTable.vue";
import { SQLEditorButtonV1 } from "./DatabaseDetail";
import { usePolicyByParentAndType } from "@/store/modules/v1/policy";
import { PolicyType, SensitiveData } from "@/types/proto/v1/org_policy_service";
import { Engine } from "@/types/proto/v1/common";
import { DatabaseV1Name, InstanceV1Name } from "@/components/v2";

const props = defineProps<{
  // Format: /databases/:databaseName
  databaseName: string;
  schemaName: string;
  tableName: string;
}>();

const emit = defineEmits(["dismiss"]);

const router = useRouter();
const databaseV1Store = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const currentUserV1 = useCurrentUserV1();
const table = ref<TableMetadata>();

const database = computed(() => {
  return databaseV1Store.getDatabaseByName(props.databaseName);
});
const instanceEngine = computed(() => {
  return database.value.instanceEntity.engine;
});

const allowQuery = computed(() => {
  if (
    database.value.project === EMPTY_PROJECT_NAME ||
    database.value.project === DEFAULT_PROJECT_V1_NAME
  ) {
    return hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-database",
      currentUserV1.value.userRole
    );
  }
  return isDatabaseV1Queryable(database.value, currentUserV1.value);
});
const hasSchemaProperty = computed(
  () => instanceEngine.value === Engine.POSTGRES
);
const shouldShowColumnTable = computed(() => {
  return instanceEngine.value !== Engine.MONGODB;
});
const getTableName = (tableName: string) => {
  if (hasSchemaProperty.value) {
    return `"${props.schemaName}"."${tableName}"`;
  }
  return tableName;
};

onMounted(() => {
  const schemaList = dbSchemaStore.getSchemaList(database.value.name);
  const schema = schemaList.find((schema) => schema.name === props.schemaName);
  if (schema) {
    table.value = schema.tables.find((table) => table.name === props.tableName);
  }
  if (!table.value) {
    router.replace({
      name: "error.404",
    });
  }
});

const sensitiveDataPolicy = usePolicyByParentAndType(
  computed(() => ({
    parentPath: database.value.name,
    policyType: PolicyType.SENSITIVE_DATA,
  }))
);

const sensitiveDataList = computed((): SensitiveData[] => {
  const policy = sensitiveDataPolicy.value;
  if (!policy || !policy.sensitiveDataPolicy) {
    return [];
  }

  return policy.sensitiveDataPolicy.sensitiveData;
});
</script>
