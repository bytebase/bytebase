import Emittery from "emittery";
import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { targetsForSpec } from "@/components/Plan/logic";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type {
  Plan,
  Plan_Spec,
  Plan_ChangeDatabaseConfig,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Task, Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { flattenTaskV1List } from "@/utils";
import type { IsolationLevel } from "../../StatementSection/directiveUtils";

export const KEY = Symbol(
  "bb.plan.setting.isolation-level"
) as InjectionKey<IsolationLevelSettingContext>;

export type IsolationLevelSettingContext = {
  allowChange: Ref<boolean>;
  isolationLevel: Ref<IsolationLevel | undefined>;
  databases: Ref<any[]>;
  shouldShow: Ref<boolean>;
  events: Emittery<{
    update: never;
  }>;
};

export const useIsolationLevelSettingContext = () => {
  return inject(KEY)!;
};

export const provideIsolationLevelSettingContext = (refs: {
  isCreating: Ref<boolean>;
  project: Ref<Project>;
  plan: Ref<Plan>;
  selectedSpec: Ref<Plan_Spec | undefined>;
  selectedTask?: Ref<Task | undefined>;
  issue?: Ref<Issue | undefined>;
  rollout?: Ref<Rollout | undefined>;
  readonly?: Ref<boolean>;
}) => {
  const databaseStore = useDatabaseV1Store();

  const { isCreating, selectedSpec, selectedTask, issue, rollout, readonly } =
    refs;

  const events = new Emittery<{
    update: never;
  }>();

  const isolationLevel = ref<IsolationLevel | undefined>(undefined);

  const databases = computed(() => {
    const targets = selectedSpec.value
      ? targetsForSpec(selectedSpec.value)
      : [];
    return targets
      .map((target) => databaseStore.getDatabaseByName(target))
      .filter((db) => isValidDatabaseName(db.name));
  });

  const shouldShow = computed(() => {
    if (!selectedSpec.value) return false;

    // Only show for MySQL/MariaDB/TiDB for now
    const supportedEngines = [Engine.MYSQL, Engine.MARIADB, Engine.TIDB];
    const allDatabasesSupported = databases.value.every((db) =>
      supportedEngines.includes(db.instanceResource.engine)
    );
    if (!allDatabasesSupported) return false;

    // Check if this is a DDL or DML change
    if (selectedSpec.value.config?.case !== "changeDatabaseConfig") {
      return false;
    }

    // Get the task type from spec or selected task
    let taskType: Task_Type | undefined;
    if (selectedTask?.value) {
      taskType = selectedTask.value.type;
    } else if (rollout?.value) {
      const tasks = flattenTaskV1List(rollout.value);
      const task = tasks.find((t) => t.specId === selectedSpec.value?.id);
      taskType = task?.type;
    } else {
      // For creating mode, we can infer from the change type
      const config = selectedSpec.value.config
        .value as Plan_ChangeDatabaseConfig;
      // Check if it's a schema or data update based on the config
      taskType = config.sheet
        ? Task_Type.DATABASE_SCHEMA_UPDATE
        : Task_Type.DATABASE_DATA_UPDATE;
    }

    return (
      !!taskType &&
      [
        Task_Type.DATABASE_SCHEMA_UPDATE,
        Task_Type.DATABASE_DATA_UPDATE,
      ].includes(taskType)
    );
  });

  const allowChange = computed(() => {
    if (readonly?.value) return false;

    if (isCreating.value) return true;

    if (issue?.value?.status !== IssueStatus.OPEN) return false;

    if (selectedTask?.value) {
      return [Task_Status.NOT_STARTED, Task_Status.PENDING].includes(
        selectedTask.value.status
      );
    }

    if (rollout?.value) {
      const tasks = flattenTaskV1List(rollout.value);
      const task = tasks.find((t) => t.specId === selectedSpec.value?.id);
      if (!task) return false;
      return [Task_Status.NOT_STARTED, Task_Status.PENDING].includes(
        task.status
      );
    }

    return false;
  });

  const context: IsolationLevelSettingContext = {
    allowChange,
    isolationLevel,
    databases,
    shouldShow,
    events,
  };

  provide(KEY, context);

  return context;
};
