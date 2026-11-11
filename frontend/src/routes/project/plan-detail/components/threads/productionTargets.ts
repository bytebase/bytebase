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

// Names some card has asked for and that have no answer yet. Finding cards
// mount together and share targets, and the store's batch fetch does not join
// a request already in flight, so each name is asked for once at a time. A
// failed name is asked for again by the next card, as the store expects.
const pendingDatabaseNames = new Set<string>();

export const requestMissingDatabases = async (
  names: readonly string[],
  isCached: (name: string) => boolean,
  fetch: (names: string[]) => Promise<unknown>
): Promise<void> => {
  const missing = names.filter(
    (name) => !isCached(name) && !pendingDatabaseNames.has(name)
  );
  if (missing.length === 0) return;
  for (const name of missing) pendingDatabaseNames.add(name);
  try {
    await fetch(missing);
  } finally {
    for (const name of missing) pendingDatabaseNames.delete(name);
  }
};
