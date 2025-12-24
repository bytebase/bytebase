<template>
  <div class="flex flex-col gap-y-2">
    <TabFilter
      v-if="selectedPlanCheckRunName && tabItemList.length > 1"
      v-model:value="selectedPlanCheckRunName"
      :items="tabItemList"
    />

    <PlanCheckRunBadgeBar
      :plan-check-run-list="planCheckRunList"
      :selected-type="selectedTypeRef"
      @select-type="handlePlanCheckRunTypeChange"
    />

    <PlanCheckRunDetail
      v-if="selectedPlanCheckRun"
      :plan-check-run="selectedPlanCheckRun"
      :selected-type="selectedTypeRef"
      :database="database"
      :show-code-location="isLatestPlanCheckRun"
    />
  </div>
</template>

<script setup lang="ts">
import { first, orderBy } from "lodash-es";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { TabFilterItem } from "@/components/v2";
import { TabFilter } from "@/components/v2";
import { type ComposedDatabase, getDateForPbTimestampProtoEs } from "@/types";
import {
  type PlanCheckRun,
  PlanCheckRun_Result_Type,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { extractPlanCheckRunUID, humanizeDate } from "@/utils";
import { HiddenPlanCheckTypes } from "./common";
import PlanCheckRunBadgeBar from "./PlanCheckRunBadgeBar.vue";
import PlanCheckRunDetail from "./PlanCheckRunDetail.vue";

const props = defineProps<{
  planCheckRunList: PlanCheckRun[];
  database: ComposedDatabase;
  selectedType?: PlanCheckRun_Result_Type;
}>();

const getInitialSelectedType = () => {
  if (props.selectedType) {
    return props.selectedType;
  }

  // Find the first result type with error or warning
  for (const run of props.planCheckRunList) {
    for (const result of run.results) {
      if (HiddenPlanCheckTypes.has(result.type)) {
        continue;
      }
      if (
        result.status === Advice_Level.ERROR ||
        result.status === Advice_Level.WARNING
      ) {
        return result.type;
      }
    }
  }

  // Fall back to first non-hidden result type
  for (const run of props.planCheckRunList) {
    for (const result of run.results) {
      if (!HiddenPlanCheckTypes.has(result.type)) {
        return result.type;
      }
    }
  }

  return PlanCheckRun_Result_Type.TYPE_UNSPECIFIED;
};

const { t } = useI18n();
const selectedTypeRef = ref<PlanCheckRun_Result_Type>(getInitialSelectedType());

const selectedPlanCheckRunList = computed(() => {
  // With consolidated model, return runs that have results of the selected type
  return orderBy(
    props.planCheckRunList.filter((run) =>
      run.results.some((result) => result.type === selectedTypeRef.value)
    ),
    (checkRun) => parseInt(extractPlanCheckRunUID(checkRun.name), 10),
    "desc"
  );
});

const selectedPlanCheckRunName = ref(
  first(selectedPlanCheckRunList.value)?.name
);

const selectedPlanCheckRun = computed(() => {
  const name = selectedPlanCheckRunName.value;
  if (!name) return undefined;
  return selectedPlanCheckRunList.value.find(
    (checkRun) => checkRun.name === name
  );
});

const isLatestPlanCheckRun = computed(() => {
  return (
    selectedPlanCheckRunName.value ===
    first(selectedPlanCheckRunList.value)?.name
  );
});

const tabItemList = computed(() => {
  return selectedPlanCheckRunList.value.map<TabFilterItem<string>>(
    (planCheckRun, i) => {
      const label =
        i === 0
          ? t("common.latest")
          : planCheckRun.createTime
            ? humanizeDate(
                getDateForPbTimestampProtoEs(planCheckRun.createTime)
              )
            : `UID(${extractPlanCheckRunUID(planCheckRun.name)})`;
      return {
        label,
        value: planCheckRun.name,
      };
    }
  );
});

const handlePlanCheckRunTypeChange = (type: PlanCheckRun_Result_Type) => {
  selectedTypeRef.value = type;
  selectedPlanCheckRunName.value = first(selectedPlanCheckRunList.value)?.name;
};
</script>
