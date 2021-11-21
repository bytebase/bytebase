<template>
  <div class="space-y-6 divide-y divide-block-border">
    <div v-if="anomalySectionList.length > 0">
      <div class="text-lg leading-6 font-medium text-main mb-4 flex flex-row">
        Anomalies
        <span class="ml-2 textinfolabel items-center flex"
          >The list might be out of date and is refreshed roughly every 10
          minutes</span
        >
      </div>
      <AnomalyTable :anomalySectionList="anomalySectionList" />
    </div>
    <div
      v-else
      class="text-lg leading-6 font-medium text-main mb-4 flex flex-row"
    >
      No anomalies detected
      <svg
        class="ml-1 w-6 h-6 text-success"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
        ></path>
      </svg>
    </div>

    <!-- Description list -->
    <dl class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-2 pt-4">
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
          <dt class="text-sm font-medium text-control-light">Collation</dt>
          <dd class="mt-1 text-sm text-main">
            {{ database.collation }}
          </dd>
        </div>
      </template>

      <div class="col-span-1 col-start-1">
        <dt class="text-sm font-medium text-control-light">Sync status</dt>
        <dd class="mt-1 text-sm text-main">
          <span>{{ database.syncStatus }}</span>
        </dd>
      </div>

      <div class="col-span-1">
        <dt class="text-sm font-medium text-control-light">
          Last successful sync
        </dt>
        <dd class="mt-1 text-sm text-main">
          {{ humanizeTs(database.lastSuccessfulSyncTs) }}
        </dd>
      </div>

      <div class="col-span-1 col-start-1">
        <dt class="text-sm font-medium text-control-light">Created</dt>
        <dd class="mt-1 text-sm text-main">
          {{ humanizeTs(database.createdTs) }}
        </dd>
      </div>

      <div class="col-span-1">
        <dt class="text-sm font-medium text-control-light">Updated</dt>
        <dd class="mt-1 text-sm text-main">
          {{ humanizeTs(database.updatedTs) }}
        </dd>
      </div>
    </dl>

    <div class="pt-6">
      <div class="text-lg leading-6 font-medium text-main mb-4">Tables</div>
      <TableTable :tableList="tableList" />

      <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">Views</div>
      <ViewTable :viewList="viewList" />
    </div>

    <!-- Hide data source list for now, as we don't allow adding new data source after creating the database. -->
    <div v-if="false" class="pt-6">
      <DataSourceTable :instance="database.instance" :database="database" />
    </div>

    <template v-if="allowViewDataSource">
      <template
        v-for="(item, index) of [
          { type: 'RW', list: readWriteDataSourceList },
          { type: 'RO', list: readonlyDataSourceList },
        ]"
        :key="index"
      >
        <div v-if="item.list.length" class="pt-6">
          <div
            v-if="hasDataSourceFeature"
            class="text-lg leading-6 font-medium text-main mb-4"
          >
            <span v-data-source-type>{{ item.type }}</span>
          </div>
          <div class="space-y-4">
            <div v-for="(ds, index) of item.list" :key="index">
              <div v-if="hasDataSourceFeature" class="relative mb-2">
                <div
                  class="absolute inset-0 flex items-center"
                  aria-hidden="true"
                >
                  <div class="w-full border-t border-gray-300"></div>
                </div>
                <div class="relative flex justify-start">
                  <router-link
                    :to="`/db/${databaseSlug}/datasource/${dataSourceSlug(ds)}`"
                    class="pr-3 bg-white font-medium normal-link"
                  >
                    {{ ds.name }}
                  </router-link>
                </div>
              </div>
              <div
                v-if="allowChangeDataSource"
                class="flex justify-end space-x-3"
              >
                <template v-if="isEditingDataSource(ds)">
                  <button
                    type="button"
                    class="btn-normal"
                    @click.prevent="cancelEditDataSource"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    class="btn-normal"
                    :disabled="!allowSaveDataSource"
                    @click.prevent="saveEditDataSource"
                  >
                    <!-- Heroicon name: solid/save -->
                    <svg
                      class="-ml-1 mr-2 h-5 w-5 text-control-light"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <path
                        d="M7.707 10.293a1 1 0 10-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L11 11.586V6h5a2 2 0 012 2v7a2 2 0 01-2 2H4a2 2 0 01-2-2V8a2 2 0 012-2h5v5.586l-1.293-1.293zM9 4a1 1 0 012 0v2H9V4z"
                      ></path>
                    </svg>
                    <span>Save</span>
                  </button>
                </template>
                <template v-else>
                  <button
                    type="button"
                    class="btn-normal"
                    @click.prevent="editDataSource(ds)"
                  >
                    <!-- Heroicon name: solid/pencil -->
                    <svg
                      class="-ml-1 mr-2 h-5 w-5 text-control-light"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <path
                        d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
                      ></path>
                    </svg>
                    <span>Edit</span>
                  </button>
                </template>
              </div>
              <DataSourceConnectionPanel
                :editing="isEditingDataSource(ds)"
                :dataSource="
                  isEditingDataSource(ds) ? state.editingDataSource : ds
                "
              />
            </div>
          </div>
        </div>
      </template>
    </template>
  </div>
</template>

<script lang="ts">
import { computed, reactive, watchEffect, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import AnomalyTable from "../components/AnomalyTable.vue";
import DataSourceTable from "../components/DataSourceTable.vue";
import DataSourceConnectionPanel from "../components/DataSourceConnectionPanel.vue";
import TableTable from "../components/TableTable.vue";
import ViewTable from "../components/ViewTable.vue";
import { timezoneString, instanceSlug, isDBAOrOwner } from "../utils";
import { Anomaly, Database, DataSource, DataSourcePatch } from "../types";
import { cloneDeep, isEqual } from "lodash";
import { BBTableSectionDataSource } from "../bbkit/types";

interface LocalState {
  editingDataSource?: DataSource;
}

export default {
  name: "DatabaseOverviewPanel",
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
  },
  components: {
    AnomalyTable,
    DataSourceConnectionPanel,
    DataSourceTable,
    TableTable,
    ViewTable,
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({});

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareTableList = () => {
      store.dispatch("table/fetchTableListByDatabaseID", props.database.id);
    };

    watchEffect(prepareTableList);

    const prepareViewList = () => {
      store.dispatch("view/fetchViewListByDatabaseID", props.database.id);
    };

    watchEffect(prepareViewList);

    const anomalySectionList = computed(
      (): BBTableSectionDataSource<Anomaly>[] => {
        const list: BBTableSectionDataSource<Anomaly>[] = [];
        if (props.database.anomalyList.length > 0) {
          list.push({
            title: props.database.name,
            list: props.database.anomalyList,
          });
        }
        return list;
      }
    );

    const hasDataSourceFeature = computed(() =>
      store.getters["plan/feature"]("bb.data-source")
    );

    const tableList = computed(() => {
      return store.getters["table/tableListByDatabaseID"](props.database.id);
    });

    const viewList = computed(() => {
      return store.getters["view/viewListByDatabaseID"](props.database.id);
    });

    const isCurrentUserDBAOrOwner = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const allowConfigInstance = computed(() => {
      return isCurrentUserDBAOrOwner.value;
    });

    const allowViewDataSource = computed(() => {
      if (isCurrentUserDBAOrOwner.value) {
        return true;
      }

      for (const member of props.database.project.memberList) {
        if (member.principal.id == currentUser.value.id) {
          return true;
        }
      }

      return false;
    });

    const allowChangeDataSource = computed(() => {
      return isCurrentUserDBAOrOwner.value;
    });

    const dataSourceList = computed(() => {
      return props.database.dataSourceList;
    });

    const readWriteDataSourceList = computed(() => {
      return dataSourceList.value.filter((dataSource: DataSource) => {
        return dataSource.type == "RW";
      });
    });

    const readonlyDataSourceList = computed(() => {
      return dataSourceList.value.filter((dataSource: DataSource) => {
        return dataSource.type == "RO";
      });
    });

    const isEditingDataSource = (dataSource: DataSource) => {
      return (
        state.editingDataSource && state.editingDataSource.id == dataSource.id
      );
    };

    const allowSaveDataSource = computed(() => {
      for (const dataSource of dataSourceList.value) {
        if (dataSource.id == state.editingDataSource!.id) {
          return !isEqual(dataSource, state.editingDataSource);
        }
      }
      return false;
    });

    const editDataSource = (dataSource: DataSource) => {
      state.editingDataSource = cloneDeep(dataSource);
    };

    const cancelEditDataSource = () => {
      state.editingDataSource = undefined;
    };

    const saveEditDataSource = () => {
      const dataSourcePatch: DataSourcePatch = {
        username: state.editingDataSource?.username,
        password: state.editingDataSource?.password,
      };
      store
        .dispatch("dataSource/patchDataSource", {
          databaseID: state.editingDataSource!.database.id,
          dataSourceID: state.editingDataSource!.id,
          dataSource: dataSourcePatch,
        })
        .then(() => {
          state.editingDataSource = undefined;
        });
    };

    const configInstance = () => {
      router.push(`/instance/${instanceSlug(props.database.instance)}`);
    };

    return {
      timezoneString,
      state,
      anomalySectionList,
      tableList,
      viewList,
      hasDataSourceFeature,
      allowConfigInstance,
      allowViewDataSource,
      allowChangeDataSource,
      readWriteDataSourceList,
      readonlyDataSourceList,
      isEditingDataSource,
      allowSaveDataSource,
      editDataSource,
      cancelEditDataSource,
      saveEditDataSource,
      configInstance,
    };
  },
};
</script>
