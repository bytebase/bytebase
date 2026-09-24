import { useTranslation } from "react-i18next";
import { EnvironmentBadge } from "@/components/EnvironmentLabel";
import { Alert } from "@/components/ui/alert";
import type { DirectExecutionScope } from "@/lib/project-member/directExecution";
import type { EnvLimitationKind } from "@/lib/project-member/utils";
import {
  type Environment,
  isProtectedEnvironment,
  resolveEnvironment,
} from "@/types/v1/environment";
import { formatList } from "@/utils/string";

/** Who the sentence is about: the surface picks the lead. */
export type DirectExecutionLead = "grant" | "request" | "approver" | "binding";

/** The approver lead names the grantee; no other lead takes a name. */
type DirectExecutionLeadProps =
  | { lead: "approver"; grantee: string }
  | { lead: Exclude<DirectExecutionLead, "approver">; grantee?: undefined };

/**
 * Hook-free, like the badge it renders: a member's drawer shows one callout
 * per binding, so the environment list and the tier feature are read once by
 * the surface and handed down.
 */
export function DirectExecutionCallout({
  kind,
  scope,
  environmentList,
  hasEnvTierFeature,
  ...leadProps
}: {
  kind: EnvLimitationKind;
  scope: DirectExecutionScope;
  environmentList: Environment[];
  hasEnvTierFeature: boolean;
} & DirectExecutionLeadProps) {
  const { t, i18n } = useTranslation();

  // Literal keys per lead so the i18n check can see every key in use.
  const leadText = (): string => {
    switch (leadProps.lead) {
      case "grant":
        return t("project.members.direct-execution.lead-grant", { kind });
      case "request":
        return t("project.members.direct-execution.lead-request", { kind });
      case "approver":
        return t("project.members.direct-execution.lead-approver", {
          kind,
          grantee: leadProps.grantee,
        });
      case "binding":
        return t("project.members.direct-execution.lead-binding", { kind });
    }
  };

  switch (scope.type) {
    case "none":
      // Info, not warning: nothing is allowed, so there is no risk to surface.
      return (
        <Alert variant="info">
          {t("project.members.direct-execution.none", { kind })}
        </Alert>
      );
    case "all":
      return (
        <Alert variant="warning">
          {t("project.members.direct-execution.all", { kind })}
        </Alert>
      );
    case "some": {
      const environments = scope.environments.map((name) =>
        resolveEnvironment(name, environmentList)
      );
      const protectedTitles = environments.flatMap((env) =>
        isProtectedEnvironment(env, hasEnvTierFeature) ? [env.title] : []
      );
      return (
        <Alert
          variant="warning"
          description={
            <div className="flex flex-col gap-y-1.5">
              <div className="flex flex-wrap items-center gap-1">
                <span>{leadText()}</span>
                {environments.map((env) => (
                  <EnvironmentBadge
                    key={env.name}
                    environment={env}
                    hasEnvTierFeature={hasEnvTierFeature}
                    className="text-xs"
                  />
                ))}
              </div>
              {protectedTitles.length > 0 && (
                <div>
                  {t("project.members.direct-execution.protected", {
                    count: protectedTitles.length,
                    environments: formatList(
                      protectedTitles,
                      i18n.resolvedLanguage ?? "en-US"
                    ),
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
