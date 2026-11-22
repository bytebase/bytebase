import { useEffect, useMemo, useState } from "react";
import { useAppStore } from "@/stores/app";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { isValidDatabaseName } from "@/types/v1/database";
import {
  type Environment,
  isProtectedEnvironment,
  resolveEnvironment,
} from "@/types/v1/environment";
import { extractDatabaseResourceName } from "@/utils/v1/database";

type TargetResources = {
  databasesByName: Record<string, Database | undefined>;
  environmentList: Environment[];
};

// The environment a database is in: its own, else its instance's.
export const databaseEnvironmentName = (
  database: Database | undefined
): string =>
  database?.effectiveEnvironment ||
  database?.instanceResource?.environment ||
  "";

// A target shaped as one database name, under a workspace instance or a
// project instance, with nothing before or after it. A database group or any
// longer resource name fails.
export const isPlainDatabaseName = (target: string): boolean =>
  isValidDatabaseName(target) &&
  extractDatabaseResourceName(target).database === target;

// The production environments a review finding's databases include, the ones
// the workspace marked "production environment", in the workspace's
// environment order. Without the environment-tier feature none counts.
export const productionEnvironmentsOf = (
  targets: readonly string[],
  resources: TargetResources,
  hasEnvTierFeature: boolean
): Environment[] => {
  const names = new Set(
    targets.flatMap((target) => {
      const name = databaseEnvironmentName(resources.databasesByName[target]);
      return name ? [name] : [];
    })
  );
  return resources.environmentList.filter(
    (environment) =>
      names.has(environment.name) &&
      isProtectedEnvironment(environment, hasEnvTierFeature)
  );
};

// A finding's databases that share an environment. A loaded database with no
// environment is unassigned; a database that could not be read is unknown, so
// a database that may be in production is never shown as having none.
export type TargetGroup =
  | { kind: "environment"; environment: Environment; targets: string[] }
  | { kind: "unassigned"; targets: string[] }
  | { kind: "unknown"; targets: string[] };

export const targetGroupKey = (group: TargetGroup): string =>
  group.kind === "environment" ? group.environment.name : group.kind;

const byDatabaseName = (a: string, b: string): number =>
  extractDatabaseResourceName(a).databaseName.localeCompare(
    extractDatabaseResourceName(b).databaseName
  ) || a.localeCompare(b);

// Groups targets by environment, closest to production first: production
// environments, then the rest from the last rollout stage back (the
// workspace's environment order is the rollout order), then environments the
// workspace no longer lists, then unassigned, then unknown. Each group is
// ordered by database name, then by full name.
export const groupTargetsByEnvironment = (
  targets: readonly string[],
  resources: TargetResources,
  hasEnvTierFeature: boolean
): TargetGroup[] => {
  const environmentTargets = new Map<string, string[]>();
  const unassigned: string[] = [];
  const unknown: string[] = [];
  for (const target of new Set(targets)) {
    const database = isPlainDatabaseName(target)
      ? resources.databasesByName[target]
      : undefined;
    if (!database) {
      unknown.push(target);
      continue;
    }
    const name = databaseEnvironmentName(database);
    if (!name) {
      unassigned.push(target);
      continue;
    }
    environmentTargets.set(name, [
      ...(environmentTargets.get(name) ?? []),
      target,
    ]);
  }

  const listed = new Map(
    resources.environmentList.map((environment) => [
      environment.name,
      environment,
    ])
  );
  const rank = (name: string) => {
    const environment = listed.get(name);
    if (!environment) return [2, 0];
    return [
      isProtectedEnvironment(environment, hasEnvTierFeature) ? 0 : 1,
      -environment.order,
    ];
  };
  const environmentNames = [...environmentTargets.keys()].sort((a, b) => {
    const [ra, oa] = rank(a);
    const [rb, ob] = rank(b);
    return ra - rb || oa - ob || a.localeCompare(b);
  });

  const groups: TargetGroup[] = environmentNames.map((name) => ({
    kind: "environment",
    environment:
      listed.get(name) ?? resolveEnvironment(name, resources.environmentList),
    targets: (environmentTargets.get(name) ?? []).sort(byDatabaseName),
  }));
  if (unassigned.length > 0) {
    groups.push({
      kind: "unassigned",
      targets: unassigned.sort(byDatabaseName),
    });
  }
  if (unknown.length > 0) {
    groups.push({ kind: "unknown", targets: unknown.sort(byDatabaseName) });
  }
  return groups;
};

// Names some card has asked for and that have no answer yet, each with the
// request that will answer it. Finding cards mount together, and the store's
// batch fetch neither joins a request in flight nor gathers names across
// callers, so the cards that ask in one tick share one request, and a name is
// asked for once at a time. A failed name is asked for again by the next card,
// as the store expects.
const pendingDatabaseNames = new Map<string, Promise<void>>();
let queuedBatch: { names: Set<string>; done: Promise<void> } | undefined;

// Resolves once every uncached name has been answered, including names another
// card's request holds.
export const requestMissingDatabases = (
  names: readonly string[],
  isCached: (name: string) => boolean,
  fetch: (names: string[]) => Promise<unknown>
): Promise<void> => {
  const waits = new Set<Promise<void>>();
  for (const name of names) {
    if (isCached(name)) continue;
    const pending = pendingDatabaseNames.get(name);
    if (pending) {
      waits.add(pending);
      continue;
    }
    if (!queuedBatch) {
      const batch = new Set<string>();
      const done = Promise.resolve().then(async () => {
        queuedBatch = undefined;
        try {
          await fetch([...batch]);
        } finally {
          for (const batched of batch) pendingDatabaseNames.delete(batched);
        }
      });
      queuedBatch = { names: batch, done };
    }
    queuedBatch.names.add(name);
    pendingDatabaseNames.set(name, queuedBatch.done);
    waits.add(queuedBatch.done);
  }
  return Promise.all(waits).then(() => undefined);
};

// Asks for the databases among targets that are not loaded yet, and says
// whether every one of them has been loaded or answered.
export const useTargetDatabases = (targets: readonly string[]): boolean => {
  const databasesByName = useAppStore((state) => state.databasesByName);
  const key = targets.filter(isPlainDatabaseName).join("\n");
  const [answeredKey, setAnsweredKey] = useState<string>();
  useEffect(() => {
    const names = key ? key.split("\n") : [];
    let cancelled = false;
    const store = useAppStore.getState();
    void requestMissingDatabases(
      names,
      (name) => Boolean(useAppStore.getState().databasesByName[name]),
      (missing) => store.batchGetOrFetchDatabases(missing, true)
    )
      .catch(() => {
        // A database that cannot be read is listed as unknown.
      })
      .finally(() => {
        if (!cancelled) setAnsweredKey(key);
      });
    return () => {
      cancelled = true;
    };
  }, [key]);
  const allLoaded = useMemo(
    () =>
      !key || key.split("\n").every((name) => Boolean(databasesByName[name])),
    [key, databasesByName]
  );
  return allLoaded || answeredKey === key;
};
