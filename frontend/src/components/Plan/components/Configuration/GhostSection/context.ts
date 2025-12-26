import Emittery from "emittery";
import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, unref } from "vue";
import { targetsForSpec } from "@/components/Plan/logic";
import { extractUserId, useCurrentUserV1, useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2, isNullOrUndefined } from "@/utils";
import { GHOST_AVAILABLE_ENGINES, getGhostEnabledForSpec } from "./common";

export const KEY = Symbol(
  "bb.plan.setting.gh-ost"
) as InjectionKey<GhostSettingContext>;

export const useGhostSettingContext = () => {
  return inject(KEY)!;
};

export const provideGhostSettingContext = (refs: {
  isCreating: Ref<boolean>;
  project: Ref<Project>;
  plan: Ref<Plan>;
  selectedSpec: Ref<Plan_Spec | undefined>;
  issue?: Ref<Issue | undefined>;
  readonly?: Ref<boolean>;
}) => {
  const currentUser = useCurrentUserV1();
  const databaseStore = useDatabaseV1Store();

  const { isCreating, project, plan, selectedSpec, issue, readonly } = refs;

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
      databases.value.every((db) =>
        GHOST_AVAILABLE_ENGINES.includes(db.instanceResource.engine)
      ) &&
      !isNullOrUndefined(getGhostEnabledForSpec(selectedSpec.value))
    );
  });

  const allowChange = computed(() => {
    // If readonly mode, disallow changes
    if (readonly?.value) {
      return false;
    }

    // Allow toggle gh-ost when creating.
    if (isCreating.value) {
      return true;
    }

    // Disallow changes if the plan has started rollout.
    if (unref(plan).hasRollout) {
      return false;
    }

    // If issue is not open, disallow.
    if (issue?.value && issue.value.status !== IssueStatus.OPEN) {
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
