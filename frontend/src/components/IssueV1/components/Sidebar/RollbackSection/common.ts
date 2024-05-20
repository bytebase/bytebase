import { cloneDeep } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  databaseForTask,
  notifyNotEditableLegacyIssue,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { planServiceClient } from "@/grpcweb";
import {
  useCurrentUserV1,
  experimentalFetchIssueByName,
  pushNotification,
  useIssueCommentStore,
} from "@/store";
import type { ComposedIssue } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { Task } from "@/types/proto/v1/rollout_service";
import { Task_Status, Task_Type } from "@/types/proto/v1/rollout_service";
import {
  extractUserResourceName,
  flattenTaskV1List,
  hasProjectPermissionV2,
  isDatabaseChangeRelatedIssue,
  semverCompare,
} from "@/utils";

const MIN_ROLLBACK_SQL_MYSQL_VERSION = "5.7.0";

export type RollbackUIType =
  | "SWITCH" // Show a simple checkbox to turn on/off rollback
  | "FULL" // Show featured rollback status
  | "NONE"; // Nothing

export const useRollbackContext = () => {
  const currentUserV1 = useCurrentUserV1();
  const context = useIssueContext();
  const { t } = useI18n();
  const { isCreating, issue, selectedTask: task, events } = context;
  const project = computed(() => issue.value.projectEntity);

  const showRollbackSection = computed((): boolean => {
    if (!isDatabaseChangeRelatedIssue(issue.value)) {
      return false;
    }
    if (task.value.type !== Task_Type.DATABASE_DATA_UPDATE) {
      return false;
    }
    return true;
  });

  // Decide with type of UI should be displayed.
  const rollbackUIType = computed((): RollbackUIType => {
    if (!showRollbackSection.value) {
      return "NONE";
    }

    const database = databaseForTask(issue.value, task.value);
    const { engine, engineVersion } = database.instanceEntity;
    switch (engine) {
      case Engine.MYSQL:
        if (
          !semverCompare(engineVersion, MIN_ROLLBACK_SQL_MYSQL_VERSION, "gte")
        ) {
          return "NONE";
        }
        break;
      default:
        return "NONE";
    }

    if (isCreating.value) {
      return "SWITCH";
    }

    switch (task.value.status) {
      case Task_Status.SKIPPED:
        return "NONE";
      case Task_Status.CANCELED:
        return "NONE";
      case Task_Status.DONE:
        return "FULL";
      default:
        return "SWITCH";
    }
  });

  // Decide whether current user can operate.
  const allowRollback = computed((): boolean => {
    if (rollbackUIType.value === "NONE") {
      return false;
    }

    if (isCreating.value) {
      return true;
    }

    const user = currentUserV1.value;

    if (user.email === extractUserResourceName(issue.value.creator)) {
      // Allowed to the issue creator
      return true;
    }

    if (user.email === extractUserResourceName(issue.value.assignee)) {
      // Allowed to the issue assignee
      return true;
    }

    if (hasProjectPermissionV2(project.value, user, "bb.plans.update")) {
      return true;
    }

    return false;
  });

  const rollbackEnabled = computed((): boolean => {
    if (isCreating.value) {
      const spec = specForTask(issue.value.planEntity, task.value);
      return spec?.changeDatabaseConfig?.rollbackEnabled ?? false;
    } else {
      return task.value.databaseDataUpdate?.rollbackEnabled ?? false;
    }
  });

  const toggleRollback = async (on: boolean) => {
    if (isCreating.value) {
      const config = task.value.databaseDataUpdate;
      if (config) {
        config.rollbackEnabled = on;
      }
      const spec = specForTask(issue.value.planEntity, task.value);
      if (spec && spec.changeDatabaseConfig) {
        spec.changeDatabaseConfig.rollbackEnabled = on;
      }
    } else {
      // patch plan to reconcile rollout/stages/tasks
      const planPatch = cloneDeep(issue.value.planEntity);
      const spec = specForTask(planPatch, task.value);
      if (!planPatch || !spec || !spec.changeDatabaseConfig) {
        notifyNotEditableLegacyIssue();
        return;
      }
      spec.changeDatabaseConfig.rollbackEnabled = on;
      if (task.value.databaseDataUpdate) {
        task.value.databaseDataUpdate.rollbackEnabled = on;
      }

      const updatedPlan = await planServiceClient.updatePlan({
        plan: planPatch,
        updateMask: ["steps"],
      });
      issue.value.planEntity = updatedPlan;

      const action = on ? "Enable" : "Disable";
      events.emit("status-changed", { eager: true });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });

      try {
        await useIssueCommentStore().createIssueComment({
          issueName: issue.value.name,
          comment: `${action} SQL rollback log for task [${issue.value.title}].`,
        });
      } catch {
        // fail to comment won't be too bad
      }
    }
  };

  return {
    showRollbackSection,
    rollbackUIType,
    allowRollback,
    rollbackEnabled,
    toggleRollback,
  };
};

export const maybeCreateBackTraceComments = async (newIssue: ComposedIssue) => {
  const rollbackList = [] as Array<{
    byTask: Task;
    fromIssue: string;
    fromTask: string;
  }>;
  const taskList = flattenTaskV1List(newIssue.rolloutEntity);
  for (let i = 0; i < taskList.length; i++) {
    const byTask = taskList[i];
    if (byTask.type !== Task_Type.DATABASE_DATA_UPDATE) {
      continue;
    }
    const config = byTask.databaseDataUpdate;
    if (!config) {
      continue;
    }
    if (config.rollbackFromIssue && config.rollbackFromTask) {
      rollbackList.push({
        byTask,
        fromIssue: config.rollbackFromIssue,
        fromTask: config.rollbackFromTask,
      });
    }
  }
  if (rollbackList.length === 0) return;

  for (let i = 0; i < rollbackList.length; i++) {
    const { fromIssue: fromIssueName, fromTask: fromTaskName } =
      rollbackList[i];
    const fromIssue = await experimentalFetchIssueByName(fromIssueName);

    if (fromIssue.uid === String(UNKNOWN_ID)) continue;
    const fromTask = flattenTaskV1List(fromIssue.rolloutEntity).find(
      (task) => task.name === fromTaskName
    );
    if (!fromTask || fromTask.uid === String(UNKNOWN_ID)) continue;
  }
};
