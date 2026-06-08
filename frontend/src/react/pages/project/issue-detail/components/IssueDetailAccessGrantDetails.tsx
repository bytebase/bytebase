import { Download, EyeOff, Loader2, Shield } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { useAppStore } from "@/react/stores/app";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { isValidDatabaseName } from "@/types";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import { unknownDatabase } from "@/types/v1/database";
import { extractProjectResourceName, hasProjectPermissionV2 } from "@/utils";
import { getAccessGrantExpirationText } from "@/utils/accessGrant";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { useIssueDetailContext } from "../context/IssueDetailContext";

export function IssueDetailAccessGrantDetails() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  const fetchAccessGrant = useAppStore((state) => state.fetchAccessGrant);
  const searchMyAccessGrants = useAppStore(
    (state) => state.searchMyAccessGrants
  );
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useProjectByName(projectName);
  void projectsByName;
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
          grant = await fetchAccessGrant(name);
        } else {
          const parent = `projects/${extractProjectResourceName(page.issue.name)}`;
          const response = await searchMyAccessGrants({
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
              void useAppStore.getState().getOrFetchDatabaseByName(target);
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
  }, [fetchAccessGrant, searchMyAccessGrants, page.issue, project]);

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
              <div className="max-h-[30em] overflow-auto rounded-xs bg-gray-50 p-4">
                <pre className="whitespace-pre-wrap font-mono text-sm">
                  {accessGrant.query}
                </pre>
              </div>
            </div>
          )}

          {(accessGrant.unmask || accessGrant.export) && (
            <div className="flex flex-col gap-y-2">
              <span className="text-sm text-control-light">
                {t("common.permissions")}
              </span>
              <GrantScopeCard
                unmask={accessGrant.unmask}
                exportAllowed={accessGrant.export}
              />
            </div>
          )}

          <div className="flex flex-col gap-y-1">
            <span className="text-sm text-control-light">
              {t("common.expiration")}
            </span>
            <div className="text-base">
              {expirationInfo.type === "never"
                ? t("project.members.never-expires")
                : expirationInfo.type === "duration"
                  ? // Pending grant — TTL is still on the proto, safe
                    // to surface as "{{duration}} after issue approved".
                    t("issue.access-grant.duration-after-approval", {
                      duration: expirationInfo.value,
                    })
                  : // Active grant — show the absolute expire datetime
                    // only. The original requested duration is gone
                    // post-activation (input-only proto field); we
                    // can't recover it without double-counting the
                    // approval wait. Bot review #3370767734.
                    expirationInfo.value}
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function IssueDetailAccessGrantTarget({ target }: { target: string }) {
  const databasesByName = useAppStore((s) => s.databasesByName);
  const database = useMemo(
    () => databasesByName[target] ?? unknownDatabase(),
    [databasesByName, target]
  );
  // Subscribe to the env cache so the row re-resolves once it loads; compute
  // via getState() in a memo because getEnvironmentByName returns a fresh
  // fallback object on a miss (unsafe as a raw selector — would loop).
  const environmentList = useAppStore((s) => s.environmentList);
  const environment = useMemo(
    () =>
      useAppStore
        .getState()
        .getEnvironmentByName(
          database.effectiveEnvironment ?? database.environment ?? ""
        ),
    [environmentList, database]
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

/**
 * "This grant allows" card: a neutral header bar and one row per
 * granted capability. The risk signal lives in the rows, not the
 * chrome — the unmask row goes warning-toned (text + icon) while
 * everything else stays grey. Built from `warning` / `control-*`
 * tokens only; kept local for now, can be lifted when the request
 * drawer's confirmation step adopts the same shape.
 */
function GrantScopeCard({
  unmask,
  exportAllowed,
}: {
  unmask: boolean;
  exportAllowed: boolean;
}) {
  const { t } = useTranslation();
  return (
    <div className="overflow-hidden rounded-sm border border-control-border">
      <div className="flex items-center gap-x-2 border-b border-control-border bg-control-bg px-3 py-2 text-sm font-medium text-control">
        <Shield className="size-4 shrink-0" />
        <span>{t("issue.access-grant.this-grant-allows")}</span>
      </div>
      <div className="flex flex-col">
        {unmask && (
          // Unmask carries the entire sensitive-access signal —
          // warning-toned text + icon make the verdict live in the
          // row itself. Export and any future Tier-1 capability stay
          // neutral; the contrast is the warning.
          <div className="flex items-center gap-x-2 px-3 py-2 text-sm text-warning-hover">
            <EyeOff className="size-4 shrink-0" />
            <span>{t("issue.access-grant.unmask-data-in-results")}</span>
          </div>
        )}
        {exportAllowed && (
          <div className="flex items-center gap-x-2 px-3 py-2 text-sm">
            <Download className="size-4 shrink-0 text-control-light" />
            <span>{t("issue.access-grant.export-results-to-file")}</span>
          </div>
        )}
      </div>
    </div>
  );
}
