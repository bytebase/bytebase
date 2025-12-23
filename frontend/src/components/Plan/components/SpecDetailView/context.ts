import { head, uniq } from "lodash-es";
import { computed } from "vue";
import { useRoute } from "vue-router";
import { isValidDatabaseGroupName } from "@/types";
import { usePlanContext } from "../../logic";

export const DEFAULT_VISIBLE_TARGETS = 20;

export const useSelectedSpec = () => {
  const route = useRoute();
  const { plan } = usePlanContext();

  const selectedSpec = computed(() => {
    if (plan.value.specs.length === 0) {
      throw new Error("No specs found in the plan.");
    }

    const specId = route.params.specId as string | undefined;

    // If no specId in route, check if this is a database export plan
    if (!specId) {
      const isExportPlan = plan.value.specs.every(
        (spec) => spec.config.case === "exportDataConfig"
      );
      if (isExportPlan) {
        // For export plans, return the first (and typically only) spec
        return plan.value.specs[0];
      }

      throw new Error("Spec ID is required in the route parameters.");
    }

    const foundSpec =
      plan.value.specs.find((spec) => spec.id === specId) ||
      head(plan.value.specs);
    if (!foundSpec) {
      throw new Error(`Spec with ID ${specId} not found in the plan.`);
    }
    return foundSpec;
  });

  const targets = computed(() => {
    if (!selectedSpec.value) return [];
    if (
      selectedSpec.value.config.case === "changeDatabaseConfig" ||
      selectedSpec.value.config.case === "exportDataConfig"
    ) {
      return selectedSpec.value.config.value.targets;
    }
    return [];
  });

  const getDatabasesForGroup = async (groupName: string): Promise<string[]> => {
    const { useDBGroupStore } = await import("@/store");
    const { DatabaseGroupView } = await import(
      "@/types/proto-es/v1/database_group_service_pb"
    );
    const dbGroupStore = useDBGroupStore();
    try {
      const dbGroup = await dbGroupStore.getOrFetchDBGroupByName(groupName, {
        view: DatabaseGroupView.FULL,
        silent: true,
      });
      return dbGroup.matchedDatabases?.map((db) => db.name) ?? [];
    } catch {
      return [];
    }
  };

  const getDatabaseTargets = async (
    targets: string[]
  ): Promise<{ databaseTargets: string[]; dbGroupTargets: string[] }> => {
    const databaseTargets: string[] = [];
    const dbGroupTargets: string[] = [];

    for (const target of targets) {
      if (isValidDatabaseGroupName(target)) {
        dbGroupTargets.push(target);
        // Fetch live databases from the database group
        const mappedDatabases = await getDatabasesForGroup(target);
        databaseTargets.push(...mappedDatabases);
      } else {
        databaseTargets.push(target);
      }
    }

    return { databaseTargets: uniq(databaseTargets), dbGroupTargets };
  };

  return { selectedSpec, targets, getDatabaseTargets };
};
