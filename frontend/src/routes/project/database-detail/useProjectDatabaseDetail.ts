import { useEffect, useMemo, useState } from "react";
import { router } from "@/app/router";
import { useAppStore } from "@/stores/app";
import { unknownDatabase } from "@/types/v1/database";
import { isDefaultProject } from "@/types/v1/project";
import {
  autoDatabaseRoute,
  getInstanceResource,
  instanceV1HasAlterSchema,
} from "@/utils";
import { extractProjectResourceName } from "@/utils/v1/project";

export interface UseProjectDatabaseDetailOptions {
  parent: string;
  projectId: string;
  instanceId: string;
  databaseName: string;
}

export function useProjectDatabaseDetail({
  parent,
  projectId,
  instanceId,
  databaseName,
}: UseProjectDatabaseDetailOptions) {
  const getOrFetchDatabaseMetadata = useAppStore(
    (s) => s.getOrFetchDatabaseMetadata
  );
  const databasesByName = useAppStore((s) => s.databasesByName);
  const fullDatabaseName = `${parent}/databases/${databaseName}`;
  const database = useMemo(
    () => databasesByName[fullDatabaseName] ?? unknownDatabase(),
    [databasesByName, fullDatabaseName]
  );
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    setLoading(true);
    void useAppStore
      .getState()
      .getOrFetchDatabaseByName(fullDatabaseName)
      .then(async (db) => {
        try {
          await getOrFetchDatabaseMetadata({
            database: db.name,
            silent: true,
          });
        } catch {
          // Permission errors should not block page rendering.
        }

        const canonicalProjectId = extractProjectResourceName(db.project);
        if (canonicalProjectId !== projectId) {
          void router.replace(autoDatabaseRoute(db));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [fullDatabaseName, getOrFetchDatabaseMetadata, projectId]);

  const allowAlterSchema = useMemo(() => {
    return database
      ? instanceV1HasAlterSchema(getInstanceResource(database))
      : false;
  }, [database]);

  return {
    projectName: `projects/${projectId}`,
    instanceName: `instances/${instanceId}`,
    databaseName: fullDatabaseName,
    database,
    loading,
    ready: !!database && !loading,
    allowAlterSchema,
    isDefaultProject: database ? isDefaultProject(database.project) : false,
  };
}
