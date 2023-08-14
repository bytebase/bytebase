<template>
  <div v-if="shouldShowEarliestAllowedTime" class="flex items-center gap-x-3">
    <h2 class="flex flex-col items-end">
      <NTooltip>
        <template #trigger>
          <div class="flex gap-x-1 items-center textlabel">
            {{ $t("common.when") }}
          </div>
        </template>
        <template #default>
          <div class="w-60">
            {{ $t("task.earliest-allowed-time-hint") }}
          </div>
        </template>
      </NTooltip>

      <div class="text-gray-600 text-xs">
        {{ "UTC" + dayjs().format("ZZ") }}
      </div>
    </h2>
    <div class="w-[12rem]">
      <NTooltip :disabled="allowEditEarliestAllowedTime">
        <template #trigger>
          <NDatePicker
            :value="earliestAllowedTime"
            :is-date-disabled="isDayPassed"
            :placeholder="$t('task.earliest-allowed-time-unset')"
            :disabled="!allowEditEarliestAllowedTime || isUpdating"
            :loading="isUpdating"
            :actions="['clear', 'confirm']"
            type="datetime"
            clearable
            @update:value="handleUpdateEarliestAllowedTime"
          />
        </template>
        <template #default>
          <div class="w-48">
            {{ $t("task.earliest-allowed-time-no-modify") }}
          </div>
        </template>
      </NTooltip>
    </div>
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
import {
  notifyNotEditableLegacyIssue,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1";
import { rolloutServiceClient } from "@/grpcweb";
import { pushNotification, useCurrentUserV1 } from "@/store";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { extractUserResourceName } from "@/utils";

dayjs.extend(isSameOrAfter);

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { isCreating, issue, isTenantMode, selectedTask, events } =
  useIssueContext();
const isUpdating = ref(false);

const shouldShowEarliestAllowedTime = computed(() => {
  if (isTenantMode.value) {
    return false;
  }
  return true;
});

// `null` to "Unset"
const earliestAllowedTime = computed(() => {
  const spec = specForTask(issue.value.planEntity, selectedTask.value);
  return spec?.earliestAllowedTime ? spec.earliestAllowedTime.getTime() : null;
});

const allowEditEarliestAllowedTime = computed(() => {
  if (isTenantMode.value) {
    return false;
  }
  if (isCreating.value) {
    return true;
  }
  // only the assignee is allowed to modify EarliestAllowedTime
  const task = selectedTask.value;

  if (issue.value.status !== IssueStatus.OPEN) {
    return false;
  }
  if (![Task_Status.NOT_STARTED, Task_Status.PENDING].includes(task.status)) {
    return false;
  }

  return (
    extractUserResourceName(issue.value.creator) === currentUser.value.email
  );
});

const handleUpdateEarliestAllowedTime = async (timestampMS: number | null) => {
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
    } finally {
      isUpdating.value = false;
    }
  }
};

const now = useNow();
const isDayPassed = (ts: number) => !dayjs(ts).isSameOrAfter(now.value, "day");
</script>
