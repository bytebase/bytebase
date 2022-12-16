<template>
  <div class="flex-1 overflow-auto focus:outline-none" tabindex="0">
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
                  {{ table.name }}

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

              <span class="ml-2 textlabel">
                {{ $t("sql-editor.self") }}
              </span>
              <button class="ml-1 btn-icon" @click.prevent="gotoSQLEditor">
                <heroicons-solid:terminal class="w-5 h-5" />
              </button>
            </dd>
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
              <div class="col-span-1 col-start-1">
                <dt class="text-sm font-medium text-control-light">
                  {{
                    database.instance.engine == "POSTGRES"
                      ? $t("db.encoding")
                      : $t("db.character-set")
                  }}
                </dt>
                <dd class="mt-1 text-sm text-main">
                  {{ database.characterSet }}
                </dd>
              </div>

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

      <div class="mt-6 px-6">
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
import { computed, defineComponent } from "vue";
import {
  bytesToString,
  connectionSlug,
  idFromSlug,
  isGhostTable,
} from "@/utils";
import {
  useDatabaseStore,
  useDBSchemaStore,
  usePolicyByDatabaseAndType,
} from "@/store";
import { SensitiveData, SensitiveDataPolicyPayload } from "@/types";
import { TableMetadata } from "@/types/proto/database";
import ColumnTable from "../components/ColumnTable.vue";
import IndexTable from "../components/IndexTable.vue";
import InstanceEngineIcon from "../components/InstanceEngineIcon.vue";

export default defineComponent({
  name: "TableDetail",
  components: { ColumnTable, IndexTable, InstanceEngineIcon },
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
    const databaseStore = useDatabaseStore();
    const databaseId = idFromSlug(props.databaseSlug);
    const dbSchemaStore = useDBSchemaStore();

    const database = computed(() => {
      return databaseStore.getDatabaseById(databaseId);
    });
    const table = computed(() => {
      return dbSchemaStore.getTableByDatabaseIdAndTableName(
        databaseId,
        props.tableName
      ) as TableMetadata;
    });

    const gotoSQLEditor = () => {
      const url = `/sql-editor/${connectionSlug(
        database.value.instance,
        database.value
      )}`;
      window.open(url);
    };

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
      gotoSQLEditor,
      bytesToString,
      isGhostTable,
      sensitiveDataList,
    };
  },
});
</script>
