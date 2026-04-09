import { useEffect, useMemo, useState } from "react";
import type { LocationQueryRaw } from "vue-router";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
} from "@/router/dashboard/projectV1";
import { useDatabaseV1Store, useDBSchemaV1Store } from "@/store";
import { isDefaultProject } from "@/types/v1/project";
import { getInstanceResource, instanceV1HasAlterSchema } from "@/utils";
import { extractProjectResourceName } from "@/utils/v1/project";

export interface UseProjectDatabaseDetailOptions {
  projectId: string;
  instanceId: string;
  databaseName: string;
  routeName?: string;
  hash?: string;
  query?: LocationQueryRaw;
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
  const databaseStore = useDatabaseV1Store();
  const dbSchemaStore = useDBSchemaV1Store();
  const fullDatabaseName = `instances/${instanceId}/databases/${databaseName}`;
  const database = useVueState(() =>
    databaseStore.getDatabaseByName(fullDatabaseName)
  );
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    setLoading(true);
    void databaseStore
      .getOrFetchDatabaseByName(fullDatabaseName)
      .then(async (db) => {
        try {
          await dbSchemaStore.getOrFetchDatabaseMetadata({
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
    databaseStore,
    dbSchemaStore,
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
