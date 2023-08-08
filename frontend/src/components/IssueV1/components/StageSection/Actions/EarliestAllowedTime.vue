<template>
  <div class="flex items-center gap-x-3">
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
      <NDatePicker
        v-if="allowEditEarliestAllowedTime"
        :value="earliestAllowedTime"
        :is-date-disabled="isDayPassed"
        :placeholder="$t('task.earliest-allowed-time-unset')"
        type="datetime"
        clearable
        @update:value="handleUpdateEarliestAllowedTime"
      />

      <NTooltip v-else>
        <template #trigger>
          <span class="textfield text-sm font-medium text-main">
            {{
              earliestAllowedTime
                ? dayjs(earliestAllowedTime).format("LLL")
                : $t("task.earliest-allowed-time-unset")
            }}
          </span>
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
import { computed } from "vue";
import { useNow } from "@vueuse/core";
import { NDatePicker, NTooltip } from "naive-ui";
import dayjs from "dayjs";
import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
import { cloneDeep } from "lodash-es";
import { useI18n } from "vue-i18n";

import { rolloutServiceClient } from "@/grpcweb";
import { pushNotification, useCurrentUserV1 } from "@/store";
import { extractUserResourceName } from "@/utils";
import {
  notifyNotEditableLegacyIssue,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";

dayjs.extend(isSameOrAfter);

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { isCreating, issue, isTenantMode, selectedTask, events } =
  useIssueContext();

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
    const spec = specForTask(issue.value.planEntity, selectedTask.value);
    if (!planPatch || !spec) {
      notifyNotEditableLegacyIssue();
      return;
    }

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
  }
};

const now = useNow();
const isDayPassed = (ts: number) => !dayjs(ts).isSameOrAfter(now.value, "day");
</script>
