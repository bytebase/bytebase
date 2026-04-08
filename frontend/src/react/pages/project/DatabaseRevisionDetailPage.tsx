import { LoaderCircle } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { RevisionDetailPanel } from "@/react/components/revision";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
} from "@/router/dashboard/projectV1";
import { useDatabaseV1Store } from "@/store";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { extractInstanceResourceName } from "@/utils/v1/instance";
import { extractProjectResourceName } from "@/utils/v1/project";

export function DatabaseRevisionDetailPage({
  project,
  instance,
  database,
  revisionId,
}: {
  project: string;
  instance: string;
  database: string;
  revisionId: string;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const [loading, setLoading] = useState(true);
  const projectId = extractProjectResourceName(project);
  const databaseFullName = database;
  const { databaseName } = extractDatabaseResourceName(database);
  const instanceId = extractInstanceResourceName(instance);
  const revisionName = `${databaseFullName}/revisions/${revisionId}`;

  useEffect(() => {
    let cancelled = false;

    setLoading(true);
    void databaseStore
      .getOrFetchDatabaseByName(databaseFullName)
      .catch((error) => {
        console.error("Failed to fetch database details", error);
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [databaseFullName, databaseStore]);

  const handleProjectBreadcrumbClick = () => {
    router.push({
      name: PROJECT_V1_ROUTE_DATABASES,
      params: { projectId },
    });
  };

  const handleDatabaseBreadcrumbClick = () => {
    router.push({
      name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
      params: {
        projectId,
        instanceId,
        databaseName,
      },
    });
  };

  const handleRevisionBreadcrumbClick = () => {
    router.push({
      name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
      params: {
        projectId,
        instanceId,
        databaseName,
      },
      hash: "#revision",
    });
  };

  return (
    <div className="flex min-h-full flex-col gap-y-4 p-4">
      <nav aria-label="Breadcrumb" className="mb-4">
        <ol className="flex flex-wrap items-center gap-x-2 text-sm text-control-light">
          <li>
            <button
              type="button"
              className="transition-colors hover:text-accent"
              onClick={handleProjectBreadcrumbClick}
            >
              {t("common.databases")}
            </button>
          </li>
          <li aria-hidden="true">/</li>
          <li>
            <button
              type="button"
              className="transition-colors hover:text-accent"
              onClick={handleDatabaseBreadcrumbClick}
            >
              {databaseName}
            </button>
          </li>
          <li aria-hidden="true">/</li>
          <li>
            <button
              type="button"
              className="transition-colors hover:text-accent"
              onClick={handleRevisionBreadcrumbClick}
            >
              {t("database.revision.self")}
            </button>
          </li>
          <li aria-hidden="true">/</li>
          <li className="text-main">{revisionId}</li>
        </ol>
      </nav>

      {loading ? (
        <div className="flex items-center justify-center py-10">
          <LoaderCircle className="h-4 w-4 animate-spin text-control-light" />
        </div>
      ) : (
        <RevisionDetailPanel revisionName={revisionName} />
      )}
    </div>
  );
}
