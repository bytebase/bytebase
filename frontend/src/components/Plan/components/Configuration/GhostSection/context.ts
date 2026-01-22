import Emittery from "emittery";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, provide } from "vue";
import { targetsForSpec } from "@/components/Plan/logic";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { getInstanceResource } from "@/utils";
import { GHOST_AVAILABLE_ENGINES } from "./common";

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
    if (!selectedSpec.value) return false;

    // Check if all databases support ghost (engine check)
    const allDatabasesSupportGhost = databases.value.every((db) =>
      GHOST_AVAILABLE_ENGINES.includes(getInstanceResource(db).engine)
    );
    if (!allDatabasesSupportGhost) return false;

    // Check if this is a change database config
    if (selectedSpec.value.config?.case !== "changeDatabaseConfig") {
      return false;
    }
    const config = selectedSpec.value.config.value;
    // Show for sheet-based migrations only (not release-based).
    // Release-based migrations don't support ghost configuration.
    return !config.release;
  });

  const context = {
    isCreating,
    selectedSpec,
    plan,
    shouldShow,
    allowChange,
    databases,
    events,
  };

  provide(KEY, context);

  return context;
};

type GhostSettingContext = ReturnType<typeof provideGhostSettingContext>;
