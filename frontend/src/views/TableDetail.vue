<template>
  <div class="flex-1 overflow-auto focus:outline-none" tabindex="0">
    <main class="flex-1 relative pb-8 overflow-y-auto">
      <!-- Highlight Panel -->
      <div
        class="
          px-4
          pb-4
          border-b border-block-border
          md:flex md:items-center md:justify-between
        "
      >
        <div class="flex-1 min-w-0">
          <!-- Summary -->
          <div class="flex items-center">
            <div>
              <div class="flex items-center">
                <h1
                  class="
                    pt-2
                    pb-2.5
                    text-xl
                    font-bold
                    leading-6
                    text-main
                    truncate
                  "
                >
                  {{ table.name }}
                </h1>
              </div>
            </div>
          </div>
          <dl
            class="
              flex flex-col
              space-y-1
              md:space-y-0 md:flex-row md:flex-wrap
            "
          >
            <dt class="sr-only">Environment</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Environment&nbsp;-&nbsp;</span>
              <router-link
                :to="`/environment/${environmentSlug(
                  database.instance.environment
                )}`"
                class="normal-link"
              >
                {{ environmentName(database.instance.environment) }}
              </router-link>
            </dd>
            <dt class="sr-only">Instance</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <InstanceEngineIcon :instance="database.instance" />
              <span class="ml-1 textlabel">Instance&nbsp;-&nbsp;</span>
              <router-link
                :to="`/instance/${instanceSlug(database.instance)}`"
                class="normal-link"
              >
                {{ instanceName(database.instance) }}
              </router-link>
            </dd>
            <dt class="sr-only">Project</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Project&nbsp;-&nbsp;</span>
              <router-link
                :to="`/project/${projectSlug(database.project)}`"
                class="normal-link"
              >
                {{ projectName(database.project) }}
              </router-link>
            </dd>
            <dt class="sr-only">Database</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Database&nbsp;-&nbsp;</span>
              <router-link :to="`/db/${databaseSlug}`" class="normal-link">
                {{ database.name }}
              </router-link>

              <span class="ml-2 textlabel">SQL Console</span>
              <button
                class="ml-1 btn-icon"
                @click.prevent="
                  window.open(urlfy(databaseConsoleLink), '_blank')
                "
              >
                <svg
                  class="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                  ></path>
                </svg>
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
              <dt class="text-sm font-medium text-control-light">Engine</dt>
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
                Row count estimate
              </dt>
              <dd class="mt-1 text-sm text-main">
                {{ table.rowCount }}
              </dd>
            </div>

            <div class="col-span-1 col-start-1">
              <dt class="text-sm font-medium text-control-light">Data size</dt>
              <dd class="mt-1 text-sm text-main">
                {{ bytesToString(table.dataSize) }}
              </dd>
            </div>

            <div class="col-span-1">
              <dt class="text-sm font-medium text-control-light">Index size</dt>
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
                      ? "Encoding"
                      : "Character set"
                  }}
                </dt>
                <dd class="mt-1 text-sm text-main">
                  {{ database.characterSet }}
                </dd>
              </div>

              <div class="col-span-1">
                <dt class="text-sm font-medium text-control-light">
                  Collation
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

            <div class="col-span-1 col-start-1">
              <dt class="text-sm font-medium text-control-light">Updated</dt>
              <dd class="mt-1 text-sm text-main">
                {{ humanizeTs(table.updatedTs) }}
              </dd>
            </div>

            <div class="col-span-1">
              <dt class="text-sm font-medium text-control-light">Created</dt>
              <dd class="mt-1 text-sm text-main">
                {{ humanizeTs(table.createdTs) }}
              </dd>
            </div>
          </dl>
        </div>
      </div>

      <div class="mt-6 px-6">
        <div class="text-lg leading-6 font-medium text-main mb-4">Columns</div>
        <ColumnTable
          :column-list="table.columnList"
          :engine="database.instance.engine"
        />
      </div>

      <div v-if="database.instance.engine != 'SNOWFLAKE'" class="mt-6 px-6">
        <div class="text-lg leading-6 font-medium text-main mb-4">Indexes</div>
        <IndexTable :index-list="table.indexList" />
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed } from "@vue/runtime-core";
import { useStore } from "vuex";
import ColumnTable from "../components/ColumnTable.vue";
import IndexTable from "../components/IndexTable.vue";
import InstanceEngineIcon from "../components/InstanceEngineIcon.vue";
import { bytesToString, consoleLink, idFromSlug } from "../utils";
import { isEmpty } from "lodash";

export default {
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
    const store = useStore();

    const table = computed(() => {
      return store.getters["table/tableListByDatabaseIdAndTableName"](
        idFromSlug(props.databaseSlug),
        props.tableName
      );
    });

    const database = computed(() => {
      return table.value.database;
    });

    const databaseConsoleLink = computed(() => {
      const consoleURL =
        store.getters["setting/settingByName"]("bb.console.url").value;
      if (!isEmpty(consoleURL)) {
        return consoleLink(consoleURL, database.value.name);
      }
      return "";
    });

    return {
      table,
      database,
      databaseConsoleLink,
      bytesToString,
    };
  },
};
</script>
