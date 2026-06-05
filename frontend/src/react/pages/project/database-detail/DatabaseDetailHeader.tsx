import { Check, Copy, ShieldAlert } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { RouterLink } from "@/react/components/RouterLink";
import { useEnvironment, usePlanFeature } from "@/react/hooks/useAppState";
import { INSTANCE_ROUTE_DETAIL } from "@/react/router/handles";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  formatEnvironmentName,
  isValidEnvironmentName,
  UNKNOWN_ENVIRONMENT_NAME,
} from "@/types/v1/environment";
import {
  extractInstanceResourceName,
  getInstanceResource,
  hasWorkspacePermissionV2,
  hexToRgb,
  instanceV1Name,
} from "@/utils";
import { extractReleaseUID } from "@/utils/v1/release";
import { DatabaseSQLEditorButton } from "./DatabaseSQLEditorButton";

const extractDatabaseParts = (resource: string) => {
  const matches = resource.match(
    /(?:^|\/)instances\/(?<instanceName>[^/]+)\/databases\/(?<databaseName>[^/]+)(?:$|\/)/
  );
  return {
    databaseName: matches?.groups?.databaseName ?? "",
    instanceName: matches?.groups?.instanceName ?? "",
  };
};

async function copyToClipboard(text: string): Promise<boolean> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      return false;
    }
  }
  return false;
}

export function DatabaseDetailHeader({
  database,
  onSQLEditorFailed,
}: {
  database: Database;
  onSQLEditorFailed?: (database: Database) => void;
}) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const { databaseName } = useMemo(
    () => extractDatabaseParts(database.name),
    [database.name]
  );

  const instanceResource = getInstanceResource(database);
  const instanceLabel = instanceV1Name(instanceResource);
  const instanceId = extractInstanceResourceName(instanceResource.name);
  const canViewInstance = hasWorkspacePermissionV2("bb.instances.get");

  const environment = useEnvironment(database.effectiveEnvironment ?? "");
  const hasEnvironmentTierFeature = usePlanFeature(
    PlanFeature.FEATURE_ENVIRONMENT_TIERS
  );

  const isValidEnv =
    !!environment &&
    isValidEnvironmentName(environment.name) &&
    environment.name !== UNKNOWN_ENVIRONMENT_NAME;

  const environmentTitle = useMemo(() => {
    if (!isValidEnv) {
      return t("common.unassigned");
    }
    return environment.title || environment.id;
  }, [environment, isValidEnv, t]);

  const isProductionEnv =
    isValidEnv &&
    hasEnvironmentTierFeature &&
    environment.tags?.protected === "protected";

  const environmentColorRgb = useMemo(() => {
    if (!isValidEnv || !environment.color) {
      return "";
    }
    return hexToRgb(environment.color).join(", ");
  }, [environment, isValidEnv]);

  const environmentBadgeStyle: React.CSSProperties | undefined = useMemo(() => {
    if (!environmentColorRgb) {
      return undefined;
    }
    return {
      backgroundColor: `rgba(${environmentColorRgb}, 0.1)`,
      borderTopColor: `rgb(${environmentColorRgb})`,
      color: `rgb(${environmentColorRgb})`,
      padding: "0 6px",
      borderRadius: "4px",
    };
  }, [environmentColorRgb]);

  const handleCopy = useCallback(async () => {
    const success = await copyToClipboard(database.name);
    setCopied(success);
    if (success) {
      window.setTimeout(() => setCopied(false), 1200);
    }
  }, [database.name]);

  return (
    <div className="flex min-w-0 flex-1 shrink-0 flex-col gap-y-2">
      <div className="flex w-full min-w-0 flex-col">
        <div className="flex items-center gap-x-2 truncate text-xl font-bold text-main">
          {databaseName}
        </div>
        <div className="mt-1 flex w-full min-w-0 items-center gap-x-1 text-sm text-control-light">
          <span className="truncate">{database.name}</span>
          <button
            type="button"
            className="inline-flex shrink-0 items-center p-0.5 text-control-light hover:text-main"
            onClick={() => void handleCopy()}
          >
            {copied ? (
              <Check className="h-3.5 w-3.5" />
            ) : (
              <Copy className="h-3.5 w-3.5" />
            )}
          </button>
        </div>
      </div>

      <div
        className="flex flex-col gap-y-1 text-sm md:flex-row md:flex-wrap md:items-center md:gap-x-4"
        data-label="bb-database-detail-info-block"
      >
        <div className="flex items-center gap-x-1.5">
          <span className="text-control-light">{t("common.environment")}</span>
          {isValidEnv ? (
            <RouterLink
              to={{ path: `/${formatEnvironmentName(environment.id)}` }}
              className="inline-flex cursor-pointer items-center gap-x-1 hover:underline"
              style={environmentBadgeStyle}
              onClick={(e) => e.stopPropagation()}
            >
              <span>{environmentTitle}</span>
              {isProductionEnv && <ShieldAlert className="h-4 w-4 shrink-0" />}
            </RouterLink>
          ) : (
            <span className="italic text-control-light">
              {environmentTitle}
            </span>
          )}
        </div>
        <div className="flex items-center gap-x-1.5">
          <span className="text-control-light">{t("common.instance")}</span>
          {canViewInstance && instanceId ? (
            <RouterLink
              to={{
                name: INSTANCE_ROUTE_DETAIL,
                params: { instanceId },
              }}
              className="inline-flex cursor-pointer items-center gap-x-1 hover:underline"
              onClick={(e) => e.stopPropagation()}
            >
              <EngineIcon
                engine={instanceResource.engine}
                className="h-4 w-4"
              />
              {instanceLabel}
            </RouterLink>
          ) : (
            <span className="inline-flex items-center gap-x-1">
              <EngineIcon
                engine={instanceResource.engine}
                className="h-4 w-4"
              />
              {instanceLabel}
            </span>
          )}
        </div>
        {database.release && (
          <div className="flex items-center gap-x-1.5">
            <span className="text-control-light">{t("common.release")}</span>
            <span>{extractReleaseUID(database.release)}</span>
          </div>
        )}
        <DatabaseSQLEditorButton
          database={database}
          onFailed={onSQLEditorFailed}
        />
      </div>
    </div>
  );
}
