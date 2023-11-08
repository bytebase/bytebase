import { uniqBy } from "lodash-es";
import { InjectionKey, Ref, computed, inject, provide, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  databaseForTask,
  getLocalSheetByName,
  isGroupingChangeTaskV1,
  sheetNameForSpec,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { useCurrentUserV1, useSubscriptionV1Store } from "@/store";
import { ComposedDatabase, ComposedIssue } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  Plan_ChangeDatabaseConfig_Type,
  Plan_Spec,
  Task,
  Task_Status,
  Task_Type,
  task_StatusToJSON,
} from "@/types/proto/v1/rollout_service";
import {
  MIN_GHOST_SUPPORT_MYSQL_VERSION,
  extractUserResourceName,
  flattenTaskV1List,
  getSheetStatement,
  hasWorkspacePermissionV1,
  semverCompare,
} from "@/utils";

export type GhostUIViewType = "NONE" | "OFF" | "ON";

export type IssueGhostContext = {
  viewType: Ref<GhostUIViewType>;
  showFlagsPanel: Ref<boolean>;
  denyEditGhostFlagsReasons: Ref<string[]>;
  showFeatureModal: Ref<boolean>;
  showMissingInstanceLicense: Ref<boolean>;
  toggleGhost: (spec: Plan_Spec, on: boolean) => Promise<void>;
};

export const KEY = Symbol(
  "bb.issue.context.ghost"
) as InjectionKey<IssueGhostContext>;

export const useIssueGhostContext = () => {
  return inject(KEY)!;
};

export const provideIssueGhostContext = () => {
  const { t } = useI18n();
  const me = useCurrentUserV1();
  const {
    isCreating,
    issue,
    selectedTask: task,
    reInitialize,
  } = useIssueContext();

  const viewType = computed((): GhostUIViewType => {
    return ghostViewTypeForTask(issue.value, task.value);
  });
  const showFlagsPanel = ref(false);
  const isDeploymentConfig = computed(() => {
    const spec = specForTask(issue.value.planEntity, task.value);
    return !!spec?.changeDatabaseConfig?.target?.match(
      /\/deploymentConfigs\/[^/]+/
    );
  });

  const denyEditGhostFlagsReasons = computed(() => {
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
      const ghostSyncTasks = flattenTaskV1List(
        issue.value.rolloutEntity
      ).filter(
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

  const showFeatureModal = ref(false);
  const showMissingInstanceLicense = computed(() => {
    const instances = uniqBy(
      flattenTaskV1List(issue.value.rolloutEntity).map(
        (task) => databaseForTask(issue.value, task).instanceEntity
      ),
      (instance) => instance.name
    );
    const subscriptionStore = useSubscriptionV1Store();
    return instances.some((instance) => {
      return subscriptionStore.instanceMissingLicense(
        "bb.feature.online-migration",
        instance
      );
    });
  });

  const toggleGhost = async (spec: Plan_Spec, on: boolean) => {
    const overrides: Record<string, string> = {};
    if (on) {
      overrides["ghost"] = "1";
    }

    // Backup editing statements to `overrides.sqlList`
    const flattenSpecs = (issue.value.planEntity?.steps ?? []).flatMap(
      (step) => step.specs
    );
    const sqlList: string[] = [];
    flattenSpecs.forEach((spec, i) => {
      const sheetName = sheetNameForSpec(spec);
      const sheet = getLocalSheetByName(sheetName);
      sqlList[i] = getSheetStatement(sheet);
    });
    overrides["sqlList"] = JSON.stringify(sqlList);

    await reInitialize(overrides);
  };

  const context: IssueGhostContext = {
    viewType,
    showFlagsPanel,
    denyEditGhostFlagsReasons,
    showFeatureModal,
    showMissingInstanceLicense,
    toggleGhost,
  };

  provide(KEY, context);

  return context;
};

export const allowChangeTaskGhostFlags = (issue: ComposedIssue, task: Task) => {
  return [
    Task_Status.STATUS_UNSPECIFIED, // Pending create
    Task_Status.NOT_STARTED,
    Task_Status.FAILED,
    Task_Status.CANCELED,
  ].includes(task.status);
};

export const allowGhostForDatabase = (database: ComposedDatabase) => {
  return (
    database.instanceEntity.engine === Engine.MYSQL &&
    semverCompare(
      database.instanceEntity.engineVersion,
      MIN_GHOST_SUPPORT_MYSQL_VERSION,
      "gte"
    )
  );
};

export const allowGhostForSpec = (spec: Plan_Spec | undefined) => {
  const config = spec?.changeDatabaseConfig;
  if (!config) return false;

  return [
    Plan_ChangeDatabaseConfig_Type.MIGRATE,
    Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST,
  ].includes(config.type);
};

export const allowGhostForTask = (issue: ComposedIssue, task: Task) => {
  if (
    task.target ===
    "instances/instance-0e0ee52e/databases/news_management__cn_sh"
  ) {
    return false;
  }
  return (
    allowGhostForSpec(specForTask(issue.planEntity, task)) &&
    allowGhostForDatabase(databaseForTask(issue, task))
  );
};

export const ghostViewTypeForTask = (
  issue: ComposedIssue,
  task: Task
): GhostUIViewType => {
  if (isGroupingChangeTaskV1(issue, task)) {
    return "NONE";
  }

  const spec = specForTask(issue.planEntity, task);
  const config = spec?.changeDatabaseConfig;
  if (!config) {
    return "NONE";
  }
  if (config.type === Plan_ChangeDatabaseConfig_Type.MIGRATE) {
    return "OFF";
  }
  if (config.type === Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST) {
    return "ON";
  }
  return "NONE";
};
