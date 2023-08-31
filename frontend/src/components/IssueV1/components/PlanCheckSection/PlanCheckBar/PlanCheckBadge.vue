<template>
  <div class="issue-debug">
    <div>type: {{ type }}</div>
    <div>count in type: {{ planCheckRunList.length }}</div>
    <div>latest: {{ latestPlanCheckRun.name }}</div>
  </div>

  <button
    class="inline-flex items-center px-3 py-0.5 rounded-full text-sm border border-control-border"
    :class="buttonClasses"
    @click="clickable && $emit('click')"
  >
    <template v-if="status === PlanCheckRun_Status.RUNNING">
      <TaskSpinner class="-ml-1 mr-1.5 h-4 w-4 text-info" />
    </template>
    <template v-else-if="status === PlanCheckRun_Status.DONE">
      <template v-if="resultStatus === PlanCheckRun_Result_Status.SUCCESS">
        <heroicons-outline:check
          class="-ml-1 mr-1.5 mt-0.5 h-4 w-4 text-success"
        />
      </template>
      <template v-else-if="resultStatus === PlanCheckRun_Result_Status.WARNING">
        <heroicons-outline:exclamation
          class="-ml-1 mr-1.5 mt-0.5 h-4 w-4 text-warning"
        />
      </template>
      <template v-else-if="resultStatus === PlanCheckRun_Result_Status.ERROR">
        <span class="mr-1.5 font-medium text-error" aria-hidden="true">
          !
        </span>
      </template>
    </template>
    <template v-else-if="status === PlanCheckRun_Status.FAILED">
      <span class="mr-1.5 font-medium text-error" aria-hidden="true"> ! </span>
    </template>
    <template v-else-if="status === PlanCheckRun_Status.CANCELED">
      <heroicons-outline:ban class="-ml-1 mr-1.5 mt-0.5 h-4 w-4 text-control" />
    </template>

    <span>{{ name }}</span>
  </button>
</template>

<script setup lang="ts">
import { maxBy } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { planCheckRunResultStatus } from "@/components/IssueV1/logic";
import {
  PlanCheckRun,
  PlanCheckRun_Result_Status,
  PlanCheckRun_Status,
  PlanCheckRun_Type,
} from "@/types/proto/v1/rollout_service";

const props = defineProps<{
  planCheckRunList: PlanCheckRun[];
  type: PlanCheckRun_Type;
  clickable?: boolean;
  selected?: boolean;
}>();

defineEmits<{
  (event: "click"): void;
}>();

const { t } = useI18n();

const latestPlanCheckRun = computed(() => {
  return maxBy(props.planCheckRunList, (check) => Number(check.uid))!;
});

const status = computed(() => {
  return latestPlanCheckRun.value.status;
});

const resultStatus = computed(() => {
  return planCheckRunResultStatus(latestPlanCheckRun.value);
});

const buttonClasses = computed(() => {
  let bgColor = "";
  let bgHoverColor = "";
  let textColor = "";
  switch (status.value) {
    case PlanCheckRun_Status.RUNNING:
      bgColor = "bg-blue-100";
      bgHoverColor = "bg-blue-300";
      textColor = "text-blue-800";
      break;
    case PlanCheckRun_Status.FAILED:
      bgColor = "bg-red-100";
      bgHoverColor = "bg-red-300";
      textColor = "text-red-800";
      break;
    case PlanCheckRun_Status.CANCELED:
      bgColor = "bg-yellow-100";
      bgHoverColor = "bg-yellow-300";
      textColor = "text-yellow-800";
      break;
    case PlanCheckRun_Status.DONE:
      switch (resultStatus.value) {
        case PlanCheckRun_Result_Status.SUCCESS:
          bgColor = "bg-gray-100";
          bgHoverColor = "bg-gray-300";
          textColor = "text-gray-800";
          break;
        case PlanCheckRun_Result_Status.WARNING:
          bgColor = "bg-yellow-100";
          bgHoverColor = "bg-yellow-300";
          textColor = "text-yellow-800";
          break;
        case PlanCheckRun_Result_Status.ERROR:
          bgColor = "bg-red-100";
          bgHoverColor = "bg-red-300";
          textColor = "text-red-800";
          break;
      }
      break;
  }

  const styleList: string[] = [textColor];
  if (props.clickable) {
    styleList.push("cursor-pointer", `hover:${bgHoverColor}`);
    if (props.selected) {
      styleList.push(bgHoverColor);
    } else {
      styleList.push(bgColor);
    }
  } else {
    styleList.push(bgColor);
    styleList.push("cursor-default");
  }
  styleList.push("cursor-pointer", `hover:${bgHoverColor}`);
  styleList.push(bgColor);

  return styleList.join(" ");
});

// Defines the mapping from PlanCheckRun_Type to an i18n resource keypath
const PlanCheckRunTypeNameDict = new Map<PlanCheckRun_Type, string>([
  [PlanCheckRun_Type.DATABASE_STATEMENT_FAKE_ADVISE, "task.check-type.fake"],
  // ["bb.task-check.database.statement.syntax", "task.check-type.syntax"],
  [
    PlanCheckRun_Type.DATABASE_STATEMENT_COMPATIBILITY,
    "task.check-type.compatibility",
  ],
  [PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE, "task.check-type.sql-review"],
  [PlanCheckRun_Type.DATABASE_STATEMENT_TYPE, "task.check-type.statement-type"],
  [PlanCheckRun_Type.DATABASE_CONNECT, "task.check-type.connection"],
  [PlanCheckRun_Type.DATABASE_GHOST_SYNC, "task.check-type.ghost-sync"],
  [PlanCheckRun_Type.DATABASE_PITR_MYSQL, "task.check-type.pitr"],
  [
    PlanCheckRun_Type.DATABASE_STATEMENT_SUMMARY_REPORT,
    "task.check-type.summary-report",
  ],
]);

const name = computed(() => {
  const { type } = latestPlanCheckRun.value;
  const has = PlanCheckRunTypeNameDict.has(type);
  console.assert(has, `Missing PlanCheckType name of "${type}"`);
  if (has) {
    const key = PlanCheckRunTypeNameDict.get(type)!;
    return t(key);
  }
  return type;
});
</script>
