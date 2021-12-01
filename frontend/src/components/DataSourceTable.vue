<template>
  <div class="flex flex-col space-y-2">
    <div class="flex justify-between items-center">
      <div class="inline-flex items-center space-x-2">
        <h3 class="text-lg leading-6 font-medium text-gray-900">
          {{ database ? "Data source list" : "All data source" }}
        </h3>
        <!-- Hide add button for now, as we don't allow adding new data source after creating the database. -->
        <BBButtonAdd v-if="false" @add="state.showCreateModal = true" />
      </div>
      <div class="flex flex-row items-center space-x-2">
        <button
          class="btn-icon"
          @click.prevent="state.showPassword = !state.showPassword"
        >
          <svg
            v-if="state.showPassword"
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
            ></path>
          </svg>
          <svg
            v-else
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
            ></path>
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
            ></path>
          </svg>
        </button>
        <BBTableSearch
          ref="searchField"
          class="w-56"
          :placeholder="database ? 'Search name' : 'Search name, database'"
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
    :title="'Create data source'"
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
import { computed, reactive, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import DataSourceCreateForm from "../components/DataSourceCreateForm.vue";
import { BBTableColumn } from "../bbkit/types";

import { databaseSlug, dataSourceSlug } from "../utils";
import { Instance, Database, DataSource, DataSourceCreate } from "../types";

const columnList: BBTableColumn[] = [
  {
    title: "Name",
  },
  {
    title: "Type",
  },
  {
    title: "Username",
  },
  {
    title: "Password",
  },
  {
    title: "Updated",
  },
  {
    title: "Created",
  },
];

interface LocalState {
  searchText: string;
  showPassword: boolean;
  showCreateModal: boolean;
}

export default {
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

    const state = reactive<LocalState>({
      searchText: "",
      showPassword: false,
      showCreateModal: false,
    });

    const dataSourceSectionList = computed(() => {
      const databaseList = props.database
        ? [props.database]
        : store.getters["database/databaseListByInstanceId"](props.instance.id);
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
      store
        .dispatch("dataSource/createDataSource", newDataSource)
        .then((dataSource) => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully created data source '${dataSource.name}'.`,
          });
          router.push(
            `/db/${databaseSlug(
              dataSource.database
            )}/datasource/${dataSourceSlug(dataSource)}`
          );
        });
    };

    const clickDataSource = function (section: number, row: number) {
      const dataSource = dataSourceSectionList.value[section].list![row];
      router.push(
        `/db/${databaseSlug(dataSource.database)}/datasource/${dataSourceSlug(
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
};
</script>
