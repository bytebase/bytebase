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
        <BBTableHeaderCell class="w-24" :title="columnList[1].title" />
        <BBTableHeaderCell class="w-4" :title="columnList[2].title" />
        <BBTableHeaderCell class="w-8" :title="columnList[3].title" />
        <BBTableHeaderCell class="w-8" :title="columnList[4].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[5].title" />
        <BBTableHeaderCell class="w-16" :title="columnList[6].title" />
      </template>
      <template v-slot:body="{ rowData: dataSource }">
        <BBTableCell :leftPadding="4" class="text-gray-500">
          <span class="">{{ dataSource.name }}</span>
        </BBTableCell>
        <BBTableCell>
          {{ dataSource.database ? dataSource.database.name : "*" }}
        </BBTableCell>
        <BBTableCell>
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
import { BBTableColumn } from "../bbkit/types";

import { dataSourceSlug, instanceSlug } from "../utils";
import { Instance } from "../types";

const columnList: BBTableColumn[] = [
  {
    title: "Name",
  },
  {
    title: "Database",
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
      const readWriteList = [];
      const readOnlyList = [];
      for (const item of store.getters["dataSource/dataSourceListByInstanceId"](
        props.instance.id
      )) {
        if (
          !state.searchText ||
          item.name.toLowerCase().includes(state.searchText.toLowerCase()) ||
          item.database?.name
            .toLowerCase()
            .includes(state.searchText.toLowerCase())
        ) {
          if (item.type === "RW") {
            readWriteList.push(item);
          } else if (item.type === "RO") {
            readOnlyList.push(item);
          }
        }
      }
      const dataSource = [];
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
