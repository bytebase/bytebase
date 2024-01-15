<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent :title="$t('database.foreign-table-detail')">
      <div
        v-if="externalTable"
        class="flex-1 overflow-auto focus:outline-none w-[calc(100vw-256px)]"
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
                      {{ getTableName(externalTable.name) }}
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
            <div class="max-w-6xl px-6 space-y-6 divide-y divide-block-border">
              <!-- Description list -->
              <dl class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-3">
                <div class="col-span-1">
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.external-server-name") }}
                  </dt>
                  <dd class="mt-1 text-sm text-main">
                    {{ externalTable.externalServerName }}
                  </dd>
                </div>

                <div class="col-span-1">
                  <dt class="text-sm font-medium text-control-light">
                    {{ $t("database.external-database-name") }}
                  </dt>
                  <dd class="mt-1 text-sm text-main">
                    {{ externalTable.externalDatabaseName }}
                  </dd>
                </div>
              </dl>
            </div>
          </div>

          <div class="mt-6 px-6">
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
              :table="TableMetadata.fromPartial({})"
              :column-list="externalTable.columns"
              :is-external-table="true"
              :mask-data-list="[]"
              :search="state.columnNameSearchKeyword"
            />
          </div>
        </main>
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { computed, watch, ref, reactive } from "vue";
import {
  DatabaseV1Name,
  InstanceV1Name,
  Drawer,
  DrawerContent,
} from "@/components/v2";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useDBSchemaV1Store,
} from "@/store";
import { DEFAULT_PROJECT_V1_NAME, defaultProject } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  ExternalTableMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { hasProjectPermissionV2, isDatabaseV1Queryable } from "@/utils";
import ColumnDataTable from "./ColumnDataTable/index.vue";
import { SQLEditorButtonV1 } from "./DatabaseDetail";

interface LocalState {
  columnNameSearchKeyword: string;
  partitionTableNameSearchKeyword: string;
}

const props = defineProps<{
  show: boolean;
  // Format: /databases/:databaseName
  databaseName: string;
  schemaName: string;
  externalTableName: string;
}>();

defineEmits(["dismiss"]);

const dbSchemaStore = useDBSchemaV1Store();
const databaseV1Store = useDatabaseV1Store();
const currentUserV1 = useCurrentUserV1();
const state = reactive<LocalState>({
  columnNameSearchKeyword: "",
  partitionTableNameSearchKeyword: "",
});
const externalTable = ref<ExternalTableMetadata>();

const database = computed(() => {
  return databaseV1Store.getDatabaseByName(props.databaseName);
});

const instanceEngine = computed(() => {
  return database.value.instanceEntity.engine;
});

const allowQuery = computed(() => {
  if (database.value.project === DEFAULT_PROJECT_V1_NAME) {
    return hasProjectPermissionV2(
      defaultProject,
      currentUserV1.value,
      "bb.databases.query"
    );
  }
  return isDatabaseV1Queryable(database.value, currentUserV1.value);
});

const hasSchemaProperty = computed(
  () =>
    instanceEngine.value === Engine.POSTGRES ||
    instanceEngine.value === Engine.RISINGWAVE
);

const getTableName = (tableName: string) => {
  if (hasSchemaProperty.value) {
    return `"${props.schemaName}"."${tableName}"`;
  }
  return tableName;
};

watch(
  () => [props.externalTableName, props.schemaName],
  ([externalTableName, schemaName]) => {
    if (!externalTableName) {
      return;
    }
    const schemaList = dbSchemaStore.getSchemaList(database.value.name);
    const schema = schemaList.find((schema) => schema.name === schemaName);
    if (schema) {
      externalTable.value = schema.externalTables.find((t) => {
        if (t.name === externalTableName) {
          externalTable.value = t;
          return true;
        }
        return false;
      });
    }
  }
);
</script>
