import { ShieldAlert } from "lucide-react";
import { memo } from "react";
import { useTranslation } from "react-i18next";
import { useEnvironment, usePlanFeature } from "@/react/hooks/useAppState";
import { cn } from "@/react/lib/utils";
import type { Environment } from "@/types";
import { NULL_ENVIRONMENT_NAME, UNKNOWN_ENVIRONMENT_NAME } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hexToRgb } from "@/utils";

/**
 * Pure-presentational environment badge — no hooks, no store reads.
 *
 * Use this in row-heavy contexts (e.g. DatabaseTableView with hundreds of
 * rows) where the caller can hoist `useEnvironmentList()` + the env-tier
 * feature check to the table level and pass the resolved `environment` and
 * `hasEnvTierFeature` flag down. That avoids N Zustand subscriptions and
 * N redundant `loadSubscription` effects per row.
 */
export const EnvironmentBadge = memo(function EnvironmentBadge({
  environment,
  hasEnvTierFeature,
  className,
}: {
  environment: Environment;
  hasEnvTierFeature: boolean;
  className?: string;
}) {
  const { t } = useTranslation();

  const isUnset =
    environment.name === UNKNOWN_ENVIRONMENT_NAME ||
    environment.name === NULL_ENVIRONMENT_NAME;
  const isProtected =
    hasEnvTierFeature && environment.tags?.protected === "protected";
  const bgColorRgb = environment.color ? hexToRgb(environment.color) : null;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-x-1 px-1.5 rounded-xs truncate",
        className
      )}
      style={
        bgColorRgb && !isUnset
          ? {
              backgroundColor: `rgba(${bgColorRgb.join(", ")}, 0.1)`,
              color: `rgb(${bgColorRgb.join(", ")})`,
            }
          : undefined
      }
    >
      <span className="truncate">
        {isUnset ? (
          <span className="text-control-light italic">
            {t("common.unassigned")}
          </span>
        ) : (
          environment.title
        )}
      </span>
      {isProtected && !isUnset && (
        <ShieldAlert className="w-3.5 h-3.5 shrink-0 text-current" />
      )}
    </span>
  );
});

/**
 * Renders an environment as a colored pill with optional shield icon
 * for protected environments. Smart wrapper that looks up the environment
 * by name and checks the env-tier feature gate.
 *
 * Two usage modes:
 * - Pass `environment` object directly (avoids env store lookup)
 * - Pass `environmentName` string (looks up from environment store)
 *
 * For high-row use, prefer `EnvironmentBadge` directly with hoisted data.
 */
export function EnvironmentLabel({
  environment: envProp,
  environmentName,
  className,
}: {
  environment?: Environment;
  environmentName?: string;
  className?: string;
}) {
  // Always called (rules of hooks); result ignored when envProp is provided.
  const envFromStore = useEnvironment(environmentName || NULL_ENVIRONMENT_NAME);
  const environment = envProp ?? envFromStore;
  const hasEnvTierFeature = usePlanFeature(
    PlanFeature.FEATURE_ENVIRONMENT_TIERS
  );
  return (
    <EnvironmentBadge
      environment={environment}
      hasEnvTierFeature={hasEnvTierFeature}
      className={className}
    />
  );
}
