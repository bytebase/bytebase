<template>
  <div class="flex flex-col">
    <FeatureAttentionForInstanceLicense
      v-if="hasSchemaDriftFeature"
      custom-class="mb-4"
      feature="bb.feature.schema-drift"
    />
    <FeatureAttention
      v-else
      custom-class="my-4"
      feature="bb.feature.schema-drift"
    />

    <div class="textinfolabel">
      {{ $t("anomaly.attention-desc") }}
      <a
        href="https://www.bytebase.com/docs/administration/anomaly-center/"
        target="_blank"
        class="normal-link inline-flex flex-row items-center"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4" />
      </a>
    </div>

    <div class="mt-4 space-y-4">
      <div v-for="(item, i) in anomalySummaryList" :key="i" class="space-y-2">
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ i == 0 ? $t("common.database") : $t("common.instance") }}
        </h3>
        <dl
          class="grid grid-cols-1 gap-4 sm:grid-cols-2"
          :class="`lg:grid-cols-${item.length}`"
        >
          <template v-for="(summary, index) in item" :key="index">
            <div class="px-4 py-2 border tooltip-wrapper">
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
              <div class="flex justify-between items-center">
                <dt class="textlabel">
                  {{ summary.environmentName }}
                </dt>
                <dd class="flex flex-row text-main space-x-2">
                  <span class="flex flex-row items-center">
                    <heroicons-outline:exclamation-circle
                      class="w-4 h-4 mr-1 text-error"
                    />
                    {{ summary.criticalCount }}
                  </span>
                  <span class="flex flex-row items-center">
                    <heroicons-outline:exclamation
                      class="w-4 h-4 mr-1 text-warning"
                    />
                    {{ summary.highCount }}
                  </span>
                  <span class="flex flex-row items-center">
                    <heroicons-outline:information-circle
                      class="w-4 h-4 mr-1 text-info"
                    />
                    {{ summary.mediumCount }}
                  </span>
                </dd>
              </div>
            </div>
          </template>
        </dl>
      </div>
    </div>

    <div class="mt-4 py-2 flex justify-between items-center">
      <div>
        <BBTabFilter
          v-if="!project"
          :tab-item-list="tabItemList"
          :selected-index="
            tabItemList.findIndex((tab) => tab.id === state.selectedTab)
          "
          @select-index="
          (index: number) => {
            state.selectedTab = tabItemList[index].id;
          }
        "
        />
      </div>
      <BBTableSearch
        ref="searchField"
        class="w-72"
        :placeholder="
          $t('anomaly.table-search-placeholder', {
            type:
              state.selectedTab === 'database'
                ? $t('common.database')
                : $t('common.instance'),
          })
        "
        @change-text="(text:string) => changeSearchText(text)"
      />
    </div>
    <template v-if="state.selectedTab === 'database'">
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

<script lang="ts" setup>
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { BBTabFilterItem, BBTableSectionDataSource } from "@/bbkit/types";
import {
  featureToRef,
  useAnomalyV1List,
  useCurrentUserV1,
  useDatabaseV1Store,
  useEnvironmentV1List,
  useInstanceV1List,
} from "@/store";
import { UNKNOWN_USER_NAME, ComposedProject } from "@/types";
import {
  Anomaly,
  Anomaly_AnomalySeverity,
} from "@/types/proto/v1/anomaly_service";
import {
  databaseV1Slug,
  instanceV1Slug,
  sortDatabaseV1List,
  sortInstanceV1List,
} from "@/utils";
import AnomalyTable from "./AnomalyTable.vue";

type Summary = {
  environmentName: string;
  criticalCount: number;
  highCount: number;
  mediumCount: number;
};

export type AnomalyTabId = "database" | "instance";

interface AnomalyTabFilterItem extends BBTabFilterItem {
  id: AnomalyTabId;
}

interface LocalState {
  selectedTab: AnomalyTabId;
  searchText: string;
}

const props = defineProps<{
  project?: ComposedProject;
  selectedTab?: AnomalyTabId;
}>();

const databaseStore = useDatabaseV1Store();
const { t } = useI18n();

const currentUserV1 = useCurrentUserV1();

const state = reactive<LocalState>({
  selectedTab: props.selectedTab ?? "database",
  searchText: "",
});

const environmentList = useEnvironmentV1List(false /* !showDeleted */);

const prepareDatabaseList = () => {
  // It will also be called when user logout
  if (currentUserV1.value.name !== UNKNOWN_USER_NAME) {
    databaseStore.searchDatabaseList({
      parent: "instances/-",
    });
  }
};

watchEffect(prepareDatabaseList);

const databaseList = computed(() => {
  return databaseStore.databaseListByUser(currentUserV1.value);
});

const { instanceList } = useInstanceV1List();

const allAnomalyList = useAnomalyV1List();

const databaseListByProject = computed(() => {
  return databaseList.value.filter((db) => {
    if (!props.project) {
      return true;
    }
    return props.project.name === db.project;
  });
});

const databaseAnomalySectionList = computed(
  (): BBTableSectionDataSource<Anomaly>[] => {
    const sectionList: BBTableSectionDataSource<Anomaly>[] = [];

    const dbList = sortDatabaseV1List(
      databaseListByProject.value.filter((database) => {
        if (!state.searchText) {
          return true;
        }
        if (
          database.databaseName
            .toLowerCase()
            .includes(state.searchText.toLowerCase()) ||
          database.effectiveEnvironmentEntity.title
            .toLowerCase()
            .includes(state.searchText.toLowerCase())
        ) {
          true;
        }
        return false;
      })
    );

    for (const database of dbList) {
      const anomalyListOfDatabase = allAnomalyList.value.filter(
        (anomaly) => anomaly.resource === database.name
      );

      if (anomalyListOfDatabase.length > 0) {
        sectionList.push({
          title: `${database.databaseName} (${database.effectiveEnvironmentEntity.title})`,
          link: `/db/${databaseV1Slug(database)}`,
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

    const insList = sortInstanceV1List(
      instanceList.value.filter((instance) => {
        if (!state.searchText) {
          return true;
        }
        if (
          instance.title
            .toLowerCase()
            .includes(state.searchText.toLowerCase()) ||
          instance.environmentEntity.title
            .toLowerCase()
            .includes(state.searchText.toLowerCase())
        ) {
          return true;
        }
        return false;
      })
    );

    for (const instance of insList) {
      const anomalyListOfInstance = allAnomalyList.value.filter((anomaly) =>
        anomaly.resource.startsWith(instance.name)
      );
      if (anomalyListOfInstance.length > 0) {
        sectionList.push({
          title: `${instance.title} (${instance.environmentEntity.title})`,
          link: `/instance/${instanceV1Slug(instance)}`,
          list: anomalyListOfInstance,
        });
      }
    }

    return sectionList;
  }
);

const databaseAnomalySummaryList = computed((): Summary[] => {
  const envMap: Map<string, Summary> = new Map();
  for (const database of databaseListByProject.value) {
    let criticalCount = 0;
    let highCount = 0;
    let mediumCount = 0;
    const anomalyListOfDatabase = allAnomalyList.value.filter(
      (anomaly) => anomaly.resource === database.name
    );
    for (const anomaly of anomalyListOfDatabase) {
      switch (anomaly.severity) {
        case Anomaly_AnomalySeverity.CRITICAL:
          criticalCount++;
          break;
        case Anomaly_AnomalySeverity.HIGH:
          highCount++;
          break;
        case Anomaly_AnomalySeverity.MEDIUM:
          mediumCount++;
          break;
      }
    }
    const summary = envMap.get(database.effectiveEnvironmentEntity.uid);
    if (summary) {
      summary.criticalCount += criticalCount;
      summary.highCount += highCount;
      summary.mediumCount += mediumCount;
    } else {
      envMap.set(String(database.effectiveEnvironmentEntity.uid), {
        environmentName: database.effectiveEnvironmentEntity.title,
        criticalCount,
        highCount,
        mediumCount,
      });
    }
  }

  const list: Summary[] = [];
  for (const environment of environmentList.value) {
    const summary = envMap.get(environment.uid);
    if (summary) {
      list.push(summary);
    }
  }

  return list.reverse();
});

const instanceAnomalySummaryList = computed((): Summary[] => {
  const envMap: Map<string, Summary> = new Map();
  for (const instance of instanceList.value) {
    let criticalCount = 0;
    let highCount = 0;
    let mediumCount = 0;
    const anomalyListOfInstance = allAnomalyList.value.filter(
      (anomaly) => anomaly.resource === instance.name
    );
    for (const anomaly of anomalyListOfInstance) {
      switch (anomaly.severity) {
        case Anomaly_AnomalySeverity.CRITICAL:
          criticalCount++;
          break;
        case Anomaly_AnomalySeverity.HIGH:
          highCount++;
          break;
        case Anomaly_AnomalySeverity.MEDIUM:
          mediumCount++;
          break;
      }
    }
    const summary = envMap.get(instance.environmentEntity.uid);
    if (summary) {
      summary.criticalCount += criticalCount;
      summary.highCount += highCount;
      summary.mediumCount += mediumCount;
    } else {
      envMap.set(instance.environmentEntity.uid, {
        environmentName: instance.environmentEntity.title,
        criticalCount,
        highCount,
        mediumCount,
      });
    }
  }

  const list: Summary[] = [];
  for (const environment of environmentList.value) {
    const summary = envMap.get(environment.uid);
    if (summary) {
      list.push(summary);
    }
  }

  return list.reverse();
});

const anomalySummaryList = computed(() => {
  const list = [databaseAnomalySummaryList.value];
  if (!props.project) {
    list.push(instanceAnomalySummaryList.value);
  }
  return list;
});

const tabItemList = computed((): AnomalyTabFilterItem[] => {
  return [
    {
      id: "database",
      title: t("common.database"),
      alert: databaseAnomalySectionList.value.length > 0,
    },
    {
      id: "instance",
      title: t("common.instance"),
      alert: instanceAnomalySectionList.value.length > 0,
    },
  ];
});

const changeSearchText = (searchText: string) => {
  state.searchText = searchText;
};

const hasSchemaDriftFeature = featureToRef("bb.feature.schema-drift");
</script>
