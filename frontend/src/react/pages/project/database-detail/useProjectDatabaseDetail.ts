import { useEffect, useMemo, useState } from "react";
import { router } from "@/react/router";
import {
  PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
} from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { unknownDatabase } from "@/types/v1/database";
import { isDefaultProject } from "@/types/v1/project";
import { getInstanceResource, instanceV1HasAlterSchema } from "@/utils";
import { extractProjectResourceName } from "@/utils/v1/project";

export interface UseProjectDatabaseDetailOptions {
  projectId: string;
  instanceId: string;
  databaseName: string;
  routeName?: string;
  hash?: string;
  query?: Record<string, string | undefined>;
  changelogId?: string;
  revisionId?: string;
}

export function useProjectDatabaseDetail({
  projectId,
  instanceId,
  databaseName,
  routeName,
  hash,
  query,
  changelogId,
  revisionId,
}: UseProjectDatabaseDetailOptions) {
  const getOrFetchDatabaseMetadata = useAppStore(
    (s) => s.getOrFetchDatabaseMetadata
  );
  const databasesByName = useAppStore((s) => s.databasesByName);
  const fullDatabaseName = `instances/${instanceId}/databases/${databaseName}`;
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
          const name =
            routeName === PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL
              ? PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL
              : routeName === PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL
                ? PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL
                : PROJECT_V1_ROUTE_DATABASE_DETAIL;
          void router.replace({
            name,
            params: {
              projectId: canonicalProjectId,
              instanceId,
              databaseName,
              ...(changelogId ? { changelogId } : {}),
              ...(revisionId ? { revisionId } : {}),
            },
            hash,
            query,
          });
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
  }, [
    changelogId,
    databaseName,
    getOrFetchDatabaseMetadata,
    fullDatabaseName,
    instanceId,
    projectId,
    revisionId,
    routeName,
  ]);

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
