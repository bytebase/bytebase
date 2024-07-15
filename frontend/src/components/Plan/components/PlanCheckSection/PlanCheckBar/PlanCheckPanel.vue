<template>
  <div class="space-y-2">
    <TabFilter
      v-if="selectedPlanCheckRunUID && tabItemList.length > 1"
      v-model:value="selectedPlanCheckRunUID"
      :items="tabItemList"
    />

    <PlanCheckBadgeBar
      :plan-check-run-list="planCheckRunList"
      :selected-type="selectedTypeRef"
      @select-type="(type) => (selectedTypeRef = type)"
    />

    <PlanCheckDetail
      v-if="selectedPlanCheckRun"
      :plan-check-run="selectedPlanCheckRun"
      :database="database"
      :show-code-location="isLatestPlanCheckRun"
      @close="$emit('close')"
    />
  </div>
</template>

<script setup lang="ts">
import { first, orderBy } from "lodash-es";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { databaseForSpec } from "@/components/Plan/logic";
import { usePlanContext } from "@/components/Plan/logic";
import type { TabFilterItem } from "@/components/v2";
import { TabFilter } from "@/components/v2";
import { EMPTY_ID } from "@/types";
import {
  PlanCheckRun_Result_Status,
  PlanCheckRun_Type,
  type PlanCheckRun,
} from "@/types/proto/v1/plan_service";
import { humanizeDate } from "@/utils";
import PlanCheckBadgeBar from "./PlanCheckBadgeBar.vue";
import PlanCheckDetail from "./PlanCheckDetail.vue";

const props = defineProps<{
  planCheckRunList: PlanCheckRun[];
  selectedType?: PlanCheckRun_Type;
}>();

defineEmits<{
  (event: "close"): void;
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
const { plan, selectedSpec } = usePlanContext();
const selectedTypeRef = ref<PlanCheckRun_Type>(getInitialSelectedType());

const selectedPlanCheckRunList = computed(() => {
  return orderBy(
    props.planCheckRunList.filter(
      (checkRun) => checkRun.type === selectedTypeRef.value
    ),
    (checkRun) => parseInt(checkRun.uid, 10),
    "desc"
  );
});

const selectedPlanCheckRunUID = ref(first(selectedPlanCheckRunList.value)?.uid);

const selectedPlanCheckRun = computed(() => {
  const uid = selectedPlanCheckRunUID.value;
  if (!uid) return undefined;
  return selectedPlanCheckRunList.value.find(
    (checkRun) => checkRun.uid === uid
  );
});

const isLatestPlanCheckRun = computed(() => {
  return (
    selectedPlanCheckRunUID.value === first(selectedPlanCheckRunList.value)?.uid
  );
});

const tabItemList = computed(() => {
  return selectedPlanCheckRunList.value.map<TabFilterItem<string>>(
    (checkRun, i) => {
      const label =
        i === 0
          ? t("common.latest")
          : checkRun.createTime
            ? humanizeDate(checkRun.createTime)
            : `UID(${checkRun.uid})`;
      return {
        label,
        value: checkRun.uid,
      };
    }
  );
});

watch(selectedPlanCheckRunList, (list) => {
  selectedPlanCheckRunUID.value = first(list)?.uid;
});

const database = computed(() => {
  const spec = selectedSpec.value;
  if (!spec || spec.id === String(EMPTY_ID)) {
    return;
  }
  return databaseForSpec(plan.value, spec);
});
</script>
