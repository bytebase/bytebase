import Emittery from "emittery";
import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { targetsForSpec } from "@/components/Plan/logic";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { type Plan, type Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { instanceV1SupportsTransactionMode } from "@/utils";

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
  readonly?: Ref<boolean>;
}) => {
  const databaseStore = useDatabaseV1Store();

  const { isCreating, plan, selectedSpec, issue, readonly } = refs;

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
    const config = selectedSpec.value.config.value;
    // Show for sheet-based migrations only (not release-based).
    // Release-based migrations don't support transaction mode configuration.
    return !config.release;
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

    // Disallow changes if the plan has started rollout.
    if (plan.value.hasRollout) {
      return false;
    }

    // If issue is not open, disallow
    if (issue?.value && issue.value.status !== IssueStatus.OPEN) {
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
