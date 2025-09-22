import Emittery from "emittery";
import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { targetsForSpec } from "@/components/Plan/logic";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ChangeDatabaseConfig_Type,
  type Plan,
  type Plan_Spec,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { flattenTaskV1List } from "@/utils";
import type { IsolationLevel } from "../../StatementSection/directiveUtils";

export const KEY = Symbol(
  "bb.plan.setting.isolation-level"
) as InjectionKey<IsolationLevelSettingContext>;

type IsolationLevelSettingContext = ReturnType<
  typeof provideIsolationLevelSettingContext
>;

export const useIsolationLevelSettingContext = () => {
  return inject(KEY)!;
};

export const provideIsolationLevelSettingContext = (refs: {
  isCreating: Ref<boolean>;
  project: Ref<Project>;
  plan: Ref<Plan>;
  selectedSpec: Ref<Plan_Spec | undefined>;
  issue?: Ref<Issue | undefined>;
  rollout?: Ref<Rollout | undefined>;
  readonly?: Ref<boolean>;
}) => {
  const databaseStore = useDatabaseV1Store();

  const { isCreating, plan, selectedSpec, issue, rollout, readonly } = refs;

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
    return [
      Plan_ChangeDatabaseConfig_Type.DATA,
      Plan_ChangeDatabaseConfig_Type.MIGRATE,
    ].includes(selectedSpec.value.config.value.type);
  });

  const allowChange = computed(() => {
    // If readonly mode, disallow changes
    if (readonly?.value) {
      return false;
    }

    // Allow changes when creating
    if (isCreating.value) {
      return true;
    }

    // If issue is not open, disallow
    if (issue?.value && issue.value.status !== IssueStatus.OPEN) {
      return false;
    }

    const tasks = flattenTaskV1List(rollout?.value) || [];
    // If any task is running/done/etc, disallow
    if (
      tasks.some((task) => {
        return [
          Task_Status.PENDING,
          Task_Status.RUNNING,
          Task_Status.DONE,
          Task_Status.SKIPPED,
        ].includes(task.status);
      })
    ) {
      return false;
    }

    return true;
  });

  const context = {
    isCreating,
    selectedSpec,
    plan,
    shouldShow,
    allowChange,
    isolationLevel,
    databases,
    events,
  };

  provide(KEY, context);

  return context;
};
