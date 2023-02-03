<template>
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
              <router-link
                :to="`/environment/${environmentSlug(
                  database.instance.environment
                )}`"
                class="normal-link"
                >{{
                  environmentName(database.instance.environment)
                }}</router-link
              >
            </dd>
            <dt class="sr-only">{{ $t("common.instance") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <InstanceEngineIcon :instance="database.instance" />
              <span class="ml-1 textlabel"
                >{{ $t("common.instance") }}&nbsp;-&nbsp;</span
              >
              <router-link
                :to="`/instance/${instanceSlug(database.instance)}`"
                class="normal-link"
                >{{ instanceName(database.instance) }}</router-link
              >
            </dd>
            <dt class="sr-only">{{ $t("common.project") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.project") }}&nbsp;-&nbsp;</span
              >
              <router-link
                :to="`/project/${projectSlug(database.project)}#databases`"
                class="normal-link"
                >{{ projectName(database.project) }}</router-link
              >
            </dd>
            <dt class="sr-only">{{ $t("common.database") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.database") }}&nbsp;-&nbsp;</span
              >
              <router-link :to="`/db/${databaseSlug}`" class="normal-link">{{
                database.name
              }}</router-link>
            </dd>
            <SQLEditorButton
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
                  database.instance.engine == "POSTGRES" ||
                  database.instance.engine == "SNOWFLAKE"
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
                  database.instance.engine == "CLICKHOUSE" ||
                  database.instance.engine == "SNOWFLAKE"
                    ? "n/a"
                    : bytesToString(table.indexSize)
                }}
              </dd>
            </div>

            <template
              v-if="
                database.instance.engine != 'CLICKHOUSE' &&
                database.instance.engine != 'SNOWFLAKE'
              "
            >
              <div class="col-span-1">
                <dt class="text-sm font-medium text-control-light">
                  {{ $t("db.collation") }}
                </dt>
                <dd class="mt-1 text-sm text-main">
                  {{
                    database.instance.engine == "POSTGRES"
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
          :table="table"
          :column-list="table.columns"
          :sensitive-data-list="sensitiveDataList"
        />
      </div>

      <div v-if="database.instance.engine != 'SNOWFLAKE'" class="mt-6 px-6">
        <div class="text-lg leading-6 font-medium text-main mb-4">
          {{ $t("database.indexes") }}
        </div>
        <IndexTable :database="database" :index-list="table.indexes" />
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  bytesToString,
  hasWorkspacePermission,
  idFromSlug,
  isDatabaseAccessible,
  isGhostTable,
} from "@/utils";
import {
  useCurrentUser,
  useDatabaseStore,
  useDBSchemaStore,
  usePolicyByDatabaseAndType,
} from "@/store";
import {
  DEFAULT_PROJECT_ID,
  SensitiveData,
  SensitiveDataPolicyPayload,
  UNKNOWN_ID,
} from "@/types";
import { TableMetadata } from "@/types/proto/store/database";
import ColumnTable from "../components/ColumnTable.vue";
import IndexTable from "../components/IndexTable.vue";
import InstanceEngineIcon from "../components/InstanceEngineIcon.vue";
import { SQLEditorButton } from "@/components/DatabaseDetail";

export default defineComponent({
  name: "TableDetail",
  components: { ColumnTable, IndexTable, InstanceEngineIcon, SQLEditorButton },
  props: {
    databaseSlug: {
      required: true,
      type: String,
    },
    tableName: {
      required: true,
      type: String,
    },
  },
  setup(props) {
    const route = useRoute();
    const router = useRouter();
    const databaseStore = useDatabaseStore();
    const dbSchemaStore = useDBSchemaStore();
    const currentUser = useCurrentUser();
    const table = ref<TableMetadata>();
    const databaseId = idFromSlug(props.databaseSlug);
    const schemaName = (route.query.schema as string) || "";

    const database = computed(() => {
      return databaseStore.getDatabaseById(databaseId);
    });
    const instanceEngine = computed(() => {
      return database.value.instance.engine;
    });

    const accessControlPolicy = usePolicyByDatabaseAndType(
      computed(() => ({
        databaseId: database.value.id,
        type: "bb.policy.access-control",
      }))
    );
    const allowQuery = computed(() => {
      if (
        database.value.projectId === UNKNOWN_ID ||
        database.value.projectId === DEFAULT_PROJECT_ID
      ) {
        return hasWorkspacePermission(
          "bb.permission.workspace.manage-database",
          currentUser.value.role
        );
      }
      const policy = accessControlPolicy.value;
      const list = policy ? [policy] : [];
      return isDatabaseAccessible(database.value, list, currentUser.value);
    });
    const hasSchemaProperty = computed(
      () => instanceEngine.value === "POSTGRES"
    );
    const shouldShowColumnTable = computed(() => {
      return instanceEngine.value !== "MONGODB";
    });
    const getTableName = (tableName: string) => {
      if (hasSchemaProperty.value) {
        return `"${schemaName}"."${tableName}"`;
      }
      return tableName;
    };

    onMounted(() => {
      const schemaList = dbSchemaStore.getSchemaListByDatabaseId(databaseId);
      const schema = schemaList.find((schema) => schema.name === schemaName);
      if (schema) {
        table.value = schema.tables.find(
          (table) => table.name === props.tableName
        );
      }
      if (!table.value) {
        router.replace({
          name: "error.404",
        });
      }
    });

    const sensitiveDataPolicy = usePolicyByDatabaseAndType(
      computed(() => ({
        databaseId: database.value.id,
        type: "bb.policy.sensitive-data",
      }))
    );

    const sensitiveDataList = computed((): SensitiveData[] => {
      const policy = sensitiveDataPolicy.value;
      if (!policy) {
        return [];
      }
      const payload = policy.payload as SensitiveDataPolicyPayload;
      return payload.sensitiveDataList;
    });

    return {
      table,
      database,
      allowQuery,
      getTableName,
      bytesToString,
      isGhostTable,
      sensitiveDataList,
      shouldShowColumnTable,
    };
  },
});
</script>
