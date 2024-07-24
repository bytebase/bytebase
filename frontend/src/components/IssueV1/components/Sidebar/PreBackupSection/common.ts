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
  pushNotification,
  useCurrentUserV1,
  useIssueCommentStore,
} from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { Task_Status, Task_Type } from "@/types/proto/v1/rollout_service";
import {
  extractUserResourceName,
  hasProjectPermissionV2,
  isDatabaseChangeRelatedIssue,
} from "@/utils";

export const usePreBackupContext = () => {
  const currentUserV1 = useCurrentUserV1();
  const context = useIssueContext();
  const { t } = useI18n();
  const { isCreating, issue, selectedTask: task, events } = context;
  const project = computed(() => issue.value.projectEntity);

  const showPreBackupSection = computed((): boolean => {
    if (!isDatabaseChangeRelatedIssue(issue.value)) {
      return false;
    }
    if (task.value.type !== Task_Type.DATABASE_DATA_UPDATE) {
      return false;
    }
    const database = databaseForTask(issue.value, task.value);
    const { engine } = database.instanceResource;
    if (
      engine !== Engine.MYSQL &&
      engine !== Engine.TIDB &&
      engine !== Engine.MSSQL &&
      engine !== Engine.ORACLE &&
      engine !== Engine.POSTGRES
    ) {
      return false;
    }
    return true;
  });

  const allowPreBackup = computed((): boolean => {
    if (isCreating.value) {
      return true;
    }
    if (
      ![Task_Status.NOT_STARTED, Task_Status.PENDING].includes(
        task.value.status
      )
    ) {
      return false;
    }

    const user = currentUserV1.value;

    if (user.email === extractUserResourceName(issue.value.creator)) {
      // Allowed to the issue creator.
      return true;
    }

    if (hasProjectPermissionV2(project.value, user, "bb.plans.update")) {
      return true;
    }

    return false;
  });

  const preBackupEnabled = computed((): boolean => {
    const spec = specForTask(issue.value.planEntity, task.value);
    const database =
      spec?.changeDatabaseConfig?.preUpdateBackupDetail?.database;
    return database !== undefined && database !== "";
  });

  const archiveDatabase = computed((): string => {
    const database = databaseForTask(issue.value, task.value);
    const { engine } = database.instanceResource;
    if (engine === Engine.ORACLE) {
      return "BBDATAARCHIVE";
    }

    return "bbdataarchive";
  });

  const togglePreBackup = async (on: boolean) => {
    if (isCreating.value) {
      const spec = specForTask(issue.value.planEntity, task.value);
      if (spec && spec.changeDatabaseConfig) {
        const database = databaseForTask(issue.value, task.value);
        if (on) {
          spec.changeDatabaseConfig.preUpdateBackupDetail = {
            database: database.instance + "/databases/" + archiveDatabase.value,
          };
        } else {
          spec.changeDatabaseConfig.preUpdateBackupDetail = undefined;
        }
      }
    } else {
      // patch plan to reconcile rollout/stages/tasks
      const planPatch = cloneDeep(issue.value.planEntity);
      const spec = specForTask(planPatch, task.value);
      if (!planPatch || !spec || !spec.changeDatabaseConfig) {
        notifyNotEditableLegacyIssue();
        return;
      }
      const database = databaseForTask(issue.value, task.value);
      if (on) {
        spec.changeDatabaseConfig.preUpdateBackupDetail = {
          database: database.instance + "/databases/" + archiveDatabase.value,
        };
      } else {
        spec.changeDatabaseConfig.preUpdateBackupDetail = undefined;
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
          comment: `${action} prior backup for task [${task.value.title}].`,
        });
      } catch {
        // fail to comment won't be too bad
      }
    }
  };

  return {
    showPreBackupSection,
    preBackupEnabled,
    allowPreBackup,
    togglePreBackup,
  };
};
