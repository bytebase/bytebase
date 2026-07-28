import { useEffect, useMemo } from "react";
import { useAppStore } from "@/stores/app";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { PlanChangeReferenceResources } from "../utils/changeReference";

export const usePlanChangeReferenceData = (
  specs: Plan_Spec[]
): PlanChangeReferenceResources => {
  const databasesByName = useAppStore((state) => state.databasesByName);
  const databaseGroupsByName = useAppStore((state) => state.dbGroupsByName);
  const environmentList = useAppStore((state) => state.environmentList);
  const targets = useMemo(() => collectTargets(specs), [specs]);
  const targetKey = targets.join("\n");

  useEffect(() => {
    let canceled = false;

    const hydrate = async () => {
      const databaseNames = new Set(targets.filter(isValidDatabaseName));
      const databaseGroupNames = targets.filter(isValidDatabaseGroupName);
      const groupResults = await Promise.allSettled(
        databaseGroupNames.map((name) =>
          useAppStore.getState().getOrFetchDBGroupByName(name, {
            silent: true,
            view: DatabaseGroupView.FULL,
          })
        )
      );
      if (canceled) {
        return;
      }

      for (const result of groupResults) {
        if (result.status !== "fulfilled") {
          continue;
        }
        for (const database of result.value.matchedDatabases ?? []) {
          databaseNames.add(database.name);
        }
      }
      if (databaseNames.size > 0) {
        await useAppStore
          .getState()
          .batchGetOrFetchDatabases([...databaseNames], true);
      }
    };

    void hydrate().catch(() => {
      // Resource-path labels remain usable when optional enrichment fails.
    });
    return () => {
      canceled = true;
    };
  }, [targetKey, targets]);

  return {
    databaseGroupsByName,
    databasesByName,
    environmentList,
  };
};

const collectTargets = (specs: Plan_Spec[]): string[] => {
  const targets = new Set<string>();
  for (const spec of specs) {
    if (
      spec.config.case !== "changeDatabaseConfig" &&
      spec.config.case !== "exportDataConfig"
    ) {
      continue;
    }
    for (const target of spec.config.value.targets ?? []) {
      targets.add(target);
    }
  }
  return [...targets];
};
