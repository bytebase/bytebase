<template>
  <div v-if="shouldShowEarliestAllowedTime" class="flex flex-col gap-y-1">
    <h2 class="flex flex-row items-center gap-x-1">
      <div class="flex gap-x-1 items-center textlabel">
        <NTooltip>
          <template #trigger>
            <span>{{ $t("task.rollout-time") }}</span>
          </template>
          <template #default>
            <div class="w-60">
              {{ $t("task.earliest-allowed-time-hint") }}
            </div>
          </template>
        </NTooltip>

        <FeatureBadge feature="bb.feature.task-schedule-time" />
      </div>

      <div class="text-gray-600 text-xs">
        {{ "UTC" + dayjs().format("ZZ") }}
      </div>
    </h2>
    <div>
      <NTooltip :disabled="disallowEditReasons.length === 0">
        <template #trigger>
          <NDatePicker
            :value="earliestAllowedTime"
            :is-date-disabled="isDayPassed"
            :placeholder="$t('task.earliest-allowed-time-unset')"
            :disabled="disallowEditReasons.length > 0 || isUpdating"
            :loading="isUpdating"
            :actions="['confirm']"
            type="datetime"
            clearable
            @update:value="handleUpdateEarliestAllowedTime"
          />
        </template>
        <template #default>
          <ErrorList :errors="disallowEditReasons" />
        </template>
      </NTooltip>
    </div>

    <FeatureModal
      :open="showFeatureModal"
      feature="bb.feature.task-schedule-time"
      @cancel="showFeatureModal = false"
    />
  </div>
</template>

<script setup lang="ts">
import { useNow } from "@vueuse/core";
import dayjs from "dayjs";
import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
import { cloneDeep } from "lodash-es";
import { NButton, NDatePicker, NTooltip, useDialog } from "naive-ui";
import { computed, h, ref } from "vue";
import { useI18n } from "vue-i18n";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import FeatureModal from "@/components/FeatureGuard/FeatureModal.vue";
import {
  isTaskEditable,
  latestTaskRunForTask,
  notifyNotEditableLegacyIssue,
  specForTask,
  stageForTask,
  useIssueContext,
} from "@/components/IssueV1";
import ErrorList from "@/components/misc/ErrorList.vue";
import { planServiceClient } from "@/grpcweb";
import { emitWindowEvent } from "@/plugins";
import {
  hasFeature,
  pushNotification,
  useCurrentUserV1,
  extractUserId,
} from "@/store";
import { getTimeForPbTimestamp } from "@/types";
import { Timestamp } from "@/types/proto/google/protobuf/timestamp";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import {
  type Task,
  TaskRun_Status,
  Task_Status,
  task_StatusToJSON,
} from "@/types/proto/v1/rollout_service";
import {
  defer,
  flattenTaskV1List,
  hasProjectPermissionV2,
  isDatabaseChangeRelatedIssue,
} from "@/utils";

dayjs.extend(isSameOrAfter);

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { isCreating, issue, selectedTask, events, getPlanCheckRunsForTask } =
  useIssueContext();
const isUpdating = ref(false);
const showFeatureModal = ref(false);
const dialog = useDialog();

const shouldShowEarliestAllowedTime = computed(() => {
  if (!isDatabaseChangeRelatedIssue(issue.value)) {
    return false;
  }
  return true;
});

// `null` to "Unset"
const earliestAllowedTime = computed(() => {
  const spec = specForTask(issue.value.planEntity, selectedTask.value);
  return spec?.earliestAllowedTime
    ? getTimeForPbTimestamp(spec.earliestAllowedTime)
    : null;
});

const disallowEditReasons = computed(() => {
  if (isCreating.value) {
    return [];
  }

  const errors: string[] = [];
  if (issue.value.status !== IssueStatus.OPEN) {
    if (issue.value.status === IssueStatus.DONE) {
      errors.push(t("issue.disallow-edit-reasons.issue-is-done"));
    }
    if (issue.value.status === IssueStatus.CANCELED) {
      errors.push(t("issue.disallow-edit-reasons.issue-is-canceled"));
    }
    return errors;
  }

  let allow = false;
  // Issue creator is allowed to change the rollout time.
  if (extractUserId(issue.value.creator) === currentUser.value.email) {
    allow = true;
  }
  if (hasProjectPermissionV2(issue.value.projectEntity, "bb.plans.update")) {
    allow = true;
  }
  if (!allow) {
    errors.push(t("issue.you-are-not-allowed-to-change-this-value"));
  }

  const task = selectedTask.value;
  if (
    task.status === Task_Status.RUNNING ||
    task.status === Task_Status.PENDING
  ) {
    const latestTaskRun = latestTaskRunForTask(issue.value, selectedTask.value);
    if (latestTaskRun) {
      if (
        latestTaskRun.status === TaskRun_Status.PENDING ||
        latestTaskRun.status === TaskRun_Status.RUNNING
      ) {
        errors.push(
          t("issue.disallow-edit-reasons.task-is-running-cancel-first")
        );
      }
    }
  } else if ([Task_Status.DONE, Task_Status.SKIPPED].includes(task.status)) {
    errors.push(
      t("issue.disallow-edit-reasons.task-is-x-status", {
        status: task_StatusToJSON(task.status).toLowerCase(),
      })
    );
  }

  return errors;
});

const chooseUpdateStatementTarget = () => {
  type Target = "CANCELED" | "TASK" | "STAGE" | "ALL";
  const d = defer<{ target: Target; tasks: Task[] }>();

  const targets: Record<Target, Task[]> = {
    CANCELED: [],
    TASK: [selectedTask.value],
    STAGE: (stageForTask(issue.value, selectedTask.value)?.tasks ?? []).filter(
      (task) => {
        return isTaskEditable(task, getPlanCheckRunsForTask(task)).length === 0;
      }
    ),
    ALL: flattenTaskV1List(issue.value.rolloutEntity).filter((task) => {
      return isTaskEditable(task, getPlanCheckRunsForTask(task)).length === 0;
    }),
  };

  if (targets.STAGE.length === 1 && targets.ALL.length === 1) {
    d.resolve({ target: "TASK", tasks: targets.TASK });
    return d.promise;
  }

  const $d = dialog.create({
    title: t("issue.update-rollout-time.self"),
    content: t("issue.update-statement.apply-current-change-to"),
    type: "info",
    autoFocus: false,
    closable: false,
    maskClosable: false,
    closeOnEsc: false,
    showIcon: false,
    action: () => {
      const finish = (target: Target) => {
        d.resolve({ target, tasks: targets[target] });
        $d.destroy();
      };

      const CANCEL = h(
        NButton,
        { size: "small", onClick: () => finish("CANCELED") },
        {
          default: () => t("common.cancel"),
        }
      );
      const TASK = h(
        NButton,
        { size: "small", onClick: () => finish("TASK") },
        {
          default: () => t("issue.update-statement.target.selected-task"),
        }
      );
      const buttons = [CANCEL, TASK];
      if (targets.STAGE.length > 1) {
        // More than one editable tasks in stage
        // Add "Selected stage" option
        const STAGE = h(
          NButton,
          { size: "small", onClick: () => finish("STAGE") },
          {
            default: () => t("issue.update-statement.target.selected-stage"),
          }
        );
        buttons.push(STAGE);
      }
      if (targets.ALL.length > targets.STAGE.length) {
        // More editable tasks in other stages
        // Add "All tasks" option
        const ALL = h(
          NButton,
          { size: "small", onClick: () => finish("ALL") },
          {
            default: () => t("issue.update-statement.target.all-tasks"),
          }
        );
        buttons.push(ALL);
      }

      return h(
        "div",
        { class: "flex items-center justify-end gap-x-2" },
        buttons
      );
    },
    onClose() {
      d.resolve({ target: "CANCELED", tasks: [] });
    },
  });

  return d.promise;
};

const handleUpdateEarliestAllowedTime = async (timestampMS: number | null) => {
  if (!hasFeature("bb.feature.task-schedule-time")) {
    showFeatureModal.value = true;
    return;
  }

  if (isCreating.value) {
    const spec = specForTask(issue.value.planEntity, selectedTask.value);
    if (!spec) return;
    if (!timestampMS) {
      spec.earliestAllowedTime = undefined;
    } else {
      spec.earliestAllowedTime = Timestamp.fromPartial({
        seconds: Math.floor(timestampMS / 1000),
      });
    }
  } else {
    const { target, tasks } = await chooseUpdateStatementTarget();

    if (target === "CANCELED" || tasks.length === 0) {
      return;
    }

    const planPatch = cloneDeep(issue.value.planEntity);
    if (!planPatch) {
      notifyNotEditableLegacyIssue();
      return;
    }

    isUpdating.value = true;
    try {
      const specs: Plan_Spec[] = [];
      tasks.forEach((task) => {
        const spec = specForTask(planPatch, task);
        if (spec) {
          specs.push(spec);
        }
      });
      const distinctSpecIds = new Set(specs.map((s) => s.id));
      if (distinctSpecIds.size === 0) {
        notifyNotEditableLegacyIssue();
        return;
      }

      const specsToPatch = planPatch.steps
        .flatMap((step) => step.specs)
        .filter((spec) => distinctSpecIds.has(spec.id));

      for (let i = 0; i < specsToPatch.length; i++) {
        const spec = specsToPatch[i];
        if (!timestampMS) {
          spec.earliestAllowedTime = undefined;
        } else {
          spec.earliestAllowedTime = Timestamp.fromPartial({
            seconds: Math.floor(timestampMS / 1000),
          });
        }
      }

      const updatedPlan = await planServiceClient.updatePlan({
        plan: planPatch,
        updateMask: ["steps"],
      });

      issue.value.planEntity = updatedPlan;

      events.emit("status-changed", { eager: true });

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });

      emitWindowEvent("bb.pipeline-task-earliest-allowed-time-update");
    } finally {
      isUpdating.value = false;
    }
  }
};

const now = useNow();
const isDayPassed = (ts: number) => !dayjs(ts).isSameOrAfter(now.value, "day");
</script>
