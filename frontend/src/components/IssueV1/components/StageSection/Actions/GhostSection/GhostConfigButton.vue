<template>
  <NTooltip :disabled="errors.length === 0">
    <template #trigger>
      <NButton
        tag="div"
        quaternary
        size="small"
        style="--n-padding: 0 5px"
        @click="showFlagsPanel = true"
      >
        <template #icon>
          <Wrench class="w-4 h-4" />
        </template>
        <template #default>
          {{ $t("task.online-migration.configure") }}
        </template>
      </NButton>
    </template>
    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { Wrench } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { specForTask, useIssueContext } from "@/components/IssueV1/logic";
import ErrorList from "@/components/misc/ErrorList.vue";
import { useCurrentUserV1 } from "@/store";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Task_Type, task_StatusToJSON } from "@/types/proto/v1/rollout_service";
import {
  extractUserResourceName,
  flattenTaskV1List,
  hasWorkspacePermissionV1,
} from "@/utils";
import { allowChangeTaskGhostFlags, useIssueGhostContext } from "./common";

const { t } = useI18n();
const { isCreating, issue, selectedTask: task } = useIssueContext();
const { showFlagsPanel } = useIssueGhostContext();
const me = useCurrentUserV1();

const isDeploymentConfig = computed(() => {
  const spec = specForTask(issue.value.planEntity, task.value);
  return !!spec?.changeDatabaseConfig?.target?.match(
    /\/deploymentConfigs\/[^/]+/
  );
});

const errors = computed(() => {
  if (isCreating.value) {
    return [];
  }

  if (issue.value.status !== IssueStatus.OPEN) {
    return [t("issue.error.issue-is-not-open")];
  }

  const errors: string[] = [];

  if (extractUserResourceName(issue.value.creator) !== me.value.email) {
    if (
      !hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-issue",
        me.value.userRole
      )
    ) {
      return [t("issue.error.you-don-have-privilege-to-edit-this-issue")];
    }
  }

  if (isDeploymentConfig.value) {
    // If a task is created by deploymentConfig. It is editable only when all
    // its "brothers" (created by the same deploymentConfig) are editable.
    // By which "editable" means a task's status meets the requirements.
    const ghostSyncTasks = flattenTaskV1List(issue.value.rolloutEntity).filter(
      (task) => task.type === Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC
    );
    if (
      ghostSyncTasks.some(
        (task) => !allowChangeTaskGhostFlags(issue.value, task)
      )
    ) {
      errors.push(
        t(
          "task.online-migration.error.some-tasks-are-not-editable-in-batch-mode"
        )
      );
    }
  } else {
    if (!allowChangeTaskGhostFlags(issue.value, task.value)) {
      errors.push(
        t("task.online-migration.error.x-status-task-is-not-editable", {
          status: task_StatusToJSON(task.value.status),
        })
      );
    }
  }
  return errors;
});
</script>
