<template>
  <div class="stage" :class="stageClass">
    <TaskStatusIcon
      :task="activeTaskInStage"
      :status="activeTaskInStage.status"
    />

    <div class="text" @click="handleClickStage">
      <div
        class="text-sm min-w-32 flex items-center gap-x-1 lg:min-w-fit with-underline whitespace-nowrap"
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
        (planCheckStatus === Advice_Level.ERROR ||
          planCheckStatus === Advice_Level.WARNING)
      "
      trigger="hover"
      placement="top"
    >
      <template #trigger>
        <heroicons:exclamation-circle-solid
          class="w-6 h-6 ml-2"
          :class="[
            planCheckStatus === Advice_Level.ERROR
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
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
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

const planCheckStatus = computed((): Advice_Level => {
  if (isCreating.value) return Advice_Level.ADVICE_LEVEL_UNSPECIFIED;
  const planCheckList = uniqBy(
    props.stage.tasks.flatMap(getPlanCheckRunsForTask),
    (checkRun) => checkRun.name
  );
  const summary = planCheckRunSummaryForCheckRunList(planCheckList);
  if (summary.errorCount > 0) {
    return Advice_Level.ERROR;
  }
  if (summary.warnCount > 0) {
    return Advice_Level.WARNING;
  }
  return Advice_Level.SUCCESS;
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
  cursor: default;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  width: 100%;
  font-size: 0.875rem;
  line-height: 1.25rem;
  position: relative;
}
@media (min-width: 1024px) {
  .stage {
    flex: 1 1 0%;
  }
}
.stage.selected .text .with-underline {
  text-decoration-line: underline;
}

.stage .text {
  cursor: pointer;
  flex: 1 1 0%;
  margin-left: 1rem;
  display: flex;
  flex-direction: column;
  row-gap: 0.125rem;
}
.stage.active .text {
  font-weight: 700;
}
.stage.status_done .text {
  color: var(--color-control);
}
.stage.status_pending .text,
.stage.status_pending_approval .text {
  color: var(--color-control);
}
.stage.active.status_pending .text,
.stage.active.status_pending_approval .text {
  color: var(--color-info);
}
.stage.status_running .text {
  color: var(--color-info);
}
.stage.status_failed .text {
  color: var(--color-red-500);
}
</style>
