<template>
  <div class="flex flex-col px-4 pb-4">
    <BBAttention
      :style="'INFO'"
      :title="$t('anomaly.attention-title')"
      :description="$t('anomaly.attention-desc')"
    />

    <FeatureAttention
      v-if="!hasSchemaDriftFeature"
      custom-class="mt-5"
      feature="bb.feature.schema-drift"
      :description="$t('subscription.features.bb-feature-schema-drift.desc')"
    />

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
          {{ i == 0 ? $t("common.database") : $t("common.instance") }}
        </h3>
        <dl
          class="grid grid-cols-1 gap-5 sm:grid-cols-2"
          :class="`lg:grid-cols-${item.length}`"
        >
          <template v-for="(summary, index) in item" :key="index">
            <div class="p-4 shadow rounded-lg tooltip-wrapper">
              <span class="text-sm tooltip">
                {{
                  $t("anomaly.tooltip", {
                    env: summary.environmentName,
                    criticalCount: summary.criticalCount,
                    highCount: summary.highCount,
                    mediumCount: summary.mediumCount,
                  })
                }}
              </span>
              <dt class="textlabel">
                {{ summary.environmentName }}
              </dt>
              <dd class="flex flex-row mt-1 text-xl text-main space-x-2">
                <span class="flex flex-row items-center">
                  <heroicons-outline:exclamation-circle
                    class="w-5 h-5 mr-1 text-error"
                  />
                  {{ summary.criticalCount }}
                </span>
                <span class="flex flex-row items-center">
                  <heroicons-outline:exclamation
                    class="w-5 h-5 mr-1 text-warning"
                  />
                  {{ summary.highCount }}
                </span>
                <span class="flex flex-row items-center">
                  <heroicons-outline:information-circle
                    class="w-5 h-5 mr-1 text-info"
                  />
                  {{ summary.mediumCount }}
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
          $t('anomaly.table-search-placeholder', {
            type:
              state.selectedIndex == DATABASE_TAB
                ? $t('common.database')
                : $t('common.instance'),
          })
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
        {{
          $t("anomaly.table-placeholder", {
            type: $t("common.database"),
          })
        }}
      </div>
    </template>
    <template v-else>
      <AnomalyTable
        v-if="instanceAnomalySectionList.length > 0"
        :anomaly-section-list="instanceAnomalySectionList"
        :compact-section="false"
      />
      <div v-else class="text-center text-control-light">
        {{
          $t("anomaly.table-placeholder", {
            type: $t("common.instance"),
          })
        }}
      </div>
    </template>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import { useI18n } from "vue-i18n";

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
import { cloneDeep } from "lodash-es";
import { featureToRef, useCurrentUser, useEnvironmentList } from "@/store";

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

export default defineComponent({
  name: "AnomalyCenterDashboard",
  components: { AnomalyTable },
  setup() {
    const store = useStore();
    const { t } = useI18n();

    const currentUser = useCurrentUser();

    const state = reactive<LocalState>({
      selectedIndex: isDBAOrOwner(currentUser.value.role)
        ? INSTANCE_TAB
        : DATABASE_TAB,
      searchText: "",
    });

    const environmentList = useEnvironmentList(["NORMAL"]);

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
          title: t("common.database"),
          alert: databaseAnomalySectionList.value.length > 0,
        },
        {
          title: t("common.instance"),
          alert: instanceAnomalySectionList.value.length > 0,
        },
      ];
    });

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    const hasSchemaDriftFeature = featureToRef("bb.feature.schema-drift");

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
      hasSchemaDriftFeature,
    };
  },
});
</script>
