<template>
  <div class="mt-6">
    <div
      class="max-w-6xl mx-auto px-6 space-y-6 divide-y divide-block-border"
    >
      <!-- Description list -->
      <dl class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-2">
        <div class="col-span-1 col-start-1">
          <dt class="text-sm font-medium text-control-light">
            Character set
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

        <div class="col-span-1 col-start-1">
          <dt class="text-sm font-medium text-control-light">
            Sync status
          </dt>
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
          <dt class="text-sm font-medium text-control-light">Updated</dt>
          <dd class="mt-1 text-sm text-main">
            {{ humanizeTs(database.updatedTs) }}
          </dd>
        </div>

        <div class="col-span-1">
          <dt class="text-sm font-medium text-control-light">Created</dt>
          <dd class="mt-1 text-sm text-main">
            {{ humanizeTs(database.createdTs) }}
          </dd>
        </div>
      </dl>

      <div class="pt-6">
        <div class="text-lg leading-6 font-medium text-main mb-4">
          Tables
        </div>
        <TableTable
          :mode="'TABLE'"
          :tableList="tableList.filter((item) => item.type == 'BASE TABLE')"
        />

        <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
          Views
        </div>
        <TableTable
          :mode="'VIEW'"
          :tableList="tableList.filter((item) => item.type == 'VIEW')"
        />
        <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
          Migration History
        </div>
        <MigrationHistoryTable
          v-if="state.migrationSetupStatus == 'OK'"
          :historyList="migrationHistoryList"
        />
        <BBAttention
          v-else
          :title="attentionTitle"
          :actionText="'Config instance'"
          @click-action="configInstance"
        />
      </div>

      <!-- Hide data source list for now, as we don't allow adding new data source after creating the database. -->
      <div v-if="false" class="pt-6">
        <DataSourceTable
          :instance="database.instance"
          :database="database"
        />
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
                      :to="`/db/${databaseSlug}/datasource/${dataSourceSlug(
                        ds
                      )}`"
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
  </div>
</template>

<script lang="ts">
import { computed, reactive, watchEffect, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import DataSourceTable from "../components/DataSourceTable.vue";
import DataSourceConnectionPanel from "../components/DataSourceConnectionPanel.vue";
import TableTable from "../components/TableTable.vue";
import MigrationHistoryTable from "../components/MigrationHistoryTable.vue";
import {
  idFromSlug,
  instanceSlug,
  isDBAOrOwner,
} from "../utils";
import {
  Database,
  DataSource,
  DataSourcePatch,
  MigrationSchemaStatus,
  InstanceMigration,
} from "../types";
import { cloneDeep, isEmpty, isEqual } from "lodash";

interface LocalState {
  editingDataSource?: DataSource;
  migrationSetupStatus: MigrationSchemaStatus;
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
    DataSourceConnectionPanel,
    DataSourceTable,
    TableTable,
    MigrationHistoryTable,
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      migrationSetupStatus: "OK",
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const database = props.database;

    const prepareTableList = () => {
      store.dispatch(
        "table/fetchTableListByDatabaseId",
        database.id
      );
    };

    watchEffect(prepareTableList);

    const prepareMigrationHistoryList = () => {
      store
        .dispatch("instance/checkMigrationSetup", database.instance.id)
        .then((migration: InstanceMigration) => {
          state.migrationSetupStatus = migration.status;
          if (state.migrationSetupStatus == "OK") {
            store.dispatch("instance/migrationHistory", {
              instanceId: database.instance.id,
              databaseName: database.name,
            });
          }
        });
    };

    watchEffect(prepareMigrationHistoryList);

    const hasDataSourceFeature = computed(() =>
      store.getters["plan/feature"]("bb.data-source")
    );

    const tableList = computed(() => {
      return store.getters["table/tableListByDatabaseId"](
        database.id
      );
    });

    const attentionTitle = computed((): string => {
      if (state.migrationSetupStatus == "NOT_EXIST") {
        return `Missing migration history schema on instance "${database.instance.name}"`;
      } else if (state.migrationSetupStatus == "UNKNOWN") {
        return `Unable to connect instance "${database.instance.name}" to retrieve migration history`;
      }
      return "";
    });

    const migrationHistoryList = computed(() => {
      return store.getters[
        "instance/migrationHistoryListByInstanceIdAndDatabaseName"
      ](database.instance.id, database.name);
    });

    const isCurrentUserDBAOrOwner = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const allowViewDataSource = computed(() => {
      if (isCurrentUserDBAOrOwner.value) {
        return true;
      }

      for (const member of database.project.memberList) {
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
      return database.dataSourceList;
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
          databaseId: state.editingDataSource!.database.id,
          dataSourceId: state.editingDataSource!.id,
          dataSource: dataSourcePatch,
        })
        .then(() => {
          state.editingDataSource = undefined;
        });
    };

    const configInstance = () => {
      router.push(`/instance/${instanceSlug(database.instance)}`);
    };

    return {
      state,
      database,
      tableList,
      attentionTitle,
      migrationHistoryList,
      hasDataSourceFeature,
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
