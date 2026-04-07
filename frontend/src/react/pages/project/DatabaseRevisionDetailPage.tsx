import { LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { RevisionDetailPanel } from "@/react/components/revision";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DATABASES } from "@/router/dashboard/projectV1";
import { useDatabaseV1ByName } from "@/store";
import {
  databaseV1Url,
  extractDatabaseResourceName,
} from "@/utils/v1/database";

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
  const databaseFullName = `instances/${instanceId}/databases/${databaseName}`;
  const { database, ready } = useDatabaseV1ByName(databaseFullName);
  const databaseItem = useVueState(() => database.value);
  const databaseReady = useVueState(() => ready.value);
  const revisionName = `${databaseFullName}/revisions/${revisionId}`;

  const handleProjectBreadcrumbClick = () => {
    router.push({
      name: PROJECT_V1_ROUTE_DATABASES,
      params: { projectId },
    });
  };

  const handleDatabaseBreadcrumbClick = () => {
    router.push(databaseV1Url(databaseItem));
  };

  const handleRevisionBreadcrumbClick = () => {
    router.push(`${databaseV1Url(databaseItem)}#revision`);
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
              {extractDatabaseResourceName(databaseItem.name).databaseName}
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

      {databaseReady ? (
        <RevisionDetailPanel revisionName={revisionName} />
      ) : (
        <div className="flex items-center justify-center py-10">
          <LoaderCircle className="h-4 w-4 animate-spin text-control-light" />
        </div>
      )}
    </div>
  );
}
