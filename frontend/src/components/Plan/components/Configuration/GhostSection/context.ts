import Emittery from "emittery";
import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, unref } from "vue";
import { targetsForSpec } from "@/components/Plan/logic";
import { useCurrentUserV1, extractUserId, useDatabaseV1Store } from "@/store";
import { isValidDatabaseName, type ComposedProject } from "@/types";
import { convertEngineToNew } from "@/utils/v1/common-conversions";
import { Issue, IssueStatus } from "@/types/proto/v1/issue_service";
import type { Plan, Plan_Spec } from "@/types/proto/v1/plan_service";
import {
  Task,
  Task_Status,
  type Rollout,
} from "@/types/proto/v1/rollout_service";
import {
  flattenTaskV1List,
  hasProjectPermissionV2,
  isNullOrUndefined,
} from "@/utils";
import {
  allowGhostForSpec,
  getGhostEnabledForSpec,
  GHOST_AVAILABLE_ENGINES,
} from "./common";

export const KEY = Symbol(
  "bb.plan.setting.gh-ost"
) as InjectionKey<GhostSettingContext>;

export const useGhostSettingContext = () => {
  return inject(KEY)!;
};

export const provideGhostSettingContext = (refs: {
  isCreating: Ref<boolean>;
  project: Ref<ComposedProject>;
  plan: Ref<Plan>;
  selectedSpec: Ref<Plan_Spec | undefined>;
  selectedTask?: Ref<Task | undefined>;
  issue?: Ref<Issue | undefined>;
  rollout?: Ref<Rollout | undefined>;
}) => {
  const currentUser = useCurrentUserV1();
  const databaseStore = useDatabaseV1Store();

  const {
    isCreating,
    project,
    plan,
    selectedSpec,
    selectedTask,
    issue,
    rollout,
  } = refs;

  const events = new Emittery<{
    update: never;
  }>();

  const databases = computed(() => {
    const targets = selectedSpec.value
      ? targetsForSpec(selectedSpec.value)
      : [];
    return targets
      .map((target) => databaseStore.getDatabaseByName(target))
      .filter((db) => isValidDatabaseName(db.name));
  });

  const shouldShow = computed(() => {
    return (
      selectedSpec.value &&
      allowGhostForSpec(selectedSpec.value) &&
      databases.value.every((db) =>
        GHOST_AVAILABLE_ENGINES.includes(convertEngineToNew(db.instanceResource.engine))
      ) &&
      !isNullOrUndefined(getGhostEnabledForSpec(selectedSpec.value))
    );
  });

  const allowChange = computed(() => {
    // Allow toggle pre-backup when creating.
    if (isCreating.value) {
      return true;
    }

    // If issue is not open, disallow.
    if (issue?.value && issue.value.status !== IssueStatus.OPEN) {
      return false;
    }

    let task: Task | undefined;
    if (selectedTask?.value) {
      task = selectedTask.value;
    } else if (rollout?.value) {
      const tasks = flattenTaskV1List(rollout.value);
      task = tasks.find((t) => t.specId === selectedSpec.value?.id);
    }
    // If task of the spec is running/done/etc..., disallow.
    if (
      task &&
      [
        Task_Status.PENDING,
        Task_Status.RUNNING,
        Task_Status.DONE,
        Task_Status.SKIPPED,
      ].includes(task.status)
    ) {
      return false;
    }

    // Allowed to the plan/issue creator.
    if (currentUser.value.email === extractUserId(unref(plan).creator)) {
      return true;
    }

    // Allowed to the permission holder.
    if (hasProjectPermissionV2(project.value, "bb.plans.update")) {
      return true;
    }

    return false;
  });

  const enabled = computed(() => {
    return (
      (selectedSpec.value && getGhostEnabledForSpec(selectedSpec.value)) ||
      false
    );
  });

  const context = {
    isCreating,
    selectedSpec,
    selectedTask,
    plan,
    shouldShow,
    allowChange,
    enabled,
    databases,
    events,
  };

  provide(KEY, context);

  return context;
};

type GhostSettingContext = ReturnType<typeof provideGhostSettingContext>;
