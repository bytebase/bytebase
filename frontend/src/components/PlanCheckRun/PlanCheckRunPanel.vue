<template>
  <div class="space-y-2">
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
import { getDateForPbTimestampProtoEs, type ComposedDatabase } from "@/types";
import {
  PlanCheckRun_Result_Status,
  PlanCheckRun_Type,
  type PlanCheckRun,
} from "@/types/proto-es/v1/plan_service_pb";
import { extractPlanCheckRunUID, humanizeDate } from "@/utils";
import PlanCheckRunBadgeBar from "./PlanCheckRunBadgeBar.vue";
import PlanCheckRunDetail from "./PlanCheckRunDetail.vue";

const props = defineProps<{
  planCheckRunList: PlanCheckRun[];
  database: ComposedDatabase;
  selectedType?: PlanCheckRun_Type;
}>();

const getInitialSelectedType = () => {
  if (props.selectedType) {
    return props.selectedType;
  }

  // Find the first plan check run with error or warning.
  const planCheck = props.planCheckRunList.find((checkRun) =>
    [PlanCheckRun_Result_Status.ERROR, PlanCheckRun_Result_Status.WARNING].some(
      (status) =>
        checkRun.results.map((result) => result.status).includes(status)
    )
  );
  if (planCheck) {
    return planCheck.type;
  }
  return (
    first(props.planCheckRunList)?.type ?? PlanCheckRun_Type.TYPE_UNSPECIFIED
  );
};

const { t } = useI18n();
const selectedTypeRef = ref<PlanCheckRun_Type>(getInitialSelectedType());

const selectedPlanCheckRunList = computed(() => {
  return orderBy(
    props.planCheckRunList.filter(
      (checkRun) => checkRun.type === selectedTypeRef.value
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

const handlePlanCheckRunTypeChange = (type: PlanCheckRun_Type) => {
  selectedTypeRef.value = type;
  selectedPlanCheckRunName.value = first(selectedPlanCheckRunList.value)?.name;
};
</script>
