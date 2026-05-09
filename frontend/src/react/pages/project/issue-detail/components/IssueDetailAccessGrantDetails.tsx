import { Loader2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { Alert } from "@/react/components/ui/alert";
import { Checkbox } from "@/react/components/ui/checkbox";
import { useVueState } from "@/react/hooks/useVueState";
import {
  useAccessGrantStore,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { isValidDatabaseName } from "@/types";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import { extractProjectResourceName, hasProjectPermissionV2 } from "@/utils";
import { getAccessGrantExpirationText } from "@/utils/accessGrant";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { useIssueDetailContext } from "../context/IssueDetailContext";

export function IssueDetailAccessGrantDetails() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const projectStore = useProjectV1Store();
  const databaseStore = useDatabaseV1Store();
  const accessGrantStore = useAccessGrantStore();
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));
  const [isLoading, setIsLoading] = useState(true);
  const [accessGrant, setAccessGrant] = useState<AccessGrant | undefined>();

  useEffect(() => {
    let canceled = false;

    const run = async () => {
      const name = page.issue?.accessGrant;
      if (!name || !page.issue) {
        setIsLoading(false);
        setAccessGrant(undefined);
        return;
      }

      setIsLoading(true);
      try {
        let grant: AccessGrant | undefined;
        if (hasProjectPermissionV2(project, "bb.accessGrants.get")) {
          grant = await accessGrantStore.getAccessGrant(name);
        } else {
          const parent = `projects/${extractProjectResourceName(page.issue.name)}`;
          const response = await accessGrantStore.searchMyAccessGrants({
            parent,
            filter: { name },
          });
          grant = response.accessGrants[0];
        }

        if (canceled) {
          return;
        }

        setAccessGrant(grant);
        if (grant) {
          for (const target of grant.targets) {
            if (isValidDatabaseName(target)) {
              void databaseStore.getOrFetchDatabaseByName(target);
            }
          }
        }
      } finally {
        if (!canceled) {
          setIsLoading(false);
        }
      }
    };

    void run();
    return () => {
      canceled = true;
    };
  }, [accessGrantStore, databaseStore, page.issue, project]);

  const expirationInfo = accessGrant
    ? getAccessGrantExpirationText(accessGrant)
    : { type: "never" as const };

  return (
    <div className="flex flex-col gap-y-4">
      <h3 className="text-base font-medium">
        {t("issue.access-grant.details")}
      </h3>

      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-5 w-5 animate-spin text-control-light" />
        </div>
      ) : accessGrant ? (
        <div className="flex flex-col gap-y-4 rounded-sm border p-4">
          {accessGrant.targets.length > 0 && (
            <div className="flex flex-col gap-y-2">
              <span className="text-sm text-control-light">
                {t("common.databases")}
              </span>
              <div className="flex flex-wrap gap-2">
                {accessGrant.targets.map((target) => (
                  <IssueDetailAccessGrantTarget key={target} target={target} />
                ))}
              </div>
            </div>
          )}

          {accessGrant.query && (
            <div className="flex flex-col gap-y-2">
              <span className="text-sm text-control-light">
                {t("common.statement")}
              </span>
              {accessGrant.unmask && (
                <Alert
                  variant="warning"
                  showIcon={false}
                  description={t("sql-editor.unmask-warning")}
                />
              )}
              <div className="max-h-[30em] overflow-auto rounded-xs bg-gray-50 p-4">
                <pre className="whitespace-pre-wrap font-mono text-sm">
                  {accessGrant.query}
                </pre>
              </div>
              <label className="flex items-center gap-2">
                <Checkbox checked={accessGrant.unmask} disabled />
                <span className="text-base">
                  {t("sql-editor.access-type-unmask")}
                </span>
              </label>
            </div>
          )}

          <div className="flex flex-col gap-y-1">
            <span className="text-sm text-control-light">
              {expirationInfo.type === "duration"
                ? t("common.duration")
                : t("issue.access-grant.expired-at")}
            </span>
            <div className="text-base">
              {expirationInfo.type === "never"
                ? t("project.members.never-expires")
                : expirationInfo.value}
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function IssueDetailAccessGrantTarget({ target }: { target: string }) {
  const databaseStore = useDatabaseV1Store();
  const environmentStore = useEnvironmentV1Store();
  const database = useVueState(() => databaseStore.getDatabaseByName(target));
  const environment = useVueState(() =>
    environmentStore.getEnvironmentByName(
      database.effectiveEnvironment ?? database.environment ?? ""
    )
  );
  const instance = database.instanceResource;
  const { databaseName } = extractDatabaseResourceName(target);

  return (
    <div className="inline-flex min-w-0 items-center gap-2 rounded-sm border px-2 py-1.5">
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 items-center truncate text-sm">
          {instance && (
            <EngineIcon
              engine={instance.engine}
              className="mr-1 inline-block h-4 w-4"
            />
          )}
          <span className="mr-1 truncate text-gray-400">
            {environment.title}
          </span>
          <span className="truncate text-gray-600">{instance?.title}</span>
          <span className="mx-1 shrink-0 text-gray-500">&gt;</span>
          <span className="truncate text-gray-800">{databaseName}</span>
        </div>
      </div>
    </div>
  );
}
