import { useCallback, useId } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentSelect } from "@/components/EnvironmentSelect";
import { Badge } from "@/components/ui/badge";
import { FormError, FormField } from "@/components/ui/form";
import { Switch } from "@/components/ui/switch";
import { useEnvironmentList, usePlanFeature } from "@/hooks/useAppState";
import type { DirectExecutionValue } from "@/lib/project-member/directExecution";
import type { EnvLimitationKind } from "@/lib/project-member/utils";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  type Environment,
  isProtectedEnvironment,
} from "@/types/v1/environment";
import {
  DirectExecutionCallout,
  type DirectExecutionLead,
} from "./DirectExecutionCallout";

export function DirectExecutionField({
  kind,
  lead,
  value,
  onChange,
}: {
  kind: EnvLimitationKind;
  lead: Extract<DirectExecutionLead, "grant" | "request">;
  value: DirectExecutionValue;
  onChange: (next: DirectExecutionValue) => void;
}) {
  const { t } = useTranslation();
  const labelId = useId();
  const hasEnvTierFeature = usePlanFeature(
    PlanFeature.FEATURE_ENVIRONMENT_TIERS
  );
  const environmentList = useEnvironmentList();
  const { enabled, environments } = value;
  // Stable so the picker's option memo survives the form's re-renders.
  const renderProtectedTag = useCallback(
    (environment: Environment) =>
      isProtectedEnvironment(environment, hasEnvTierFeature) ? (
        <Badge variant="warning">
          {t("project.members.direct-execution.protected-tag")}
        </Badge>
      ) : null,
    [hasEnvTierFeature, t]
  );

  return (
    <FormField
      title={<>{t("project.members.direct-execution.title", { kind })}</>}
    >
      <div className="flex items-center gap-x-2">
        <Switch
          checked={enabled}
          onCheckedChange={(checked) =>
            onChange({ enabled: checked, environments })
          }
          aria-labelledby={labelId}
        />
        <span id={labelId} className="text-sm leading-5">
          {t("project.members.direct-execution.switch-label", { kind })}
        </span>
      </div>
      {enabled ? (
        <div className="flex flex-col gap-y-1.5 pt-2">
          <span className="text-sm font-medium leading-5">
            {t("project.members.direct-execution.in-environments")}
          </span>
          <EnvironmentSelect
            multiple
            portal
            value={environments}
            onChange={(next) => onChange({ enabled, environments: next })}
            placeholder={t(
              "project.members.direct-execution.select-environments"
            )}
            renderSuffix={renderProtectedTag}
          />
          {environments.length === 0 ? (
            <FormError>
              {t("project.members.direct-execution.environment-required")}
            </FormError>
          ) : (
            <DirectExecutionCallout
              kind={kind}
              lead={lead}
              scope={{ type: "some", environments }}
              environmentList={environmentList}
              hasEnvTierFeature={hasEnvTierFeature}
            />
          )}
        </div>
      ) : (
        <p className="text-xs leading-4 text-control-light">
          {t("project.members.direct-execution.none", { kind })}
        </p>
      )}
    </FormField>
  );
}
