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
                ? $t("db.encoding")
                : $t("db.character-set")
            }}
          </dt>
          <dd class="mt-1 text-sm text-main">{{ database.characterSet }}</dd>
        </div>

        <div class="col-span-1">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("db.collation") }}
          </dt>
          <dd class="mt-1 text-sm text-main">{{ database.collation }}</dd>
        </div>
      </template>

      <div class="col-span-1 col-start-1">
        <dt class="text-sm font-medium text-control-light">
          {{ $t("database.sync-status") }}
        </dt>
        <dd class="mt-1 text-sm text-main">
          <span>{{ database.syncStatus }}</span>
        </dd>
      </div>

      <div class="col-span-1">
        <dt class="text-sm font-medium text-control-light">
          {{ $t("database.last-successful-sync") }}
        </dt>
        <dd class="mt-1 text-sm text-main">
          {{ humanizeTs(database.lastSuccessfulSyncTs) }}
        </dd>
      </div>

      <div class="col-span-1 col-start-1">
        <dt class="text-sm font-medium text-control-light">
          {{ $t("common.created-at") }}
        </dt>
        <dd class="mt-1 text-sm text-main">
          {{ humanizeTs(database.createdTs) }}
        </dd>
      </div>

      <div class="col-span-1">
        <dt class="text-sm font-medium text-control-light">
          {{ $t("common.updated-at") }}
        </dt>
        <dd class="mt-1 text-sm text-main">
          {{ humanizeTs(database.updatedTs) }}
        </dd>
      </div>
    </dl>

    <div class="pt-6">
      <div class="text-lg leading-6 font-medium text-main mb-4">
        {{ $t("db.tables") }}
      </div>
      <TableTable :table-list="tableList" />

      <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
        {{ $t("db.views") }}
      </div>
      <ViewTable :view-list="viewList" />

      <template v-if="database.instance.engine == 'POSTGRES' && isDev">
        <div class="mt-6 text-lg leading-6 font-medium text-main mb-4">
          {{ $t("db.extensions") }}
        </div>
        <DBExtensionTable :db-extension-list="dbExtensionList" />
      </template>
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
            <div v-for="(ds, dsIndex) of item.list" :key="dsIndex">
              <div v-if="hasDataSourceFeature" class="relative mb-2">
                <div
                  class="absolute inset-0 flex items-center"
                  aria-hidden="true"
                >
                  <div class="w-full border-t border-gray-300"></div>
                </div>
                <div class="relative flex justify-start">
                  <router-link
                    :to="`/db/${databaseSlug}/data-source/${dataSourceSlug(
                      ds
                    )}`"
                    class="pr-3 bg-white font-medium normal-link"
                    >{{ ds.name }}</router-link
                  >
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
                    {{ $t("common.cancel") }}
                  </button>
                  <button
                    type="button"
                    class="btn-normal"
                    :disabled="!allowSaveDataSource"
                    @click.prevent="saveEditDataSource"
                  >
                    <!-- Heroicon name: solid/save -->
                    <heroicons-solid:save
                      class="-ml-1 mr-2 h-5 w-5 text-control-light"
                    />
                    <span>{{ $t("common.save") }}</span>
                  </button>
                </template>
                <template v-else>
                  <button
                    type="button"
                    class="btn-normal"
                    @click.prevent="editDataSource(ds)"
                  >
                    <!-- Heroicon name: solid/pencil -->
                    <heroicons-solid:pencil
                      class="-ml-1 mr-2 h-5 w-5 text-control-light"
                    />
                    <span>{{ $t("common.edit") }}</span>
                  </button>
                </template>
              </div>
              <DataSourceConnectionPanel
                :editing="isEditingDataSource(ds)"
                :data-source="
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
import {
  computed,
  reactive,
  watchEffect,
  PropType,
  defineComponent,
} from "vue";
import { useRouter } from "vue-router";
import AnomalyTable from "../components/AnomalyTable.vue";
import DataSourceTable from "../components/DataSourceTable.vue";
import DataSourceConnectionPanel from "../components/DataSourceConnectionPanel.vue";
import TableTable from "../components/TableTable.vue";
import ViewTable from "../components/ViewTable.vue";
import { timezoneString, instanceSlug, isDBAOrOwner } from "../utils";
import { Anomaly, Database, DataSource, DataSourcePatch } from "../types";
import { cloneDeep, isEqual } from "lodash-es";
import { BBTableSectionDataSource } from "../bbkit/types";
import {
  featureToRef,
  useCurrentUser,
  useDataSourceStore,
  useTableStore,
  useViewStore,
  useDBExtensionStore,
} from "@/store";

interface LocalState {
  editingDataSource?: DataSource;
}

export default defineComponent({
  name: "DatabaseOverviewPanel",
  components: {
    AnomalyTable,
    DataSourceConnectionPanel,
    DataSourceTable,
    TableTable,
    ViewTable,
  },
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
  },
  setup(props) {
    const router = useRouter();
    const dataSourceStore = useDataSourceStore();

    const state = reactive<LocalState>({});

    const currentUser = useCurrentUser();
    const tableStore = useTableStore();
    const viewStore = useViewStore();
    const dbExtensionStore = useDBExtensionStore();

    const prepareTableList = () => {
      tableStore.fetchTableListByDatabaseId(props.database.id);
    };

    watchEffect(prepareTableList);

    const prepareViewList = () => {
      viewStore.fetchViewListByDatabaseId(props.database.id);
    };

    watchEffect(prepareViewList);

    const prepareDBExtensionList = () => {
      dbExtensionStore.fetchdbExtensionListByDatabaseId(props.database.id);
    };

    watchEffect(prepareDBExtensionList);

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

    const hasDataSourceFeature = featureToRef("bb.feature.data-source");

    const tableList = computed(() => {
      return tableStore.getTableListByDatabaseId(props.database.id);
    });

    const viewList = computed(() => {
      return viewStore.getViewListByDatabaseId(props.database.id);
    });

    const dbExtensionList = computed(() => {
      return dbExtensionStore.getDBExtensionListByDatabaseId(props.database.id);
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
      dataSourceStore
        .patchDataSource({
          databaseId: state.editingDataSource?.databaseId as number,
          dataSourceId: state.editingDataSource?.id as number,
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
      dbExtensionList,
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
});
</script>
