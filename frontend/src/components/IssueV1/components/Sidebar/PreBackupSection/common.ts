import { cloneDeep } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  databaseForTask,
  latestTaskRunForTask,
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
import {
  Task_Status,
  Task_Type,
  TaskRun_Status,
} from "@/types/proto/v1/rollout_service";
import {
  extractUserResourceName,
  hasProjectPermissionV2,
  isDatabaseChangeRelatedIssue,
} from "@/utils";

const PRE_BACKUP_AVAILABLE_ENGINES = [
  Engine.MYSQL,
  Engine.TIDB,
  Engine.MSSQL,
  Engine.ORACLE,
  Engine.POSTGRES,
];

const ROLLBACK_AVAILABLE_ENGINES = [Engine.MYSQL, Engine.POSTGRES, Engine.MSSQL, Engine.ORACLE];

export const usePreBackupContext = () => {
  const { t } = useI18n();
  const currentUserV1 = useCurrentUserV1();
  const { isCreating, issue, selectedTask, events } = useIssueContext();

  const project = computed(() => issue.value.projectEntity);

  const database = computed(() =>
    databaseForTask(issue.value, selectedTask.value)
  );

  const latestTaskRun = computed(() =>
    latestTaskRunForTask(issue.value, selectedTask.value)
  );

  const showPreBackupSection = computed((): boolean => {
    if (!isDatabaseChangeRelatedIssue(issue.value)) {
      return false;
    }
    if (selectedTask.value.type !== Task_Type.DATABASE_DATA_UPDATE) {
      return false;
    }
    const { engine } = database.value.instanceResource;
    if (!PRE_BACKUP_AVAILABLE_ENGINES.includes(engine)) {
      return false;
    }
    return true;
  });

  const allowPreBackup = computed((): boolean => {
    // Disallow pre-backup if no backup available for the target database.
    if (!database.value.backupAvailable) {
      return false;
    }

    // Allow toggle pre-backup when creating.
    if (isCreating.value) {
      return true;
    }

    // Only allow pre-backup for non-done and non-running tasks.
    if (
      [Task_Status.DONE, Task_Status.RUNNING].includes(
        selectedTask.value.status
      )
    ) {
      return false;
    }

    // Allowed to the issue creator.
    if (
      currentUserV1.value.email === extractUserResourceName(issue.value.creator)
    ) {
      return true;
    }

    if (hasProjectPermissionV2(project.value, "bb.plans.update")) {
      return true;
    }

    return false;
  });

  const preBackupEnabled = computed((): boolean => {
    const spec = specForTask(issue.value.planEntity, selectedTask.value);
    const database =
      spec?.changeDatabaseConfig?.preUpdateBackupDetail?.database;
    return database !== undefined && database !== "";
  });

  const archiveDatabase = computed((): string => {
    const { engine } = database.value.instanceResource;
    return getArchiveDatabase(engine);
  });

  const togglePreBackup = async (on: boolean) => {
    if (isCreating.value) {
      const spec = specForTask(issue.value.planEntity, selectedTask.value);
      if (spec && spec.changeDatabaseConfig) {
        if (on) {
          spec.changeDatabaseConfig.preUpdateBackupDetail = {
            database:
              database.value.instance + "/databases/" + archiveDatabase.value,
          };
        } else {
          spec.changeDatabaseConfig.preUpdateBackupDetail = undefined;
        }
      }
    } else {
      // patch plan to reconcile rollout/stages/tasks
      const planPatch = cloneDeep(issue.value.planEntity);
      const spec = specForTask(planPatch, selectedTask.value);
      if (!planPatch || !spec || !spec.changeDatabaseConfig) {
        notifyNotEditableLegacyIssue();
        return;
      }

      if (on) {
        spec.changeDatabaseConfig.preUpdateBackupDetail = {
          database:
            database.value.instance + "/databases/" + archiveDatabase.value,
        };
      } else {
        spec.changeDatabaseConfig.preUpdateBackupDetail = undefined;
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

      const action = on ? "Enable" : "Disable";
      try {
        await useIssueCommentStore().createIssueComment({
          issueName: issue.value.name,
          comment: `${action} prior backup for task [${selectedTask.value.target}].`,
        });
      } catch {
        // fail to comment won't be too bad
      }
    }
  };

  const showRollbackSection = computed((): boolean => {
    if (!showPreBackupSection.value) {
      return false;
    }
    if (!preBackupEnabled.value) {
      return false;
    }
    if (
      !ROLLBACK_AVAILABLE_ENGINES.includes(
        database.value.instanceResource.engine
      )
    ) {
      return false;
    }
    if (!latestTaskRun.value) {
      return false;
    }
    if (latestTaskRun.value.status !== TaskRun_Status.DONE) {
      return false;
    }
    if (latestTaskRun.value.priorBackupDetail?.items.length === 0) {
      return false;
    }
    return true;
  });

  const allowRollback = computed((): boolean => {
    return hasProjectPermissionV2(project.value, "bb.issues.create");
  });

  return {
    // Pre-backup related.
    showPreBackupSection,
    preBackupEnabled,
    allowPreBackup,
    togglePreBackup,

    // Rollback related.
    showRollbackSection,
    allowRollback,
  };
};

export const getArchiveDatabase = (engine: Engine): string => {
  if (engine === Engine.ORACLE) {
    return "BBDATAARCHIVE";
  }
  return "bbdataarchive";
};
