<template>
  <div class="flex flex-col space-y-2">
    <div class="flex justify-between items-center">
      <div class="inline-flex items-center space-x-2">
        <h3 class="text-lg leading-6 font-medium text-gray-900">
          {{
            database
              ? $t("datasource.data-source-list")
              : $t("datasource.all-data-source")
          }}
        </h3>
        <!-- Hide add button for now, as we don't allow adding new data source after creating the database. -->
        <BBButtonAdd v-if="false" @add="state.showCreateModal = true" />
      </div>
      <div class="flex flex-row items-center space-x-2">
        <button
          class="btn-icon"
          @click.prevent="state.showPassword = !state.showPassword"
        >
          <heroicons-outline:eye-off
            v-if="state.showPassword"
            class="w-6 h-6"
          />
          <heroicons-outline:eye v-else class="w-6 h-6" />
        </button>
        <BBTableSearch
          ref="searchField"
          class="w-56"
          :placeholder="
            database
              ? $t('datasource.search-name')
              : $t('datasource.search-name-database')
          "
          @change-text="(text) => changeSearchText(text)"
        />
      </div>
    </div>
    <BBTable
      :column-list="columnList"
      :section-data-source="dataSourceSectionList"
      :show-header="true"
      :compact-section="database != undefined"
      @click-row="clickDataSource"
    >
      <template #header>
        <BBTableHeaderCell
          :left-padding="4"
          class="w-24"
          :title="columnList[0].title"
        />
        <BBTableHeaderCell class="w-4" :title="columnList[1].title" />
        <BBTableHeaderCell class="w-8" :title="columnList[2].title" />
        <BBTableHeaderCell class="w-8" :title="columnList[3].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[4].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[5].title" />
      </template>
      <template #body="{ rowData: dataSource }">
        <BBTableCell :left-padding="4">
          <span class="">{{ dataSource.name }}</span>
        </BBTableCell>
        <BBTableCell v-data-source-type>
          {{ dataSource.type }}
        </BBTableCell>
        <BBTableCell>
          {{ dataSource.username }}
        </BBTableCell>
        <BBTableCell>
          {{ state.showPassword ? dataSource.password : "******" }}
        </BBTableCell>
        <BBTableCell>
          {{ humanizeTs(dataSource.updatedTs) }}
        </BBTableCell>
        <BBTableCell>
          {{ humanizeTs(dataSource.createdTs) }}
        </BBTableCell>
      </template>
    </BBTable>
  </div>
  <BBModal
    v-if="state.showCreateModal"
    :title="$t('datasource.create-data-source')"
    @close="state.showCreateModal = false"
  >
    <DataSourceCreateForm
      :instance-i-d="instance.id"
      :database="database"
      @create="doCreate"
      @cancel="state.showCreateModal = false"
    />
  </BBModal>
</template>

<script lang="ts">
import { computed, reactive, PropType, defineComponent } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import DataSourceCreateForm from "../components/DataSourceCreateForm.vue";
import { BBTableColumn } from "../bbkit/types";

import { databaseSlug, dataSourceSlug } from "../utils";
import { Instance, Database, DataSource, DataSourceCreate } from "../types";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useDatabaseStore,
  useDataSourceStore,
} from "@/store";

interface LocalState {
  searchText: string;
  showPassword: boolean;
  showCreateModal: boolean;
}

export default defineComponent({
  name: "DataSourceTable",
  components: { DataSourceCreateForm },
  props: {
    instance: {
      required: true,
      type: Object as PropType<Instance>,
    },
    // If database is specified, then we just list the data source for that database.
    database: {
      type: Object as PropType<Database>,
    },
  },
  setup(props) {
    const store = useStore();
    const router = useRouter();
    const { t } = useI18n();
    const dataSourceStore = useDataSourceStore();

    const columnList: BBTableColumn[] = [
      {
        title: t("common.name"),
      },
      {
        title: t("common.type"),
      },
      {
        title: t("common.username"),
      },
      {
        title: t("common.password"),
      },
      {
        title: t("common.updated-at"),
      },
      {
        title: t("common.created-at"),
      },
    ];

    const state = reactive<LocalState>({
      searchText: "",
      showPassword: false,
      showCreateModal: false,
    });

    const dataSourceSectionList = computed(() => {
      const databaseList = props.database
        ? [props.database]
        : useDatabaseStore().getDatabaseListByInstanceId(props.instance.id);
      const dataSourceListByDatabase: Map<string, DataSource[]> = new Map();
      databaseList.forEach((database: Database) => {
        for (const dataSource of database.dataSourceList) {
          if (
            !state.searchText ||
            dataSource.name
              .toLowerCase()
              .includes(state.searchText.toLowerCase()) ||
            database.name.toLowerCase().includes(state.searchText.toLowerCase())
          ) {
            const list = dataSourceListByDatabase.get(database.name);
            if (list) {
              list.push(dataSource);
            } else {
              dataSourceListByDatabase.set(database.name, [dataSource]);
            }
          }
        }
      });

      dataSourceListByDatabase.forEach((list) =>
        list.sort((a: DataSource, b: DataSource) => {
          if (a.type == b.type) {
            return a.name.localeCompare(b.name, undefined, {
              sensitivity: "base",
            });
          }
          if (a.type == "RW") {
            return -1;
          }
          return 1;
        })
      );

      const sectionList = [];

      for (const database of databaseList) {
        if (dataSourceListByDatabase.get(database.name)) {
          sectionList.push({
            title: database.name,
            link: `/db/${databaseSlug(database)}`,
            list: dataSourceListByDatabase.get(database.name),
          });
        }
      }

      return sectionList;
    });

    const doCreate = (newDataSource: DataSourceCreate) => {
      dataSourceStore.createDataSource(newDataSource).then((dataSource) => {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t(
            "datasource.successfully-created-data-source-datasource-name",
            [dataSource.name]
          ),
        });
        router.push(
          `/db/${databaseSlug(
            dataSource.database
          )}/data-source/${dataSourceSlug(dataSource)}`
        );
      });
    };

    const clickDataSource = function (section: number, row: number) {
      const dataSource = dataSourceSectionList.value[section].list![row];
      router.push(
        `/db/${databaseSlug(dataSource.database)}/data-source/${dataSourceSlug(
          dataSource
        )}`
      );
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    return {
      state,
      columnList,
      dataSourceSectionList,
      doCreate,
      clickDataSource,
      changeSearchText,
    };
  },
});
</script>
