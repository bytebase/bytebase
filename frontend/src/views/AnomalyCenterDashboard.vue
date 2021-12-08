<template>
  <div class="flex flex-col px-4 pb-4">
    <BBAttention
      :style="'INFO'"
      :title="'Anomaly detection'"
      :description="'Bytebase periodically scans the managed resources and list the detected anomalies here. The list is refreshed roughly every 10 minutes.'"
    />
    <!-- This example requires Tailwind CSS v2.0+ -->
    <div class="mt-4 space-y-4">
      <div
        v-for="(item, i) in [
          databaseAnomalySummaryList,
          instanceAnomalySummaryList,
        ]"
        :key="i"
        class="space-y-2"
      >
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ i == 0 ? "Database" : "Instance" }}
        </h3>
        <dl
          class="grid grid-cols-1 gap-5 sm:grid-cols-2"
          :class="`lg:grid-cols-${item.length}`"
        >
          <template v-for="(summary, index) in item" :key="index">
            <div class="p-4 shadow rounded-lg tooltip-wrapper">
              <span class="text-sm tooltip"
                >{{ summary.environmentName }} has
                {{ summary.criticalCount }} CRITICAL,
                {{ summary.highCount }} HIGH and
                {{ summary.mediumCount }} MEDIUM anomalies</span
              >
              <dt class="textlabel">
                {{ summary.environmentName }}
              </dt>
              <dd class="flex flex-row mt-1 text-xl text-main space-x-2">
                <span class="flex flex-row items-center">
                  <svg
                    class="w-5 h-5 mr-1 text-error"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    ></path></svg
                  >{{ summary.criticalCount }}
                </span>
                <span class="flex flex-row items-center">
                  <svg
                    class="w-5 h-5 mr-1 text-warning"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                    ></path></svg
                  >{{ summary.highCount }}
                </span>
                <span class="flex flex-row items-center">
                  <svg
                    class="w-5 h-5 mr-1 text-info"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    ></path></svg
                  >{{ summary.mediumCount }}
                </span>
              </dd>
            </div>
          </template>
        </dl>
      </div>
    </div>

    <div class="mt-4 py-2 flex justify-between items-center">
      <BBTabFilter
        :tab-item-list="tabItemList"
        :selected-index="state.selectedIndex"
        @select-index="
          (index) => {
            state.selectedIndex = index;
          }
        "
      />
      <BBTableSearch
        ref="searchField"
        class="w-72"
        :placeholder="
          state.selectedIndex == DATABASE_TAB
            ? 'Search database or environment'
            : 'Search instance or environment'
        "
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <template v-if="state.selectedIndex == DATABASE_TAB">
      <AnomalyTable
        v-if="databaseAnomalySectionList.length > 0"
        :anomaly-section-list="databaseAnomalySectionList"
        :compact-section="false"
      />
      <div v-else class="text-center text-control-light">
        Hooray, no database anomaly detected!
      </div>
    </template>
    <template v-else>
      <AnomalyTable
        v-if="instanceAnomalySectionList.length > 0"
        :anomaly-section-list="instanceAnomalySectionList"
        :compact-section="false"
      />
      <div v-else class="text-center text-control-light">
        Hooray, no instance anomaly detected!
      </div>
    </template>
  </div>
</template>

<script lang="ts">
import { computed, reactive, watchEffect } from "vue-demi";
import { useStore } from "vuex";
import AnomalyTable from "../components/AnomalyTable.vue";
import { Anomaly, Database, EnvironmentId } from "../types";
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

type Summary = {
  environmentName: string;
  criticalCount: number;
  highCount: number;
  mediumCount: number;
};

interface LocalState {
  selectedIndex: number;
  searchText: string;
}

export default {
  name: "AnomalyCenterDashboard",
  components: { AnomalyTable },
  setup() {
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

    const databaseAnomalySummaryList = computed((): Summary[] => {
      const envMap: Map<EnvironmentId, Summary> = new Map();
      for (const database of databaseList.value) {
        let criticalCount = 0;
        let highCount = 0;
        let mediumCount = 0;
        for (const anomaly of database.anomalyList) {
          switch (anomaly.severity) {
            case "CRITICAL":
              criticalCount++;
              break;
            case "HIGH":
              highCount++;
              break;
            case "MEDIUM":
              mediumCount++;
              break;
          }
        }
        let summary = envMap.get(database.instance.environment.id);
        if (summary) {
          summary.criticalCount += criticalCount;
          summary.highCount += highCount;
          summary.mediumCount += mediumCount;
        } else {
          envMap.set(database.instance.environment.id, {
            environmentName: database.instance.environment.name,
            criticalCount,
            highCount,
            mediumCount,
          });
        }
      }

      const list: Summary[] = [];
      for (const environment of environmentList.value) {
        const summary = envMap.get(environment.id);
        if (summary) {
          list.push(summary);
        }
      }

      return list.reverse();
    });

    const instanceAnomalySummaryList = computed((): Summary[] => {
      const envMap: Map<EnvironmentId, Summary> = new Map();
      for (const instance of instanceList.value) {
        let criticalCount = 0;
        let highCount = 0;
        let mediumCount = 0;
        for (const anomaly of instance.anomalyList) {
          switch (anomaly.severity) {
            case "CRITICAL":
              criticalCount++;
              break;
            case "HIGH":
              highCount++;
              break;
            case "MEDIUM":
              mediumCount++;
              break;
          }
        }
        let summary = envMap.get(instance.environment.id);
        if (summary) {
          summary.criticalCount += criticalCount;
          summary.highCount += highCount;
          summary.mediumCount += mediumCount;
        } else {
          envMap.set(instance.environment.id, {
            environmentName: instance.environment.name,
            criticalCount,
            highCount,
            mediumCount,
          });
        }
      }

      const list: Summary[] = [];
      for (const environment of environmentList.value) {
        const summary = envMap.get(environment.id);
        if (summary) {
          list.push(summary);
        }
      }

      return list.reverse();
    });

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
      databaseAnomalySummaryList,
      instanceAnomalySummaryList,
      tabItemList,
      changeSearchText,
    };
  },
};
</script>
