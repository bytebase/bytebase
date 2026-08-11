import { LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { PROJECT_V1_ROUTE_DATABASES } from "@/app/router/handles";
import { RouterLink } from "@/components/RouterLink";
import { RevisionDetailPanel } from "@/components/revision";
import { autoDatabaseRoute } from "@/utils";
import { useProjectDatabaseDetail } from "./database-detail/useProjectDatabaseDetail";

export function DatabaseRevisionDetailPage({
  projectId,
  instanceId,
  databaseName,
  revisionId,
  routeQuery,
}: {
  projectId: string;
  instanceId: string;
  databaseName: string;
  revisionId: string;
  routeQuery?: Record<string, string | undefined>;
}) {
  const { t } = useTranslation();
  const parent = routeQuery?.parent ?? `instances/${instanceId}`;
  const databaseRouteQuery = { ...routeQuery, parent };
  const detail = useProjectDatabaseDetail({
    parent,
    projectId,
    instanceId,
    databaseName,
  });
  const revisionName = `${detail.databaseName}/revisions/${revisionId}`;

  if (detail.loading) {
    return (
      <div className="flex items-center justify-center py-10">
        <LoaderCircle className="h-4 w-4 animate-spin text-control-light" />
      </div>
    );
  }

  if (!detail.ready) {
    return null;
  }

  const baseDatabaseRoute = autoDatabaseRoute(detail.database);
  const databaseRoute = {
    ...baseDatabaseRoute,
    query: {
      ...databaseRouteQuery,
      ...baseDatabaseRoute.query,
    },
  };

  return (
    <div className="flex min-h-full flex-col gap-y-4 p-4">
      <nav aria-label="Breadcrumb" className="mb-4">
        <ol className="flex flex-wrap items-center gap-x-2 text-sm text-control-light">
          <li>
            <RouterLink
              to={{
                name: PROJECT_V1_ROUTE_DATABASES,
                params: { projectId },
              }}
              className="transition-colors hover:text-accent"
            >
              {t("common.databases")}
            </RouterLink>
          </li>
          <li aria-hidden="true">/</li>
          <li>
            <RouterLink
              to={databaseRoute}
              className="transition-colors hover:text-accent"
            >
              {databaseName}
            </RouterLink>
          </li>
          <li aria-hidden="true">/</li>
          <li>
            <RouterLink
              to={{
                ...databaseRoute,
                hash: "#revision",
              }}
              className="transition-colors hover:text-accent"
            >
              {t("database.revision.self")}
            </RouterLink>
          </li>
          <li aria-hidden="true">/</li>
          <li className="text-main">{revisionId}</li>
        </ol>
      </nav>

      <RevisionDetailPanel revisionName={revisionName} />
    </div>
  );
}
