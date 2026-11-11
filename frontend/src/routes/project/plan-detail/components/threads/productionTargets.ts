import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  type Environment,
  isProtectedEnvironment,
} from "@/types/v1/environment";

// The production environments a review finding's databases include, the ones
// the workspace marked "production environment", in the workspace's
// environment order. Without the environment-tier feature none counts.
export const productionEnvironmentsOf = (
  targets: readonly string[],
  resources: {
    databasesByName: Record<string, Database | undefined>;
    environmentList: Environment[];
  },
  hasEnvTierFeature: boolean
): Environment[] => {
  const names = new Set(
    targets.flatMap((target) => {
      const name = resources.databasesByName[target]?.effectiveEnvironment;
      return name ? [name] : [];
    })
  );
  return resources.environmentList.filter(
    (environment) =>
      names.has(environment.name) &&
      isProtectedEnvironment(environment, hasEnvTierFeature)
  );
};
