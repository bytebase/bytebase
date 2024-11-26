<template>
  <div class="flex flex-col space-y-4">
    <FeatureAttentionForInstanceLicense
      v-if="hasSchemaDriftFeature"
      feature="bb.feature.schema-drift"
    />
    <FeatureAttention v-else feature="bb.feature.schema-drift" />

    <div class="textinfolabel">
      {{ $t("anomaly.attention-desc") }}
      <a
        href="https://www.bytebase.com/docs/change-database/drift-detection?source=console"
        target="_blank"
        class="normal-link inline-flex flex-row items-center"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4" />
      </a>
    </div>

    <div class="space-y-4">
      <div v-for="(item, i) in anomalySummaryList" :key="i" class="space-y-2">
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ i == 0 ? $t("common.database") : $t("common.instance") }}
        </h3>
        <dl
          class="grid grid-cols-1 gap-4 sm:grid-cols-2"
          :class="`lg:grid-cols-${item.length}`"
        >
          <template v-for="(summary, index) in item" :key="index">
            <NTooltip>
              <template #trigger>
                <div class="px-4 py-2 border">
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
              <span class="text-sm">
                {{
                  $t("anomaly.tooltip", {
                    env: summary.environmentName,
                    criticalCount: summary.criticalCount,
                    highCount: summary.highCount,
                    mediumCount: summary.mediumCount,
                  })
                }}
              </span>
            </NTooltip>
          </template>
        </dl>
      </div>
    </div>

    <div class="mt-4 py-2 flex justify-between items-center">
      <TabFilter v-model:value="state.selectedTab" :items="tabItemList" />

      <SearchBox
        ref="searchField"
        v-model:value="state.searchText"
        :placeholder="
          $t('anomaly.table-search-placeholder', {
            type:
              state.selectedTab === 'database'
                ? $t('common.database')
                : $t('common.instance'),
          })
        "
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
            type: $t("common.database").toLocaleLowerCase(),
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
            type: $t("common.instance").toLocaleLowerCase(),
          })
        }}
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import type { BBTableSectionDataSource } from "@/bbkit/types";
import {
  featureToRef,
  useAnomalyV1List,
  useDatabaseV1Store,
  useEnvironmentV1List,
  useEnvironmentV1Store,
  useInstanceResourceList,
} from "@/store";
import type { ComposedProject } from "@/types";
import type { Anomaly } from "@/types/proto/v1/anomaly_service";
import { Anomaly_AnomalySeverity } from "@/types/proto/v1/anomaly_service";
import { databaseV1Url, sortDatabaseV1List, sortInstanceV1List } from "@/utils";
import {
  FeatureAttention,
  FeatureAttentionForInstanceLicense,
} from "../FeatureGuard";
import { SearchBox, TabFilter } from "../v2";
import AnomalyTable from "./AnomalyTable.vue";

type Summary = {
  environmentName: string;
  criticalCount: number;
  highCount: number;
  mediumCount: number;
};

export type AnomalyTabId = "database" | "instance";

interface LocalState {
  selectedTab: AnomalyTabId;
  searchText: string;
}

const props = defineProps<{
  project?: ComposedProject;
  selectedTab?: AnomalyTabId;
}>();

const { t } = useI18n();
const databaseStore = useDatabaseV1Store();
const environmentStore = useEnvironmentV1Store();
const allAnomalyList = useAnomalyV1List();
const instanceList = useInstanceResourceList();
const environmentList = useEnvironmentV1List(false /* !showDeleted */);

const state = reactive<LocalState>({
  selectedTab: props.selectedTab ?? "database",
  searchText: "",
});

const databaseList = computed(() => {
  return databaseStore.databaseListByUser;
});

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
          return true;
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
          link: databaseV1Url(database),
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
          environmentStore
            .getEnvironmentByName(instance.environment)
            .title.toLowerCase()
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
          title: `${instance.title} (${
            environmentStore.getEnvironmentByName(instance.environment).title
          })`,
          link: `/${instance.name}`,
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
    const summary = envMap.get(database.effectiveEnvironment);
    if (summary) {
      summary.criticalCount += criticalCount;
      summary.highCount += highCount;
      summary.mediumCount += mediumCount;
    } else {
      envMap.set(database.effectiveEnvironment, {
        environmentName: database.effectiveEnvironmentEntity.title,
        criticalCount,
        highCount,
        mediumCount,
      });
    }
  }

  const list: Summary[] = [];
  for (const environment of environmentList.value) {
    const summary = envMap.get(environment.name);
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
    const summary = envMap.get(instance.environment);
    if (summary) {
      summary.criticalCount += criticalCount;
      summary.highCount += highCount;
      summary.mediumCount += mediumCount;
    } else {
      envMap.set(instance.environment, {
        environmentName: environmentStore.getEnvironmentByName(
          instance.environment
        ).title,
        criticalCount,
        highCount,
        mediumCount,
      });
    }
  }

  const list: Summary[] = [];
  for (const environment of environmentList.value) {
    const summary = envMap.get(environment.name);
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

const tabItemList = computed(() => {
  return [
    {
      value: "database",
      label: t("common.database"),
      alert: databaseAnomalySectionList.value.length > 0,
    },
    {
      value: "instance",
      label: t("common.instance"),
      alert: instanceAnomalySectionList.value.length > 0,
    },
  ];
});

const hasSchemaDriftFeature = featureToRef("bb.feature.schema-drift");
</script>
