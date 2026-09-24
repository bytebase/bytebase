import { useId } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentSelect } from "@/components/EnvironmentSelect";
import { Badge } from "@/components/ui/badge";
import { FormError, FormField } from "@/components/ui/form";
import { Switch } from "@/components/ui/switch";
import { usePlanFeature } from "@/hooks/useAppState";
import type { EnvLimitationKind } from "@/lib/project-member/utils";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  DirectExecutionCallout,
  type DirectExecutionLead,
} from "./DirectExecutionCallout";

/**
 * The form's view of the field. `enabled` is kept beside the list rather
 * than derived from it: off and on-with-nothing-picked serialize to the
 * same empty clause, and only the form knows which one the reader meant.
 */
export interface DirectExecutionValue {
  enabled: boolean;
  environments: string[];
}

export const EMPTY_DIRECT_EXECUTION: DirectExecutionValue = {
  enabled: false,
  environments: [],
};

/** On with nothing picked is the one invalid shape; the form must not submit it. */
export const isDirectExecutionValid = (value: DirectExecutionValue) =>
  !value.enabled || value.environments.length > 0;

/**
 * What the binding's environment clause should carry. Off writes the empty
 * list — byte-for-byte what an empty picker wrote before the switch existed.
 */
export const directExecutionEnvironments = (value: DirectExecutionValue) =>
  value.enabled ? value.environments : [];

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
  const { enabled, environments } = value;

  return (
    <FormField
      title={<>{t("project.members.direct-execution.title", { kind })}</>}
      data-testid="direct-execution-field"
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
            renderSuffix={(environment) =>
              hasEnvTierFeature &&
              environment.tags?.protected === "protected" ? (
                <Badge variant="warning">
                  {t("project.members.direct-execution.protected-tag")}
                </Badge>
              ) : null
            }
          />
          {environments.length === 0 ? (
            <FormError>
              {t("project.members.direct-execution.pick-or-off")}
            </FormError>
          ) : (
            <DirectExecutionCallout
              kind={kind}
              lead={lead}
              scope={{ type: "some", environments }}
            />
          )}
        </div>
      ) : (
        <p className="text-xs leading-4 text-control-light">
          {t("project.members.direct-execution.off-caption", { kind })}
        </p>
      )}
    </FormField>
  );
}
