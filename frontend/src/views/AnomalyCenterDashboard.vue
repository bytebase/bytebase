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
import { useI18n } from "vue-i18n";

import AnomalyTable from "../components/AnomalyTable.vue";
import { Anomaly, EnvironmentId, UNKNOWN_ID } from "../types";
import {
  databaseSlug,
  instanceSlug,
  hasWorkspacePermission,
  sortDatabaseList,
  sortInstanceList,
} from "../utils";
import { BBTabFilterItem, BBTableSectionDataSource } from "../bbkit/types";
import { cloneDeep } from "lodash-es";
import {
  featureToRef,
  useAnomalyList,
  useCurrentUser,
  useDatabaseStore,
  useEnvironmentList,
  useInstanceList,
} from "@/store";

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
    const databaseStore = useDatabaseStore();
    const { t } = useI18n();

    const currentUser = useCurrentUser();

    const state = reactive<LocalState>({
      selectedIndex: hasWorkspacePermission(
        "bb.permission.workspace.manage-instance",
        currentUser.value.role
      )
        ? INSTANCE_TAB
        : DATABASE_TAB,
      searchText: "",
    });

    const environmentList = useEnvironmentList(["NORMAL"]);

    const prepareDatabaseList = () => {
      // It will also be called when user logout
      if (currentUser.value.id !== UNKNOWN_ID) {
        databaseStore.fetchDatabaseList();
      }
    };

    watchEffect(prepareDatabaseList);

    const databaseList = computed(() => {
      return databaseStore.getDatabaseListByPrincipalId(currentUser.value.id);
    });

    const instanceList = useInstanceList();

    const allAnomalyList = useAnomalyList();

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
          const anomalyListOfDatabase = allAnomalyList.value.filter(
            (anomaly) => anomaly.databaseId === database.id
          );

          if (anomalyListOfDatabase.length > 0) {
            sectionList.push({
              title: `${database.name} (${database.instance.environment.name})`,
              link: `/db/${databaseSlug(database)}`,
              list: anomalyListOfDatabase,
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
          const anomalyListOfInstance = allAnomalyList.value.filter(
            (anomaly) => anomaly.instanceId === instance.id
          );
          if (anomalyListOfInstance.length > 0) {
            sectionList.push({
              title: `${instance.name} (${instance.environment.name})`,
              link: `/instance/${instanceSlug(instance)}`,
              list: anomalyListOfInstance,
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
        const anomalyListOfDatabase = allAnomalyList.value.filter(
          (anomaly) => anomaly.databaseId === database.id
        );
        for (const anomaly of anomalyListOfDatabase) {
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
        const summary = envMap.get(database.instance.environment.id);
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
        const anomalyListOfInstance = allAnomalyList.value.filter(
          (anomaly) => anomaly.instanceId === instance.id
        );
        for (const anomaly of anomalyListOfInstance) {
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
        const summary = envMap.get(instance.environment.id);
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
