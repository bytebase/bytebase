<template>
  <div>
    <!-- Description list -->
    <DatabaseOverviewInfo :database="database" class="pb-6" />

    <div v-if="allowGetSchema" class="border-t border-block-border pt-6">
      <div
        v-if="hasSchemaPropertyV1"
        class="flex flex-row justify-start items-center mb-4"
      >
        <span class="text-lg leading-6 font-medium text-main mr-2">Schema</span>
        <NSelect
          v-model:value="state.selectedSchemaName"
          :options="schemaNameOptions"
          :disabled="state.loading"
          :placeholder="$t('database.schema.select')"
          class="w-auto! min-w-48"
        />
      </div>

      <template v-if="databaseEngine !== Engine.REDIS">
        <div class="mb-4 w-full flex flex-row justify-between items-center">
          <div class="text-lg leading-6 font-medium text-main">
            <span v-if="databaseEngine === Engine.MONGODB">
              {{ $t("db.collections") }}
            </span>
            <span v-else>{{ $t("db.tables") }}</span>
          </div>
          <SearchBox
            v-model:value="state.tableNameSearchKeyword"
            :placeholder="$t('common.filter-by-name')"
            :disabled="state.loading"
          />
        </div>

        <TableDataTable
          :database="database"
          :schema-name="state.selectedSchemaName"
          :table-list="tableList"
          :search="state.tableNameSearchKeyword.trim().toLowerCase()"
          :loading="state.loading"
        />

        <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
          {{ $t("db.views") }}
        </div>
        <ViewDataTable
          :database="database"
          :schema-name="state.selectedSchemaName"
          :view-list="viewList"
          :loading="state.loading"
        />

        <template
          v-if="
            databaseEngine === Engine.POSTGRES || databaseEngine === Engine.HIVE
          "
        >
          <div
            class="mt-6 w-full flex flex-row justify-between items-center mb-4"
          >
            <div class="text-lg leading-6 font-medium text-main">
              {{ $t("db.external-tables") }}
            </div>
            <SearchBox
              v-model:value="state.externalTableNameSearchKeyword"
              :placeholder="$t('common.filter-by-name')"
              :disabled="state.loading"
            />
          </div>
          <ExternalTableDataTable
            :database="database"
            :schema-name="state.selectedSchemaName"
            :external-table-list="externalTableList"
            :search="state.externalTableNameSearchKeyword.trim().toLowerCase()"
            :loading="state.loading"
          />
        </template>

        <template v-if="databaseEngine === Engine.POSTGRES">
          <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
            {{ $t("db.extensions") }}
          </div>
          <DBExtensionDataTable
            :db-extension-list="dbExtensionList"
            :loading="state.loading"
          />
        </template>

        <template
          v-if="
            databaseEngine === Engine.POSTGRES ||
            databaseEngine === Engine.MSSQL
          "
        >
          <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
            {{ $t("db.functions") }}
          </div>
          <FunctionDataTable
            :database="database"
            :schema-name="state.selectedSchemaName"
            :function-list="functionList"
            :loading="state.loading"
          />
        </template>

        <template v-if="instanceV1SupportsSequence(databaseEngine)">
          <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
            {{ $t("db.sequences") }}
          </div>
          <SequenceDataTable
            :database="database"
            :schema-name="state.selectedSchemaName"
            :sequence-list="sequenceList"
            :loading="state.loading"
          />
        </template>

        <template v-if="databaseEngine === Engine.SNOWFLAKE">
          <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
            {{ $t("db.streams") }}
          </div>
          <StreamTable
            :database="database"
            :schema-name="state.selectedSchemaName"
            :stream-list="streamList"
            :loading="state.loading"
          />

          <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
            {{ $t("db.tasks") }}
          </div>
          <TaskTable
            :database="database"
            :schema-name="state.selectedSchemaName"
            :task-list="taskList"
            :loading="state.loading"
          />
        </template>

        <template v-if="instanceV1SupportsPackage(databaseEngine)">
          <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
            {{ $t("db.packages") }}
          </div>
          <PackageDataTable
            :database="database"
            :schema-name="state.selectedSchemaName"
            :package-list="packageList"
            :loading="state.loading"
          />
        </template>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { NSelect } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { useDatabaseDetailContext } from "@/components/Database/context";
import DBExtensionDataTable from "@/components/DBExtensionDataTable.vue";
import ExternalTableDataTable from "@/components/ExternalTableDataTable.vue";
import FunctionDataTable from "@/components/FunctionDataTable.vue";
import StreamTable from "@/components/StreamTable.vue";
import TableDataTable from "@/components/TableDataTable.vue";
import TaskTable from "@/components/TaskTable.vue";
import ViewDataTable from "@/components/ViewDataTable.vue";
import { useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  hasSchemaProperty,
  instanceV1SupportsPackage,
  instanceV1SupportsSequence,
} from "@/utils";
import PackageDataTable from "../PackageDataTable.vue";
import SequenceDataTable from "../SequenceDataTable.vue";
import { SearchBox } from "../v2";
import DatabaseOverviewInfo from "./DatabaseOverviewInfo.vue";

interface LocalState {
  loading: boolean;
  selectedSchemaName?: string;
  tableNameSearchKeyword: string;
  externalTableNameSearchKeyword: string;
}

const props = defineProps<{
  database: ComposedDatabase;
}>();

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const state = reactive<LocalState>({
  loading: true,
  selectedSchemaName: "",
  tableNameSearchKeyword: "",
  externalTableNameSearchKeyword: "",
});

const { allowGetSchema } = useDatabaseDetailContext();

const dbSchemaStore = useDBSchemaV1Store();

const databaseEngine = computed(() => {
  return props.database.instanceResource.engine;
});

const hasSchemaPropertyV1 = computed(() => {
  return hasSchemaProperty(databaseEngine.value);
});

watch(
  () => props.database.name,
  async (database) => {
    state.loading = true;
    await dbSchemaStore.getOrFetchDatabaseMetadata({
      database,
      skipCache: false,
    });
    if (schemaList.value.length > 0) {
      const schemaInQuery = route.query.schema as string;
      if (
        schemaInQuery &&
        schemaList.value.find((schema) => schema.name === schemaInQuery)
      ) {
        state.selectedSchemaName = schemaInQuery;
      } else {
        const publicSchema = schemaList.value.find(
          (schema) => schema.name.toLowerCase() === "public"
        );
        if (publicSchema) {
          state.selectedSchemaName = publicSchema.name;
        } else {
          state.selectedSchemaName = head(schemaList.value)?.name || "";
        }
      }
    } else {
      state.selectedSchemaName = undefined;
    }
    state.loading = false;
  },
  { immediate: true }
);

const schemaList = computed(() => {
  return dbSchemaStore.getSchemaList(props.database.name);
});

const schemaNameOptions = computed(() => {
  return schemaList.value.map((schema) => ({
    value: schema.name,
    label: schema.name || t("db.schema.default"),
  }));
});

const tableList = computed(() => {
  return dbSchemaStore.getTableList({
    database: props.database.name,
    schema: state.selectedSchemaName,
  });
});

const viewList = computed(() => {
  return dbSchemaStore.getViewList({
    database: props.database.name,
    schema: state.selectedSchemaName,
  });
});

const dbExtensionList = computed(() => {
  return dbSchemaStore.getExtensionList(props.database.name);
});

const externalTableList = computed(() => {
  return dbSchemaStore.getExternalTableList({
    database: props.database.name,
    schema: state.selectedSchemaName,
  });
});

const functionList = computed(() => {
  return dbSchemaStore.getFunctionList({
    database: props.database.name,
    schema: state.selectedSchemaName,
  });
});

const streamList = computed(() => {
  if (hasSchemaPropertyV1.value) {
    return (
      schemaList.value.find(
        (schema) => schema.name === state.selectedSchemaName
      )?.streams || []
    );
  }
  return dbSchemaStore
    .getDatabaseMetadata(props.database.name)
    .schemas.map((schema) => schema.streams)
    .flat();
});

const taskList = computed(() => {
  if (hasSchemaPropertyV1.value) {
    return (
      schemaList.value.find(
        (schema) => schema.name === state.selectedSchemaName
      )?.tasks || []
    );
  }
  return dbSchemaStore
    .getDatabaseMetadata(props.database.name)
    .schemas.map((schema) => schema.tasks)
    .flat();
});

const packageList = computed(() => {
  if (hasSchemaPropertyV1.value) {
    return (
      schemaList.value.find(
        (schema) => schema.name === state.selectedSchemaName
      )?.packages || []
    );
  }
  return dbSchemaStore
    .getDatabaseMetadata(props.database.name)
    .schemas.map((schema) => schema.packages)
    .flat();
});

const sequenceList = computed(() => {
  if (hasSchemaPropertyV1.value) {
    return (
      schemaList.value.find(
        (schema) => schema.name === state.selectedSchemaName
      )?.sequences || []
    );
  }
  return dbSchemaStore
    .getDatabaseMetadata(props.database.name)
    .schemas.map((schema) => schema.sequences)
    .flat();
});

watch(
  () => state.selectedSchemaName,
  (schema) => {
    router.replace({
      query: {
        schema: schema ? schema : undefined,
      },
    });
  }
);
</script>
