<template>
  <div class="stage" :class="stageClass">
    <TaskStatusIcon
      :task="activeTaskInStage"
      :status="activeTaskInStage.status"
    />

    <div class="text" @click="handleClickStage">
      <div
        class="text-sm min-w-32 flex items-center space-x-1 lg:min-w-fit with-underline whitespace-nowrap"
      >
        <heroicons:arrow-small-right
          v-if="isActiveStage"
          class="w-5 h-5 inline-block"
        />
        <span>{{ $t("common.stage") }}</span>
        <span>-</span>
        <i18n-t
          v-if="!isCreating && isActiveStage"
          keypath="issue.stage-select.current"
        >
          <template #name>
            <EnvironmentV1Name :environment="environment" :link="false" />
          </template>
        </i18n-t>
        <span v-else>{{ environment.title }}</span>
      </div>
      <div class="text-xs flex gap-1 flex-row items-center">
        <div class="whitespace-no-wrap with-underline">
          {{ $t("common.task", 2) }}
        </div>
        <StageSummary :stage="stage as Stage" />
      </div>
    </div>

    <NTooltip v-if="isCreating && !isValid" trigger="hover" placement="top">
      <template #trigger>
        <heroicons:exclamation-circle-solid
          class="w-6 h-6 ml-2 text-error hover:text-error-hover"
        />
      </template>
      <span>{{ $t("issue.missing-sql-statement") }}</span>
    </NTooltip>
    <NTooltip
      v-if="
        !isCreating &&
        (planCheckStatus === PlanCheckRun_Result_Status.ERROR ||
          planCheckStatus === PlanCheckRun_Result_Status.WARNING)
      "
      trigger="hover"
      placement="top"
    >
      <template #trigger>
        <heroicons:exclamation-circle-solid
          class="w-6 h-6 ml-2"
          :class="[
            planCheckStatus === PlanCheckRun_Result_Status.ERROR
              ? 'text-error hover:text-error-hover'
              : 'text-warning hover:text-warning-hover',
          ]"
        />
      </template>
      <span>{{
        $t(
          "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
        )
      }}</span>
    </NTooltip>
  </div>
</template>

<script lang="ts" setup>
import { first, uniqBy } from "lodash-es";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import {
  isTaskFinished,
  isValidStage,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { planCheckRunSummaryForCheckRunList } from "@/components/PlanCheckRun/common";
import { EnvironmentV1Name } from "@/components/v2";
import { useEnvironmentV1Store } from "@/store";
import { EMPTY_TASK_NAME } from "@/types";
import { PlanCheckRun_Result_Status } from "@/types/proto-es/v1/plan_service_pb";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { activeTaskInStageV1 } from "@/utils";
import TaskStatusIcon from "../TaskStatusIcon.vue";
import StageSummary from "./StageSummary.vue";

const props = defineProps<{
  stage: Stage;
}>();

const {
  isCreating,
  selectedTask,
  selectedStage,
  events,
  getPlanCheckRunsForTask,
} = useIssueContext();
const environmentStore = useEnvironmentV1Store();

const activeTaskInStage = computed(() => {
  return activeTaskInStageV1(props.stage);
});

const isValid = computed(() => {
  return isValidStage(props.stage);
});

const isSelectedStage = computed(() => {
  return props.stage === selectedStage.value;
});

const isActiveStage = computed(() => {
  if (isCreating.value) {
    return false;
  }

  const taskFound = props.stage.tasks.find(
    (t) => t.name === selectedTask.value.name
  );
  if (taskFound && !isTaskFinished(taskFound)) {
    // A stage is "Active" if the ActiveTaskOfPipeline is inside this stage
    return true;
  }

  return false;
});

const stageClass = computed(() => {
  const classList: string[] = [];
  if (isCreating.value) {
    classList.push("create");
    if (!isValid.value) {
      classList.push("invalid");
    }
  }
  if (isSelectedStage.value) classList.push("selected");
  if (isActiveStage.value) classList.push("active");
  const task = activeTaskInStage.value;
  classList.push(`status_${Task_Status[task.status].toLowerCase()}`);

  return classList;
});

const environment = computed(() =>
  environmentStore.getEnvironmentByName(props.stage.environment)
);

const planCheckStatus = computed((): PlanCheckRun_Result_Status => {
  if (isCreating.value) return PlanCheckRun_Result_Status.STATUS_UNSPECIFIED;
  const planCheckList = uniqBy(
    props.stage.tasks.flatMap(getPlanCheckRunsForTask),
    (checkRun) => checkRun.name
  );
  const summary = planCheckRunSummaryForCheckRunList(planCheckList);
  if (summary.errorCount > 0) {
    return PlanCheckRun_Result_Status.ERROR;
  }
  if (summary.warnCount > 0) {
    return PlanCheckRun_Result_Status.WARNING;
  }
  return PlanCheckRun_Result_Status.SUCCESS;
});

const activeOrFirstTaskInStage = (stage: Stage) => {
  if (isCreating.value) {
    return first(stage.tasks);
  }
  const activeTask = activeTaskInStageV1(stage);
  if (activeTask.name === EMPTY_TASK_NAME) {
    return first(stage.tasks);
  }
  return activeTask;
};

const handleClickStage = () => {
  const { stage } = props;
  if (stage === selectedStage.value) return;

  if (stage) {
    const task = activeOrFirstTaskInStage(stage);
    if (task) {
      events.emit("select-task", { task });
    }
  }
};
</script>

<style scoped lang="postcss">
.stage {
  @apply cursor-default flex items-center justify-start w-full text-sm relative;
  @apply lg:flex-1;
}
.stage.selected .text .with-underline {
  @apply underline;
}

.stage .text {
  @apply cursor-pointer flex-1 ml-4 flex flex-col gap-y-0.5;
}
.stage.active .text {
  @apply font-bold;
}
.stage.status_done .text {
  @apply text-control;
}
.stage.status_pending .text,
.stage.status_pending_approval .text {
  @apply text-control;
}
.stage.active.status_pending .text,
.stage.active.status_pending_approval .text {
  @apply text-info;
}
.stage.status_running .text {
  @apply text-info;
}
.stage.status_failed .text {
  @apply text-red-500;
}
</style>
