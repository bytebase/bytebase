<template>
  <div class="flex flex-col space-y-2">
    <div class="flex justify-between items-center">
      <h3 class="text-lg leading-6 font-medium text-gray-900">
        All Data Source
      </h3>
      <BBTableSearch
        class="w-56"
        ref="searchField"
        :placeholder="'Search name, database'"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <BBTable
      :columnList="columnList"
      :sectionDataSource="dataSourceSectionList"
      :showHeader="true"
      @click-row="clickDataSource"
    >
      <template v-slot:header>
        <BBTableHeaderCell
          :leftPadding="4"
          class="w-24"
          :title="columnList[0].title"
        />
        <BBTableHeaderCell class="w-4" :title="columnList[1].title" />
        <BBTableHeaderCell class="w-8" :title="columnList[2].title" />
        <BBTableHeaderCell class="w-8" :title="columnList[3].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[4].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[5].title" />
      </template>
      <template v-slot:body="{ rowData: dataSource }">
        <BBTableCell :leftPadding="4">
          <span class="">{{ dataSource.name }}</span>
        </BBTableCell>
        <BBTableCell v-data-source-type>
          {{ dataSource.type }}
        </BBTableCell>
        <BBTableCell>
          {{ dataSource.username }}
        </BBTableCell>
        <BBTableCell>
          {{ dataSource.password }}
        </BBTableCell>
        <BBTableCell>
          {{ humanizeTs(dataSource.lastUpdatedTs) }}
        </BBTableCell>
        <BBTableCell>
          {{ humanizeTs(dataSource.createdTs) }}
        </BBTableCell>
      </template>
    </BBTable>
  </div>
</template>

<script lang="ts">
import { computed, reactive, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";

import { dataSourceSlug, instanceSlug } from "../utils";
import { Instance, Database, DataSource } from "../types";

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
    title: "Last updated",
  },
  {
    title: "Created",
  },
];

interface LocalState {
  searchText: string;
}

export default {
  name: "DataSourceTable",
  components: {},
  props: {
    instance: {
      required: true,
      type: Object as PropType<Instance>,
    },
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      searchText: "",
    });

    const dataSourceSectionList = computed(() => {
      const databaseList = store.getters["database/databaseListByInstanceId"](
        props.instance.id
      );
      const dataSourceListByDatabase: Map<string, DataSource[]> = new Map();
      for (const dataSource of store.getters[
        "dataSource/dataSourceListByInstanceId"
      ](props.instance.id)) {
        let databaseName = "*";
        if (dataSource.databaseId) {
          databaseName = store.getters["database/databaseById"](
            dataSource.databaseId,
            props.instance.id
          ).name;
        }
        if (
          !state.searchText ||
          dataSource.name
            .toLowerCase()
            .includes(state.searchText.toLowerCase()) ||
          databaseName.toLowerCase().includes(state.searchText.toLowerCase())
        ) {
          const list = dataSourceListByDatabase.get(databaseName);
          if (list) {
            list.push(dataSource);
          } else {
            dataSourceListByDatabase.set(databaseName, [dataSource]);
          }
        }
      }

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

      const sectionList = dataSourceListByDatabase.get("*")
        ? [
            {
              title: "* (All databases)",
              list: dataSourceListByDatabase.get("*"),
            },
          ]
        : [];

      for (const database of databaseList) {
        if (dataSourceListByDatabase.get(database.name)) {
          sectionList.push({
            title: database.name,
            list: dataSourceListByDatabase.get(database.name),
          });
        }
      }

      return sectionList;
    });

    const clickDataSource = function (section: number, row: number) {
      const ds = dataSourceSectionList.value[section].list![row];
      const environmentName = props.instance.environment.name;
      router.push(
        `/instance/${instanceSlug(props.instance)}/ds/${dataSourceSlug(
          ds.name,
          ds.id
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
      clickDataSource,
      changeSearchText,
    };
  },
};
</script>
