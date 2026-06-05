import { LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { RouterLink } from "@/react/components/RouterLink";
import { RevisionDetailPanel } from "@/react/components/revision";
import {
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
} from "@/react/router/handles";
import { useProjectDatabaseDetail } from "./database-detail/useProjectDatabaseDetail";

export function DatabaseRevisionDetailPage({
  projectId,
  instanceId,
  databaseName,
  revisionId,
}: {
  projectId: string;
  instanceId: string;
  databaseName: string;
  revisionId: string;
}) {
  const { t } = useTranslation();
  const detail = useProjectDatabaseDetail({
    projectId,
    instanceId,
    databaseName,
    routeName: PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
    revisionId,
  });
  const revisionName = `${detail.databaseName}/revisions/${revisionId}`;

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
              to={{
                name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
                params: {
                  projectId,
                  instanceId,
                  databaseName,
                },
              }}
              className="transition-colors hover:text-accent"
            >
              {databaseName}
            </RouterLink>
          </li>
          <li aria-hidden="true">/</li>
          <li>
            <RouterLink
              to={{
                name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
                params: {
                  projectId,
                  instanceId,
                  databaseName,
                },
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

      {detail.loading ? (
        <div className="flex items-center justify-center py-10">
          <LoaderCircle className="h-4 w-4 animate-spin text-control-light" />
        </div>
      ) : (
        <RevisionDetailPanel revisionName={revisionName} />
      )}
    </div>
  );
}
