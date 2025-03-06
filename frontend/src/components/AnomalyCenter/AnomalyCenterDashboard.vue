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
      <div class="space-y-2">
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ $t("common.database") }}
        </h3>
        <dl
          class="grid grid-cols-1 gap-4 sm:grid-cols-2"
          :class="`lg:grid-cols-${databaseAnomalySummaryList.length}`"
        >
          <div
            v-for="(summary, index) in databaseAnomalySummaryList"
            :key="index"
            class="px-4 py-2 border"
          >
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
        </dl>
      </div>
    </div>

    <AnomalyTable
      v-if="databaseAnomalySectionList.length > 0"
      :anomaly-section-list="databaseAnomalySectionList"
      :compact-section="false"
    />
    <div v-else class="text-left text-control-light my-4">
      {{
        $t("anomaly.table-placeholder", {
          type: $t("common.database").toLocaleLowerCase(),
        })
      }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import type { BBTableSectionDataSource } from "@/bbkit/types";
import {
  featureToRef,
  useAnomalyV1Store,
  useDatabaseV1Store,
  useEnvironmentV1List,
  batchGetOrFetchDatabases,
} from "@/store";
import type { ComposedDatabase, ComposedProject } from "@/types";
import { isValidDatabaseName } from "@/types";
import type { Anomaly } from "@/types/proto/v1/anomaly_service";
import { Anomaly_AnomalySeverity } from "@/types/proto/v1/anomaly_service";
import { databaseV1Url } from "@/utils";
import {
  FeatureAttention,
  FeatureAttentionForInstanceLicense,
} from "../FeatureGuard";
import AnomalyTable from "./AnomalyTable.vue";

type Summary = {
  environmentName: string;
  criticalCount: number;
  highCount: number;
  mediumCount: number;
};

interface LocalDataSource extends BBTableSectionDataSource<Anomaly> {
  database: ComposedDatabase;
}

const props = defineProps<{
  project: ComposedProject;
}>();

const databaseStore = useDatabaseV1Store();
const environmentList = useEnvironmentV1List(false /* !showDeleted */);
const allAnomalyList = ref<Anomaly[]>([]);

onMounted(async () => {
  // Prepare all anomaly list.
  allAnomalyList.value = await useAnomalyV1Store().fetchAnomalyList(
    props.project?.name,
    {}
  );

  await batchGetOrFetchDatabases(
    allAnomalyList.value.map((anomaly) => anomaly.resource)
  );
});

const databaseAnomalySectionList = computed((): LocalDataSource[] => {
  const sectionMap: Map<string, LocalDataSource> = new Map();

  for (const anomaly of allAnomalyList.value) {
    const database = databaseStore.getDatabaseByName(anomaly.resource);
    if (isValidDatabaseName(database.name)) {
      if (!sectionMap.has(database.name)) {
        sectionMap.set(database.name, {
          database,
          title: `${database.databaseName} (${database.effectiveEnvironmentEntity.title})`,
          link: databaseV1Url(database),
          list: [],
        });
      }
      sectionMap.get(database.name)!.list.push(anomaly);
    }
  }

  return [...sectionMap.values()];
});

const databaseAnomalySummaryList = computed((): Summary[] => {
  const envMap: Map<string, Summary> = new Map();
  for (const item of databaseAnomalySectionList.value) {
    const { database, list } = item;
    let criticalCount = 0;
    let highCount = 0;
    let mediumCount = 0;

    for (const anomaly of list) {
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

const hasSchemaDriftFeature = featureToRef("bb.feature.schema-drift");
</script>
