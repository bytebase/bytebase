import { ShieldAlert } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { featureToRef, useEnvironmentV1Store } from "@/store";
import type { Environment } from "@/types";
import { NULL_ENVIRONMENT_NAME, UNKNOWN_ENVIRONMENT_NAME } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hexToRgb } from "@/utils";

/**
 * Renders an environment as a colored pill with optional shield icon
 * for protected environments.
 *
 * Two usage modes:
 * - Pass `environment` object directly (avoids store lookup)
 * - Pass `environmentName` string (looks up from environment store)
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
  const { t } = useTranslation();
  const environmentStore = useEnvironmentV1Store();

  // Always called (rules of hooks); result ignored when envProp is provided.
  const envFromStore = useVueState(() =>
    environmentStore.getEnvironmentByName(
      environmentName || NULL_ENVIRONMENT_NAME
    )
  );

  const environment = envProp ?? envFromStore;

  const isUnset =
    environment.name === UNKNOWN_ENVIRONMENT_NAME ||
    environment.name === NULL_ENVIRONMENT_NAME;

  const hasEnvTierFeature = useVueState(
    () => featureToRef(PlanFeature.FEATURE_ENVIRONMENT_TIERS).value
  );
  const isProtected =
    hasEnvTierFeature && environment.tags?.protected === "protected";

  const bgColorRgb = environment.color ? hexToRgb(environment.color) : null;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-x-1 px-1.5 rounded truncate",
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
}
