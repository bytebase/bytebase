<template>
  <div v-if="shouldShowEarliestAllowedTime" class="flex items-center gap-x-3">
    <h2 class="flex flex-col items-end">
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
    <div class="w-[12rem]">
      <NTooltip :disabled="disallowEditReasons.length === 0">
        <template #trigger>
          <NDatePicker
            :value="earliestAllowedTime"
            :is-date-disabled="isDayPassed"
            :placeholder="$t('task.earliest-allowed-time-unset')"
            :disabled="disallowEditReasons.length > 0 || isUpdating"
            :loading="isUpdating"
            :actions="['clear', 'confirm']"
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
import { NDatePicker, NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import FeatureModal from "@/components/FeatureGuard/FeatureModal.vue";
import {
  isGroupingChangeTaskV1,
  latestTaskRunForTask,
  notifyNotEditableLegacyIssue,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1";
import { rolloutServiceClient } from "@/grpcweb";
import { emitWindowEvent } from "@/plugins";
import { hasFeature, pushNotification, useCurrentUserV1 } from "@/store";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  TaskRun_Status,
  Task_Status,
  task_StatusToJSON,
} from "@/types/proto/v1/rollout_service";
import { extractUserResourceName, hasWorkspacePermissionV1 } from "@/utils";
import { ErrorList } from "../../common";

dayjs.extend(isSameOrAfter);

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { isCreating, issue, isTenantMode, selectedTask, events } =
  useIssueContext();
const isUpdating = ref(false);
const showFeatureModal = ref(false);

const shouldShowEarliestAllowedTime = computed(() => {
  if (isTenantMode.value) {
    return false;
  }
  if (isGroupingChangeTaskV1(issue.value, selectedTask.value)) {
    return false;
  }
  return true;
});

// `null` to "Unset"
const earliestAllowedTime = computed(() => {
  const spec = specForTask(issue.value.planEntity, selectedTask.value);
  return spec?.earliestAllowedTime ? spec.earliestAllowedTime.getTime() : null;
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
  // Super users are always allowed change the rollout time.
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-issue",
      currentUser.value.userRole
    )
  ) {
    allow = true;
  }
  // Issue creator is allowed to change the rollout time.
  if (extractUserResourceName(issue.value.creator) == currentUser.value.email) {
    allow = true;
  }
  // Issue assignee is allowed to change the rollout time.
  if (
    extractUserResourceName(issue.value.assignee) == currentUser.value.email
  ) {
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
      spec.earliestAllowedTime = new Date();
      spec.earliestAllowedTime.setTime(timestampMS);
    }
  } else {
    const planPatch = cloneDeep(issue.value.planEntity);
    const spec = specForTask(planPatch, selectedTask.value);
    if (!planPatch || !spec) {
      notifyNotEditableLegacyIssue();
      return;
    }

    isUpdating.value = true;
    try {
      if (!timestampMS) {
        spec.earliestAllowedTime = undefined;
      } else {
        spec.earliestAllowedTime = new Date();
        spec.earliestAllowedTime.setTime(timestampMS);
      }

      const updatedPlan = await rolloutServiceClient.updatePlan({
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
