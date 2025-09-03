import { create } from "@bufbuild/protobuf";
import { NButton } from "naive-ui";
import { h } from "vue";
import { t } from "@/plugins/i18n";
import {
  projectNamePrefix,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceResourceByName,
  useProjectV1Store,
} from "@/store";
import {
  isValidDatabaseName,
  isValidInstanceName,
  unknownDatabase,
  unknownInstance,
  type ComposedDatabase,
  type ComposedIssue,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { InstanceResourceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { IssueStatus, type Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import {
  defer,
  extractDatabaseResourceName,
  extractProjectResourceName,
  flattenTaskV1List,
  isValidIssueName,
} from "@/utils";
import type { IssueContext } from "./context";

export const projectOfIssue = (issue: Issue): Project => {
  return useProjectV1Store().getProjectByName(
    `${projectNamePrefix}${extractProjectResourceName(issue.name)}`
  );
};

export const useInstanceForTask = (task: Task) => {
  let instanceName: string = "";
  switch (task.type) {
    case Task_Type.DATABASE_CREATE:
      instanceName = task.target;
      break;
    case Task_Type.DATABASE_SCHEMA_UPDATE:
    case Task_Type.DATABASE_SCHEMA_UPDATE_SDL:
    case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST:
    case Task_Type.DATABASE_DATA_UPDATE:
    case Task_Type.DATABASE_EXPORT:
      instanceName = extractDatabaseResourceName(task.target).instance;
      break;
    default:
  }

  if (!isValidInstanceName(instanceName)) {
    return {
      instance: {
        ...unknownInstance(),
        name: instanceName,
      },
      ready: true,
    };
  }

  return useInstanceResourceByName(instanceName);
};

export const mockDatabase = (projectEntity: Project, database: string) => {
  // Database not found, it's probably NOT_FOUND (maybe dropped actually)
  // Mock a database using all known resources
  const db = unknownDatabase();
  db.project = projectEntity.name;

  db.name = database;
  const { instance, databaseName } = extractDatabaseResourceName(db.name);
  db.databaseName = databaseName;
  db.instance = instance;
  const { instance: instanceFromStore } = useInstanceResourceByName(instance);
  // Create InstanceResource from the instance data
  const instanceData = instanceFromStore.value;
  db.instanceResource = create(InstanceResourceSchema, {
    name: instanceData.name,
    engine: instanceData.engine,
    title: instanceData.title,
    activation: instanceData.activation ?? true,
    dataSources: instanceData.dataSources ?? [],
    environment: instanceData.environment,
    engineVersion: instanceData.engineVersion ?? "",
  });
  db.environment = db.instanceResource.environment;
  db.effectiveEnvironment = db.instanceResource.environment;
  db.effectiveEnvironmentEntity = useEnvironmentV1Store().getEnvironmentByName(
    db.instanceResource.environment ?? ""
  );
  db.state = State.DELETED;
  return db;
};

export const extractCoreDatabaseInfoFromDatabaseCreateTask = (
  project: Project,
  task: Task
) => {
  const coreDatabaseInfo = (
    instanceName: string,
    databaseName: string
  ): ComposedDatabase => {
    const name = `${instanceName}/databases/${databaseName}`;
    const maybeExistedDatabase = useDatabaseV1Store().getDatabaseByName(name);
    if (isValidDatabaseName(maybeExistedDatabase.name)) {
      return maybeExistedDatabase;
    }

    const environmentStore = useEnvironmentV1Store();
    const { instance: instanceFromStore } =
      useInstanceResourceByName(instanceName);
    // Create InstanceResource from the instance data
    const instanceData = instanceFromStore.value;
    const instanceResource = create(InstanceResourceSchema, {
      name: instanceData.name,
      engine: instanceData.engine,
      title: instanceData.title,
      activation: instanceData.activation ?? true,
      dataSources: instanceData.dataSources ?? [],
      environment: instanceData.environment,
      engineVersion: instanceData.engineVersion ?? "",
    });
    const effectiveEnvironmentEntity = environmentStore.getEnvironmentByName(
      instanceResource.environment ?? ""
    );
    return {
      ...unknownDatabase(),
      name,
      databaseName,
      instance: instanceName,
      project: project.name,
      projectEntity: project,
      effectiveEnvironment: instanceResource.environment,
      effectiveEnvironmentEntity: effectiveEnvironmentEntity,
      instanceResource,
    };
  };

  if (task.payload?.case === "databaseCreate") {
    const databaseName = task.payload.value.database;
    const instance = task.target;
    return coreDatabaseInfo(instance, databaseName);
  }

  return unknownDatabase();
};

export const specForTask = (plan: Plan | undefined, task: Task) => {
  if (!plan) return undefined;
  return (plan.specs || []).find((spec) => spec.id === task.specId);
};

export const stageForTask = (issue: ComposedIssue, task: Task) => {
  const rollout = issue.rolloutEntity;
  return rollout?.stages.find(
    (stage) => stage.tasks.findIndex((t) => t.name === task.name) >= 0
  );
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

export const isUnfinishedResolvedTask = (issue: ComposedIssue | undefined) => {
  if (!issue) {
    return false;
  }
  if (!isValidIssueName(issue.name)) {
    return false;
  }
  if (issue.status !== IssueStatus.DONE) {
    return false;
  }
  return flattenTaskV1List(issue.rolloutEntity).some((task) => {
    return ![Task_Status.DONE, Task_Status.SKIPPED].includes(task.status);
  });
};
