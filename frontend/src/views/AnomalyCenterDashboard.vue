<template>
  <div class="flex flex-col px-4">
    <BBAttention
      :style="'INFO'"
      :title="'Anomaly detection'"
      :description="'Bytebase periodically scans the managed resources and list the detected anomolies here. The list is refreshed roughly every 30 minutes.'"
    />
    <div class="py-2 flex justify-between items-center">
      <BBTabFilter
        :tabItemList="tabItemList"
        :selectedIndex="state.selectedIndex"
        @select-index="
          (index) => {
            state.selectedIndex = index;
          }
        "
      />
      <BBTableSearch
        class="w-72"
        ref="searchField"
        :placeholder="
          state.selectedIndex == DATABASE_TAB
            ? 'Search database or environment'
            : 'Search instance or environment'
        "
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <AnomalyTable
      v-if="state.selectedIndex == DATABASE_TAB"
      :mode="'DATABASE'"
      :anomalySectionList="databaseAnomalySectionList"
      :multiple="true"
    />
    <AnomalyTable
      v-if="state.selectedIndex == INSTANCE_TAB"
      :mode="'INSTANCE'"
      :anomalySectionList="instanceAnomalySectionList"
      :multiple="true"
    />
  </div>
</template>

<script lang="ts">
import { computed, reactive, watchEffect } from "vue-demi";
import { useStore } from "vuex";
import AnomalyTable from "../components/AnomalyTable.vue";
import { Anomaly, Database } from "../types";
import {
  databaseSlug,
  instanceSlug,
  isDBAOrOwner,
  sortDatabaseList,
  sortInstanceList,
} from "../utils";
import { BBTabFilterItem, BBTableSectionDataSource } from "../bbkit/types";
import { cloneDeep } from "lodash";

const DATABASE_TAB = 0;
const INSTANCE_TAB = 1;

interface LocalState {
  selectedIndex: number;
  searchText: string;
}

export default {
  name: "AnomalyCenterDashboard",
  components: { AnomalyTable },
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const state = reactive<LocalState>({
      selectedIndex: isDBAOrOwner(currentUser.value.role)
        ? INSTANCE_TAB
        : DATABASE_TAB,
      searchText: "",
    });

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"](["NORMAL"]);
    });

    const prepareDatabaseList = () => {
      // It will also be called when user logout
      store.dispatch("database/fetchDatabaseList");
    };

    watchEffect(prepareDatabaseList);

    const databaseList = computed((): Database[] => {
      return store.getters["database/databaseListByPrincipalId"](
        currentUser.value.id
      );
    });

    const prepareInstanceList = () => {
      store.dispatch("instance/fetchInstanceList");
    };

    watchEffect(prepareInstanceList);

    const instanceList = computed(() => {
      return store.getters["instance/instanceList"]();
    });

    const databaseAnomalySectionList = computed(
      (): BBTableSectionDataSource<Anomaly>[] => {
        const sectionList: BBTableSectionDataSource<Anomaly>[] = [];
        const dbList = state.searchText ? [] : cloneDeep(databaseList.value);
        if (state.searchText) {
          for (const database of databaseList.value) {
            if (
              database.name
                .toLowerCase()
                .includes(state.searchText.toLowerCase()) ||
              database.instance.environment.name
                .toLowerCase()
                .includes(state.searchText.toLowerCase())
            ) {
              dbList.push(database);
            }
          }
        }

        sortDatabaseList(dbList, environmentList.value);

        for (const database of dbList) {
          if (database.anomalyList.length > 0) {
            sectionList.push({
              title: `${database.name} (${database.instance.environment.name})`,
              link: `/db/${databaseSlug(database)}`,
              list: database.anomalyList,
            });
          }
        }

        return sectionList;
      }
    );

    const instanceAnomalySectionList = computed(
      (): BBTableSectionDataSource<Anomaly>[] => {
        const sectionList: BBTableSectionDataSource<Anomaly>[] = [];
        const insList = state.searchText ? [] : cloneDeep(instanceList.value);
        if (state.searchText) {
          for (const instance of instanceList.value) {
            if (
              instance.name
                .toLowerCase()
                .includes(state.searchText.toLowerCase()) ||
              instance.environment.name
                .toLowerCase()
                .includes(state.searchText.toLowerCase())
            ) {
              insList.push(instance);
            }
          }
        }

        sortInstanceList(insList, environmentList.value);

        for (const instance of insList) {
          if (instance.anomalyList.length > 0) {
            sectionList.push({
              title: `${instance.name} (${instance.environment.name})`,
              link: `/instance/${instanceSlug(instance)}`,
              list: instance.anomalyList,
            });
          }
        }

        return sectionList;
      }
    );

    const tabItemList = computed((): BBTabFilterItem[] => {
      return [
        {
          title: "Database",
          alert: databaseAnomalySectionList.value.length > 0,
        },
        {
          title: "Instance",
          alert: instanceAnomalySectionList.value.length > 0,
        },
      ];
    });

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    return {
      DATABASE_TAB,
      INSTANCE_TAB,
      state,
      databaseAnomalySectionList,
      instanceAnomalySectionList,
      tabItemList,
      changeSearchText,
    };
  },
};
</script>
