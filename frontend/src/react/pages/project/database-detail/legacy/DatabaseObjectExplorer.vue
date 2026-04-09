<template>
  <div>
    <div
      v-if="hasSchemaPropertyV1"
      class="mb-4 flex flex-row items-center justify-start"
    >
      <span class="mr-2 text-lg leading-6 font-medium text-main">
        {{ $t("common.schema") }}
      </span>
      <NSelect
        :value="selectedSchemaName"
        :options="schemaNameOptions"
        :disabled="loading"
        :placeholder="$t('database.schema.select')"
        class="w-auto! min-w-48"
        @update:value="
          $emit('update:selected-schema-name', (($event as string) ?? '').trim())
        "
      />
    </div>

    <template v-if="databaseEngine !== Engine.REDIS">
      <div class="mb-4 flex w-full flex-row items-center justify-between">
        <div class="text-lg leading-6 font-medium text-main">
          <span v-if="databaseEngine === Engine.MONGODB">
            {{ $t("db.collections") }}
          </span>
          <span v-else>{{ $t("db.tables") }}</span>
        </div>
        <SearchBox
          :value="tableSearchKeyword"
          :placeholder="$t('common.filter-by-name')"
          :disabled="loading"
          @update:value="$emit('update:table-search-keyword', $event)"
        />
      </div>

      <TableDataTable
        :database="database"
        :schema-name="selectedSchemaName"
        :table-list="tableList"
        :search="tableSearchKeyword.trim().toLowerCase()"
        :loading="loading"
      />

      <div class="mb-4 mt-6 text-lg leading-6 font-medium text-main">
        {{ $t("db.views") }}
      </div>
      <ViewDataTable
        :database="database"
        :schema-name="selectedSchemaName"
        :view-list="viewList"
        :loading="loading"
      />

      <template
        v-if="
          databaseEngine === Engine.POSTGRES || databaseEngine === Engine.HIVE
        "
      >
        <div class="mb-4 mt-6 flex w-full flex-row items-center justify-between">
          <div class="text-lg leading-6 font-medium text-main">
            {{ $t("db.external-tables") }}
          </div>
          <SearchBox
            :value="externalTableSearchKeyword"
            :placeholder="$t('common.filter-by-name')"
            :disabled="loading"
            @update:value="$emit('update:external-table-search-keyword', $event)"
          />
        </div>
        <ExternalTableDataTable
          :database="database"
          :schema-name="selectedSchemaName"
          :external-table-list="externalTableList"
          :search="externalTableSearchKeyword.trim().toLowerCase()"
          :loading="loading"
        />
      </template>

      <template v-if="databaseEngine === Engine.POSTGRES">
        <div class="mb-4 mt-6 text-lg leading-6 font-medium text-main">
          {{ $t("db.extensions") }}
        </div>
        <DBExtensionDataTable
          :db-extension-list="dbExtensionList"
          :loading="loading"
        />
      </template>

      <template
        v-if="
          databaseEngine === Engine.POSTGRES || databaseEngine === Engine.MSSQL
        "
      >
        <div class="mb-4 mt-6 text-lg leading-6 font-medium text-main">
          {{ $t("db.functions") }}
        </div>
        <FunctionDataTable
          :database="database"
          :schema-name="selectedSchemaName"
          :function-list="functionList"
          :loading="loading"
        />
      </template>

      <template v-if="instanceV1SupportsSequence(databaseEngine)">
        <div class="mb-4 mt-6 text-lg leading-6 font-medium text-main">
          {{ $t("db.sequences") }}
        </div>
        <SequenceDataTable
          :database="database"
          :schema-name="selectedSchemaName"
          :sequence-list="sequenceList"
          :loading="loading"
        />
      </template>

      <template v-if="databaseEngine === Engine.SNOWFLAKE">
        <div class="mb-4 mt-6 text-lg leading-6 font-medium text-main">
          {{ $t("db.streams") }}
        </div>
        <StreamTable
          :database="database"
          :schema-name="selectedSchemaName"
          :stream-list="streamList"
          :loading="loading"
        />

        <div class="mb-4 mt-6 text-lg leading-6 font-medium text-main">
          {{ $t("db.tasks") }}
        </div>
        <TaskTable
          :database="database"
          :schema-name="selectedSchemaName"
          :task-list="taskList"
          :loading="loading"
        />
      </template>

      <template v-if="instanceV1SupportsPackage(databaseEngine)">
        <div class="mb-4 mt-6 text-lg leading-6 font-medium text-main">
          {{ $t("db.packages") }}
        </div>
        <PackageDataTable
          :database="database"
          :schema-name="selectedSchemaName"
          :package-list="packageList"
          :loading="loading"
        />
      </template>
    </template>
  </div>
</template>
<script setup lang="ts">
import { NSelect } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import DBExtensionDataTable from "@/components/DBExtensionDataTable.vue";
import ExternalTableDataTable from "@/components/ExternalTableDataTable.vue";
import FunctionDataTable from "@/components/FunctionDataTable.vue";
import PackageDataTable from "@/components/PackageDataTable.vue";
import SequenceDataTable from "@/components/SequenceDataTable.vue";
import StreamTable from "@/components/StreamTable.vue";
import TableDataTable from "@/components/TableDataTable.vue";
import TaskTable from "@/components/TaskTable.vue";
import ViewDataTable from "@/components/ViewDataTable.vue";
import { SearchBox } from "@/components/v2";
import { useDBSchemaV1Store } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  getDatabaseEngine,
  hasSchemaProperty,
  instanceV1SupportsPackage,
  instanceV1SupportsSequence,
} from "@/utils";

const props = defineProps<{
  database: Database;
  loading: boolean;
  selectedSchemaName: string;
  tableSearchKeyword: string;
  externalTableSearchKeyword: string;
}>();

defineEmits<{
  (event: "update:selected-schema-name", value: string): void;
  (event: "update:table-search-keyword", value: string): void;
  (event: "update:external-table-search-keyword", value: string): void;
}>();

const dbSchemaStore = useDBSchemaV1Store();
const { t } = useI18n();

const databaseEngine = computed(() => {
  return getDatabaseEngine(props.database);
});

const hasSchemaPropertyV1 = computed(() => {
  return hasSchemaProperty(databaseEngine.value);
});

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
    schema: props.selectedSchemaName,
  });
});

const viewList = computed(() => {
  return dbSchemaStore.getViewList({
    database: props.database.name,
    schema: props.selectedSchemaName,
  });
});

const dbExtensionList = computed(() => {
  return dbSchemaStore.getExtensionList(props.database.name);
});

const externalTableList = computed(() => {
  return dbSchemaStore.getExternalTableList({
    database: props.database.name,
    schema: props.selectedSchemaName,
  });
});

const functionList = computed(() => {
  return dbSchemaStore.getFunctionList({
    database: props.database.name,
    schema: props.selectedSchemaName,
  });
});

const streamList = computed(() => {
  if (hasSchemaPropertyV1.value) {
    return (
      schemaList.value.find(
        (schema) => schema.name === props.selectedSchemaName
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
        (schema) => schema.name === props.selectedSchemaName
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
        (schema) => schema.name === props.selectedSchemaName
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
        (schema) => schema.name === props.selectedSchemaName
      )?.sequences || []
    );
  }
  return dbSchemaStore
    .getDatabaseMetadata(props.database.name)
    .schemas.map((schema) => schema.sequences)
    .flat();
});
</script>
