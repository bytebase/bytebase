import { NButton } from "naive-ui";
import { h } from "vue";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useDatabaseV1Store,
  useInstanceV1Store,
} from "@/store";
import {
  ComposedIssue,
  ComposedProject,
  unknownDatabase,
  UNKNOWN_ID,
} from "@/types";
import { Plan, Task, Task_Type } from "@/types/proto/v1/rollout_service";
import {
  defer,
  extractDatabaseResourceName,
  flattenSpecList,
  flattenTaskV1List,
} from "@/utils";
import { IssueContext } from "./context";

export const databaseForTask = (issue: ComposedIssue, task: Task) => {
  if (
    task.type === Task_Type.DATABASE_CREATE ||
    task.type === Task_Type.DATABASE_RESTORE_RESTORE ||
    task.type === Task_Type.DATABASE_RESTORE_CUTOVER
  ) {
    // The database is not created yet.
    // extract database info from the task's and payload's properties.
    return extractCoreDatabaseInfoFromDatabaseCreateTask(
      issue.projectEntity,
      task
    );
  } else {
    if (
      task.databaseDataUpdate ||
      task.databaseSchemaUpdate ||
      task.databaseRestoreRestore ||
      task.type === Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER ||
      task.type === Task_Type.DATABASE_SCHEMA_BASELINE
    ) {
      return useDatabaseV1Store().getDatabaseByName(task.target);
    }
  }
  return unknownDatabase();
};

const extractCoreDatabaseInfoFromDatabaseCreateTask = (
  project: ComposedProject,
  task: Task
) => {
  const coreDatabaseInfo = (instance: string, databaseName: string) => {
    const name = `${instance}/databases/${databaseName}`;
    const maybeExistedDatabase = useDatabaseV1Store().getDatabaseByName(name);
    if (maybeExistedDatabase.uid !== String(UNKNOWN_ID)) {
      return maybeExistedDatabase;
    }

    const instanceEntity = useInstanceV1Store().getInstanceByName(instance);
    return {
      ...unknownDatabase(),
      name,
      uid: String(UNKNOWN_ID),
      databaseName,
      instance,
      instanceEntity,
      project: project.name,
      projectEntity: project,
      effectiveEnvironment: instanceEntity.environment,
      effectiveEnvironmentEntity: instanceEntity.environmentEntity,
    };
  };

  if (task.databaseCreate) {
    const databaseName = task.databaseCreate.database;
    const instance = task.target;
    return coreDatabaseInfo(instance, databaseName);
  }
  if (task.databaseRestoreRestore) {
    const db = extractDatabaseResourceName(
      task.databaseRestoreRestore.target || task.target
    );
    const databaseName = db.database;
    const instance = `instances/${db.instance}`;
    return coreDatabaseInfo(instance, databaseName);
  }
  if (task.type === Task_Type.DATABASE_RESTORE_CUTOVER) {
    const db = extractDatabaseResourceName(task.target);
    const databaseName = db.database;
    const instance = `instances/${db.instance}`;
    return coreDatabaseInfo(instance, databaseName);
  }

  return unknownDatabase();
};

export const specForTask = (plan: Plan | undefined, task: Task) => {
  if (!plan) return undefined;
  return flattenSpecList(plan).find((spec) => spec.id === task.specId);
};

export const stageForTask = (issue: ComposedIssue, task: Task) => {
  const rollout = issue.rolloutEntity;
  return rollout.stages.find(
    (stage) => stage.tasks.findIndex((t) => t.name === task.name) >= 0
  );
};

export const notifyNotEditableLegacyIssue = () => {
  pushNotification({
    module: "bytebase",
    style: "CRITICAL",
    title: t("issue.not-editable-legacy-issue"),
  });
};

export const chooseUpdateTarget = (
  issue: ComposedIssue,
  selectedTask: Task,
  filter: (task: Task) => boolean,
  dialog: IssueContext["dialog"],
  updateType: string,
  isBatchMode: boolean
) => {
  type Target = "CANCELED" | "TASK" | "STAGE" | "ALL";
  const d = defer<{ target: Target; tasks: Task[] }>();

  const targets: Record<Target, Task[]> = {
    CANCELED: [],
    TASK: [selectedTask],
    STAGE: (stageForTask(issue, selectedTask)?.tasks ?? []).filter(filter),
    ALL: flattenTaskV1List(issue.rolloutEntity).filter(filter),
  };

  if (isBatchMode) {
    dialog.info({
      title: t("issue.update-statement.self", { type: updateType }),
      content: t(
        "issue.update-statement.current-change-will-apply-to-all-tasks-in-batch-mode"
      ),
      type: "info",
      autoFocus: false,
      closable: false,
      maskClosable: false,
      closeOnEsc: false,
      showIcon: false,
      positiveText: t("common.confirm"),
      negativeText: t("common.cancel"),
      onPositiveClick: () => {
        d.resolve({ target: "ALL", tasks: targets.ALL });
      },
      onNegativeClick: () => {
        d.resolve({ target: "CANCELED", tasks: [] });
      },
    });
    return d.promise;
  }

  if (targets.STAGE.length === 1 && targets.ALL.length === 1) {
    d.resolve({ target: "TASK", tasks: targets.TASK });
    return d.promise;
  }

  const $d = dialog.create({
    title: t("issue.update-statement.self", { type: updateType }),
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
