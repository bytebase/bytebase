<template>
  <NTag
    :class="[selected && 'shadow-sm']"
    :type="tagType"
    round
    :bordered="selected"
    @click="clickable && $emit('click')"
  >
    <template #icon>
      <template v-if="status === PlanCheckRun_Status.RUNNING">
        <TaskSpinner class="h-4 w-4 text-info" />
      </template>
      <template v-else-if="status === PlanCheckRun_Status.DONE">
        <CheckIcon
          v-if="resultStatus === Advice_Level.SUCCESS"
          class="text-success"
          :size="16"
        />
        <TriangleAlertIcon
          v-else-if="resultStatus === Advice_Level.WARNING"
          :size="16"
        />
        <CircleAlertIcon
          v-else-if="resultStatus === Advice_Level.ERROR"
          :size="16"
        />
      </template>
      <template v-else-if="status === PlanCheckRun_Status.FAILED">
        <CircleAlertIcon :size="16" />
      </template>
      <template v-else-if="status === PlanCheckRun_Status.CANCELED">
        <TriangleAlertIcon :size="16" />
      </template>
    </template>
    <span>{{ title }}</span>
  </NTag>
</template>

<script setup lang="ts">
import { maxBy } from "lodash-es";
import { CheckIcon, CircleAlertIcon, TriangleAlertIcon } from "lucide-vue-next";
import { NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { TaskSpinner } from "@/components/IssueV1/components/common";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import {
  PlanCheckRun_Result_Type,
  PlanCheckRun_Status,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { extractPlanCheckRunUID } from "@/utils";
import { planCheckRunResultStatus } from "./common";

const props = defineProps<{
  planCheckRuns: PlanCheckRun[];
  type: PlanCheckRun_Result_Type;
  clickable?: boolean;
  selected?: boolean;
}>();

defineEmits<{
  (event: "click"): void;
}>();

const { t } = useI18n();

const latestPlanCheckRun = computed(() => {
  // Get the latest PlanCheckRun by UID.
  return maxBy(props.planCheckRuns, (check) =>
    Number(extractPlanCheckRunUID(check.name))
  )!;
});

const status = computed(() => {
  return latestPlanCheckRun.value.status;
});

const tagType = computed(() => {
  if (latestPlanCheckRun.value.status === PlanCheckRun_Status.FAILED) {
    return "error";
  }
  if (latestPlanCheckRun.value.status === PlanCheckRun_Status.CANCELED) {
    return "warning";
  }
  if (latestPlanCheckRun.value.status === PlanCheckRun_Status.RUNNING) {
    return "info";
  }
  if (latestPlanCheckRun.value.status !== PlanCheckRun_Status.DONE) {
    // Should not reach here.
    return "default";
  }

  switch (planCheckRunResultStatus(latestPlanCheckRun.value)) {
    case Advice_Level.SUCCESS:
      return "default";
    case Advice_Level.WARNING:
      return "warning";
    case Advice_Level.ERROR:
      return "error";
  }
  // Should not reach here.
  return "default";
});

const resultStatus = computed(() => {
  return planCheckRunResultStatus(latestPlanCheckRun.value);
});

const title = computed(() => {
  // Use the type prop directly instead of getting from checkRun
  switch (props.type) {
    case PlanCheckRun_Result_Type.STATEMENT_ADVISE:
      return t("task.check-type.sql-review.self");
    case PlanCheckRun_Result_Type.GHOST_SYNC:
      return t("task.check-type.ghost-sync");
    case PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT:
      return t("task.check-type.summary-report");
    default:
      console.assert(false, `Missing PlanCheckType name of "${props.type}"`);
      return String(props.type);
  }
});
</script>
