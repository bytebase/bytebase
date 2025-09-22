import Emittery from "emittery";
import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { targetsForSpec } from "@/components/Plan/logic";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
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
import { flattenTaskV1List, instanceV1SupportsTransactionMode } from "@/utils";

export const KEY = Symbol(
  "bb.plan.setting.transaction-mode"
) as InjectionKey<TransactionModeSettingContext>;

export const useTransactionModeSettingContext = () => {
  return inject(KEY)!;
};

export const provideTransactionModeSettingContext = (refs: {
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

  const transactionMode = ref<"on" | "off">("on");

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

    // Check if all databases support transaction mode
    const allDatabasesSupportTransactionMode = databases.value.every((db) =>
      instanceV1SupportsTransactionMode(db.instanceResource.engine)
    );
    if (!allDatabasesSupportTransactionMode) return false;

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
    transactionMode,
    databases,
    events,
  };

  provide(KEY, context);

  return context;
};

type TransactionModeSettingContext = ReturnType<
  typeof provideTransactionModeSettingContext
>;
