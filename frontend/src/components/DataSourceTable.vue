<template>
  <div class="flex flex-col space-y-2">
    <div class="flex justify-between items-center">
      <h3 class="text-lg leading-6 font-medium text-gray-900">
        All Data Source
      </h3>
      <BBTableSearch
        class="w-56"
        ref="searchField"
        :placeholder="'Search data source name'"
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
          class="w-4 table-cell"
          :title="columnList[0].title"
        />
        <BBTableHeaderCell
          class="w-4 table-cell"
          :title="columnList[1].title"
        />
        <BBTableHeaderCell
          class="w-4 table-cell"
          :title="columnList[2].title"
        />
        <BBTableHeaderCell
          class="w-24 table-cell"
          :title="columnList[3].title"
        />
        <BBTableHeaderCell
          class="w-4 table-cell"
          :title="columnList[4].title"
        />
        <BBTableHeaderCell
          class="w-4 table-cell"
          :title="columnList[5].title"
        />
      </template>
      <template v-slot:body="{ rowData: dataSource }">
        <BBTableCell :leftPadding="4" class="w-4 table-cell text-gray-500">
          <span class="">{{ dataSource.name }}</span>
        </BBTableCell>
        <BBTableCell class="w-24 table-cell">
          {{ dataSource.type }}
        </BBTableCell>
        <BBTableCell class="w-24 table-cell">
          {{ dataSource.username }}
        </BBTableCell>
        <BBTableCell class="w-24 table-cell">
          {{ dataSource.password }}
        </BBTableCell>
        <BBTableCell class="w-24 table-cell">
          {{ humanizeTs(dataSource.lastUpdatedTs) }}
        </BBTableCell>
        <BBTableCell class="w-24 table-cell">
          {{ humanizeTs(dataSource.createdTs) }}
        </BBTableCell>
      </template>
    </BBTable>
  </div>
</template>

<script lang="ts">
import { computed, reactive, watchEffect, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";

import { idFromSlug, dataSourceSlug, instanceSlug } from "../utils";
import { Instance, DataSource } from "../types";

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

    const prepareDataSourceList = () => {
      store
        .dispatch(
          "dataSource/fetchDataSourceListByInstanceId",
          props.instance.id
        )
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareDataSourceList);

    const dataSourceSectionList = computed(() => {
      const adminList = [];
      const readWriteList = [];
      const readOnlyList = [];
      for (const item of store.getters["dataSource/dataSourceListByInstanceId"](
        props.instance.id
      )) {
        if (
          !state.searchText ||
          item.name.toLowerCase().includes(state.searchText.toLowerCase())
        ) {
          if (item.type === "ADMIN") {
            adminList.push(item);
          } else if (item.type === "READWRITE") {
            readWriteList.push(item);
          } else if (item.type === "READONLY") {
            readOnlyList.push(item);
          }
        }
      }
      const dataSource = [];
      dataSource.push({
        title: "Admin",
        list: adminList,
      });
      dataSource.push({
        title: "Read and write",
        list: readWriteList,
      });
      dataSource.push({
        title: "Read only",
        list: readOnlyList,
      });
      return dataSource;
    });

    const clickDataSource = function (section: number, row: number) {
      const ds = dataSourceSectionList.value[section].list[row];
      const environmentName = store.getters["environment/environmentById"](
        props.instance.environmentId
      )?.name;
      router.push(
        `/instance/${instanceSlug(
          environmentName,
          props.instance.name,
          props.instance.id
        )}/ds/${dataSourceSlug(ds.name, ds.id)}`
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
