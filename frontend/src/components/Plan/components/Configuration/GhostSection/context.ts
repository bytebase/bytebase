import Emittery from "emittery";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, provide } from "vue";
import { targetsForSpec } from "@/components/Plan/logic";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { isNullOrUndefined } from "@/utils";
import { GHOST_AVAILABLE_ENGINES, getGhostEnabledForSpec } from "./common";

export const KEY = Symbol(
  "bb.plan.setting.gh-ost"
) as InjectionKey<GhostSettingContext>;

export const useGhostSettingContext = () => {
  return inject(KEY)!;
};

export const provideGhostSettingContext = (refs: {
  isCreating: Ref<boolean>;
  plan: Ref<Plan>;
  selectedSpec: Ref<Plan_Spec | undefined>;
  allowChange: ComputedRef<boolean>;
}) => {
  const databaseStore = useDatabaseV1Store();

  const { isCreating, plan, selectedSpec, allowChange } = refs;

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
