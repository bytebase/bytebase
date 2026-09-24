import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/components/EnvironmentLabel";
import { Alert } from "@/components/ui/alert";
import { useEnvironmentList, usePlanFeature } from "@/hooks/useAppState";
import type { EnvLimitationKind } from "@/lib/project-member/utils";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { DirectExecutionScope } from "./directExecutionScope";

/** Who the sentence is about: the surface picks the lead. */
export type { DirectExecutionScope } from "./directExecutionScope";
export { directExecutionScopeFromCondition } from "./directExecutionScope";

export type DirectExecutionLead = "grant" | "request" | "approver" | "binding";

const formatList = (items: string[], language: string): string => {
  try {
    return new Intl.ListFormat(language, { type: "conjunction" }).format(items);
  } catch {
    return items.join(", ");
  }
};

export function DirectExecutionCallout({
  kind,
  lead,
  scope,
  grantee,
}: {
  kind: EnvLimitationKind;
  lead: DirectExecutionLead;
  scope: DirectExecutionScope;
  /** Display name interpolated into the approver lead. */
  grantee?: string;
}) {
  const { t, i18n } = useTranslation();
  const environmentList = useEnvironmentList();
  const hasEnvTierFeature = usePlanFeature(
    PlanFeature.FEATURE_ENVIRONMENT_TIERS
  );

  // Literal keys per lead so the i18n check can see every key in use.
  const leadText = (which: DirectExecutionLead): string => {
    switch (which) {
      case "grant":
        return t("project.members.direct-execution.lead-grant", { kind });
      case "request":
        return t("project.members.direct-execution.lead-request", { kind });
      case "approver":
        return t("project.members.direct-execution.lead-approver", {
          kind,
          grantee,
        });
      case "binding":
        return t("project.members.direct-execution.lead-binding", { kind });
    }
  };

  switch (scope.type) {
    case "none":
      // Info, not warning: the binding adds nothing, so there is no risk to
      // surface. Yellow on the benign state would mislead readers.
      return (
        <Alert variant="info">
          {lead === "binding"
            ? t("project.members.direct-execution.none-binding", { kind })
            : t("project.members.direct-execution.none-grant", { kind })}
        </Alert>
      );
    case "all":
      return (
        <Alert variant="warning">
          {t("project.members.direct-execution.all", { kind })}
        </Alert>
      );
    case "custom":
      return (
        <Alert
          variant="warning"
          description={
            <div className="flex flex-col gap-y-1">
              <span>
                {t("project.members.direct-execution.custom", { kind })}
              </span>
              <code className="font-mono text-xs leading-4 break-all">
                {scope.expression}
              </code>
            </div>
          }
        />
      );
    case "some": {
      const protectedTitles = hasEnvTierFeature
        ? scope.environments
            .map((name) => environmentList.find((env) => env.name === name))
            .filter((env) => env?.tags?.protected === "protected")
            .map((env) => env?.title ?? "")
        : [];
      return (
        <Alert
          variant="warning"
          description={
            <div className="flex flex-col gap-y-1.5">
              <div className="flex flex-wrap items-center gap-1">
                <span>{leadText(lead)}</span>
                {scope.environments.map((name) => (
                  <EnvironmentLabel
                    key={name}
                    environmentName={name}
                    className="text-xs"
                  />
                ))}
              </div>
              {protectedTitles.length > 0 && (
                <div>
                  {t("project.members.direct-execution.protected", {
                    count: protectedTitles.length,
                    environments: formatList(protectedTitles, i18n.language),
                  })}
                </div>
              )}
            </div>
          }
        />
      );
    }
  }
}
