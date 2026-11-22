import { useEffect, useMemo, useState } from "react";
import { useAppStore } from "@/stores/app";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

// What GitOps deploys to: one database group, or one or more databases.
export type DeployPick =
  | { kind: "group"; name: string }
  | { kind: "databases"; names: string[] };

export interface DeployStage {
  // The environment resource name, such as "environments/prod".
  environment: string;
  // The environment ID, such as "prod": the deploy job's target stage.
  id: string;
  databases: Database[];
}

export interface DeployTargets {
  // One stage per environment, in the order the rollout deploys them.
  stages: DeployStage[];
  // Databases with no environment. The rollout skips them.
  unassigned: Database[];
  // Every picked database, with or without an environment.
  total: number;
  loading: boolean;
}

const environmentOf = (database: Database) =>
  database.effectiveEnvironment ?? "";

// Groups databases the way the rollout does: environments in the workspace
// order, an environment missing from that order first, and databases with no
// environment left out.
export function groupByEnvironment(
  databases: Database[],
  environmentOrder: string[]
): Pick<DeployTargets, "stages" | "unassigned"> {
  const byEnvironment = new Map<string, Database[]>();
  const unassigned: Database[] = [];
  for (const database of databases) {
    const environment = environmentOf(database);
    if (!environment) {
      unassigned.push(database);
      continue;
    }
    const list = byEnvironment.get(environment) ?? [];
    list.push(database);
    byEnvironment.set(environment, list);
  }
  const toStage = (environment: string): DeployStage => ({
    environment,
    id: environment.replace(/^environments\//, ""),
    databases: byEnvironment.get(environment) ?? [],
  });
  const ordered = environmentOrder.filter((e) => byEnvironment.has(e));
  const unknown = [...byEnvironment.keys()].filter(
    (e) => !environmentOrder.includes(e)
  );
  return {
    stages: [...unknown, ...ordered].map(toStage),
    unassigned,
  };
}

// The pick as a string, so the effect below reruns only when it changes.
// Resource names never contain a comma.
const pickKey = (pick: DeployPick | undefined) => {
  if (!pick) return "";
  if (pick.kind === "group") return pick.name ? `group:${pick.name}` : "";
  return pick.names.length > 0 ? `databases:${pick.names.join(",")}` : "";
};

const namesOf = async (key: string): Promise<string[]> => {
  const store = useAppStore.getState();
  if (key.startsWith("group:")) {
    const group = await store.getOrFetchDBGroupByName(
      key.slice("group:".length),
      { view: DatabaseGroupView.FULL, silent: true }
    );
    return group.matchedDatabases.map((database) => database.name);
  }
  return key.slice("databases:".length).split(",");
};

// Loads the picked databases, expanding a group to the databases it matches,
// and groups them by environment.
export function useDeployTargets(pick: DeployPick | undefined): DeployTargets {
  const environmentList = useAppStore((s) => s.environmentList);
  const key = pickKey(pick);
  const [loaded, setLoaded] = useState<{ key: string; databases: Database[] }>({
    key: "",
    databases: [],
  });

  useEffect(() => {
    if (key === "") return;
    let cancelled = false;
    const load = async () => {
      const names = await namesOf(key);
      const databases =
        names.length > 0
          ? await useAppStore.getState().batchGetOrFetchDatabases(names, true)
          : [];
      if (!cancelled) setLoaded({ key, databases });
    };
    void load().catch(() => {
      if (!cancelled) setLoaded({ key, databases: [] });
    });
    return () => {
      cancelled = true;
    };
  }, [key]);

  return useMemo(() => {
    const loading = key !== "" && loaded.key !== key;
    const databases = loaded.key === key ? loaded.databases : [];
    const { stages, unassigned } = groupByEnvironment(
      databases,
      environmentList.map((environment) => environment.name)
    );
    return { stages, unassigned, total: databases.length, loading };
  }, [key, loaded, environmentList]);
}
