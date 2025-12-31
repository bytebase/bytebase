import { head, uniq } from "lodash-es";
import { computed, type InjectionKey, inject, provide, type Ref } from "vue";
import { useRoute } from "vue-router";
import { useDBGroupStore } from "@/store";
import { isValidDatabaseGroupName } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { usePlanContext } from "../../logic";

export const DEFAULT_VISIBLE_TARGETS = 20;

export const SELECTED_SPEC_INJECTION_KEY: InjectionKey<Ref<Plan_Spec>> =
  Symbol("selected-spec");

export const provideSelectedSpec = (spec: Ref<Plan_Spec>) => {
  provide(SELECTED_SPEC_INJECTION_KEY, spec);
};

export const useSelectedSpec = () => {
  const route = useRoute();
  const { plan } = usePlanContext();
  const dbGroupStore = useDBGroupStore();

  const injectedSpec = inject(SELECTED_SPEC_INJECTION_KEY, null);

  const selectedSpec = computed(() => {
    if (injectedSpec?.value) {
      return injectedSpec.value;
    }

    if (plan.value.specs.length === 0) {
      throw new Error("No specs found in the plan.");
    }

    const specId = route.params.specId as string | undefined;
    if (!specId) {
      // For export plans without specId in route, use first spec
      if (plan.value.specs.every((s) => s.config.case === "exportDataConfig")) {
        return plan.value.specs[0];
      }
      throw new Error("Spec ID is required in the route parameters.");
    }

    const foundSpec =
      plan.value.specs.find((s) => s.id === specId) ?? head(plan.value.specs);
    if (!foundSpec) {
      throw new Error(`Spec with ID ${specId} not found in the plan.`);
    }
    return foundSpec;
  });

  const targets = computed(() => {
    const config = selectedSpec.value?.config;
    if (
      config?.case === "changeDatabaseConfig" ||
      config?.case === "exportDataConfig"
    ) {
      return config.value.targets;
    }
    return [];
  });

  const getDatabaseTargets = async (
    targetList: string[]
  ): Promise<{ databaseTargets: string[]; dbGroupTargets: string[] }> => {
    const databaseTargets: string[] = [];
    const dbGroupTargets: string[] = [];

    for (const target of targetList) {
      if (isValidDatabaseGroupName(target)) {
        dbGroupTargets.push(target);
        try {
          const dbGroup = await dbGroupStore.getOrFetchDBGroupByName(target, {
            view: DatabaseGroupView.FULL,
            silent: true,
          });
          databaseTargets.push(
            ...(dbGroup.matchedDatabases?.map((db) => db.name) ?? [])
          );
        } catch {
          // Ignore errors fetching database group
        }
      } else {
        databaseTargets.push(target);
      }
    }

    return { databaseTargets: uniq(databaseTargets), dbGroupTargets };
  };

  return { selectedSpec, targets, getDatabaseTargets };
};
