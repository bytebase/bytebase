import Emittery from "emittery";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { targetsForSpec } from "@/components/Plan/logic";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { type Plan, type Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

export const KEY = Symbol(
  "bb.plan.setting.instance-role"
) as InjectionKey<InstanceRoleSettingContext>;

export const useInstanceRoleSettingContext = () => {
  return inject(KEY)!;
};

export const provideInstanceRoleSettingContext = (refs: {
  isCreating: Ref<boolean>;
  project: Ref<Project>;
  plan: Ref<Plan>;
  selectedSpec: Ref<Plan_Spec | undefined>;
  allowChange: ComputedRef<boolean>;
}) => {
  const databaseStore = useDatabaseV1Store();

  const { isCreating, plan, selectedSpec, allowChange } = refs;

  const events = new Emittery<{
    update: never;
  }>();

  const selectedRole = ref<string | undefined>(undefined);

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

    // Only show for PostgreSQL databases
    const allDatabasesArePostgres = databases.value.every(
      (db) => db.instanceResource.engine === Engine.POSTGRES
    );
    if (!allDatabasesArePostgres) return false;

    // Check if this is a database change spec.
    if (selectedSpec.value.config?.case !== "changeDatabaseConfig") {
      return false;
    }
    const config = selectedSpec.value.config.value;
    // Show for MIGRATE type, but not SDL.
    return !config.release;
  });

  const context = {
    isCreating,
    selectedSpec,
    plan,
    shouldShow,
    allowChange,
    selectedRole,
    databases,
    events,
  };

  provide(KEY, context);

  return context;
};

type InstanceRoleSettingContext = ReturnType<
  typeof provideInstanceRoleSettingContext
>;
