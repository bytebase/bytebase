<template>
  <div class="space-y-6 divide-y divide-block-border">
    <div v-if="anomalySectionList.length > 0">
      <div class="text-lg leading-6 font-medium text-main mb-4 flex flex-row">
        {{ $t("common.anomalies") }}
        <span class="ml-2 textinfolabel items-center flex">
          {{
            $t(
              "database.the-list-might-be-out-of-date-and-is-refreshed-roughly-every-10-minutes"
            )
          }}
        </span>
      </div>
      <AnomalyTable :anomaly-section-list="anomalySectionList" />
    </div>
    <div
      v-else
      class="text-lg leading-6 font-medium text-main mb-4 flex flex-row"
    >
      {{ $t("database.no-anomalies-detected") }}
      <heroicons-outline:check-circle class="ml-1 w-6 h-6 text-success" />
    </div>

    <!-- Description list -->
    <dl
      class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-2 pt-4"
      data-label="bb-database-overview-description-list"
    >
      <template
        v-if="
          databaseEngine !== Engine.CLICKHOUSE &&
          databaseEngine !== Engine.SNOWFLAKE &&
          databaseEngine !== Engine.MONGODB
        "
      >
        <div class="col-span-1 col-start-1">
          <dt class="text-sm font-medium text-control-light">
            {{
              databaseEngine === Engine.POSTGRES
                ? $t("db.encoding")
                : $t("db.character-set")
            }}
          </dt>
          <dd class="mt-1 text-sm text-main">
            {{ databaseSchemaMetadata.characterSet }}
          </dd>
        </div>

        <div class="col-span-1">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("db.collation") }}
          </dt>
          <dd class="mt-1 text-sm text-main">
            {{ databaseSchemaMetadata.collation }}
          </dd>
        </div>
      </template>

      <div class="col-span-1 col-start-1">
        <dt class="text-sm font-medium text-control-light">
          {{ $t("database.sync-status") }}
        </dt>
        <dd class="mt-1 text-sm text-main">
          <span>
            {{ database.syncState === State.ACTIVE ? "OK" : "NOT_FOUND" }}
          </span>
        </dd>
      </div>

      <div class="col-span-1">
        <dt class="text-sm font-medium text-control-light">
          {{ $t("database.last-successful-sync") }}
        </dt>
        <dd class="mt-1 text-sm text-main">
          {{ humanizeDate(database.successfulSyncTime) }}
        </dd>
      </div>
    </dl>

    <div class="pt-6">
      <div
        v-if="hasSchemaProperty"
        class="flex flex-row justify-start items-center mb-4"
      >
        <span class="text-lg leading-6 font-medium text-main mr-2">Schema</span>
        <BBSelect
          class="!w-auto min-w-[12rem]"
          :selected-item="state.selectedSchemaName"
          :item-list="schemaNameList"
          :placeholder="$t('database.schema.select')"
          :show-prefix-item="true"
          @select-item="(schema: string) => state.selectedSchemaName = schema"
        >
          <template #menuItem="{ item: schema }">
            {{ schema }}
          </template>
        </BBSelect>
      </div>

      <template v-if="databaseEngine !== Engine.REDIS">
        <div class="text-lg leading-6 font-medium text-main mb-4">
          <span v-if="databaseEngine === Engine.MONGODB">{{
            $t("db.collections")
          }}</span>
          <span v-else>{{ $t("db.tables") }}</span>
        </div>

        <TableTable
          :database="database"
          :schema-name="state.selectedSchemaName"
          :table-list="tableList"
        />

        <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
          {{ $t("db.views") }}
        </div>
        <ViewTable
          :database="database"
          :schema-name="state.selectedSchemaName"
          :view-list="viewList"
        />

        <template v-if="databaseEngine === Engine.POSTGRES">
          <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
            {{ $t("db.extensions") }}
          </div>
          <DBExtensionTable :db-extension-list="dbExtensionList" />
        </template>

        <template v-if="databaseEngine === Engine.POSTGRES">
          <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
            {{ $t("db.functions") }}
          </div>
          <FunctionTable
            :database="database"
            :schema-name="state.selectedSchemaName"
            :function-list="functionList"
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
          />

          <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
            {{ $t("db.tasks") }}
          </div>
          <TaskTable
            :database="database"
            :schema-name="state.selectedSchemaName"
            :task-list="taskList"
          />
        </template>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { computed, reactive, watchEffect, PropType } from "vue";
import { useAnomalyV1List, useDBSchemaV1Store } from "@/store";
import { Anomaly } from "@/types/proto/v1/anomaly_service";
import { Engine, State } from "@/types/proto/v1/common";
import { BBTableSectionDataSource } from "../bbkit/types";
import AnomalyTable from "../components/AnomalyTable.vue";
import FunctionTable from "../components/FunctionTable.vue";
import TableTable from "../components/TableTable.vue";
import ViewTable from "../components/ViewTable.vue";
import { ComposedDatabase, DataSource } from "../types";
import StreamTable from "./StreamTable.vue";
import TaskTable from "./TaskTable.vue";

interface LocalState {
  selectedSchemaName: string;
  editingDataSource?: DataSource;
}

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
});

const state = reactive<LocalState>({
  selectedSchemaName: "",
});

const dbSchemaStore = useDBSchemaV1Store();

const databaseEngine = computed(() => {
  return props.database.instanceEntity.engine;
});

const hasSchemaProperty = computed(() => {
  return (
    databaseEngine.value === Engine.POSTGRES ||
    databaseEngine.value === Engine.SNOWFLAKE ||
    databaseEngine.value === Engine.ORACLE ||
    databaseEngine.value === Engine.DM ||
    databaseEngine.value === Engine.MSSQL ||
    databaseEngine.value === Engine.REDSHIFT
  );
});

const prepareDatabaseMetadata = async () => {
  await dbSchemaStore.getOrFetchDatabaseMetadata(props.database.name);
  if (hasSchemaProperty.value && schemaList.value.length > 0) {
    const publicSchema = schemaList.value.find(
      (schema) => schema.name.toLowerCase() === "public"
    );
    if (publicSchema) {
      state.selectedSchemaName = publicSchema.name;
    } else {
      state.selectedSchemaName = head(schemaList.value)?.name || "";
    }
  }
};

watchEffect(prepareDatabaseMetadata);

const anomalyList = useAnomalyV1List(
  computed(() => ({ database: props.database.name }))
);

const anomalySectionList = computed((): BBTableSectionDataSource<Anomaly>[] => {
  const list: BBTableSectionDataSource<Anomaly>[] = [];
  if (anomalyList.value.length > 0) {
    list.push({
      title: props.database.name,
      list: anomalyList.value,
    });
  }
  return list;
});

const schemaList = computed(() => {
  return dbSchemaStore.getSchemaList(props.database.name);
});

const schemaNameList = computed(() => {
  return schemaList.value.map((schema) => schema.name);
});

const databaseSchemaMetadata = computed(() => {
  return dbSchemaStore.getDatabaseMetadata(props.database.name);
});

const tableList = computed(() => {
  if (hasSchemaProperty.value) {
    return (
      schemaList.value.find(
        (schema) => schema.name === state.selectedSchemaName
      )?.tables || []
    );
  }
  return dbSchemaStore.getTableList(props.database.name);
});

const viewList = computed(() => {
  if (hasSchemaProperty.value) {
    return (
      schemaList.value.find(
        (schema) => schema.name === state.selectedSchemaName
      )?.views || []
    );
  }
  return dbSchemaStore.getViewList(props.database.name);
});

const dbExtensionList = computed(() => {
  return dbSchemaStore.getExtensionList(props.database.name);
});

const functionList = computed(() => {
  if (hasSchemaProperty.value) {
    return (
      schemaList.value.find(
        (schema) => schema.name === state.selectedSchemaName
      )?.functions || []
    );
  }
  return dbSchemaStore.getFunctionList(props.database.name);
});

const streamList = computed(() => {
  if (hasSchemaProperty.value) {
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
  if (hasSchemaProperty.value) {
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
</script>
