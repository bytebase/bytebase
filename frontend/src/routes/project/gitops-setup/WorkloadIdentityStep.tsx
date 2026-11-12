import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { PROJECT_V1_ROUTE_MEMBERS } from "@/app/router/handles";
import { RouterLink } from "@/components/RouterLink";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Field,
  StepHeading,
} from "@/components/WorkloadIdentityFlow/StepParts";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/workload_identity_service_pb";
import { getWorkloadIdentityProviderText } from "@/utils";
import { displayRoleTitle } from "@/utils/role";
import { WorkloadIdentitySelect } from "../WorkloadIdentitySelect";
import { repositoriesOf, workflowProviderOf } from "./identity";
import { ProviderLogo } from "./ProviderLogo";

// Step 2: the workload identity the workflow signs in as, and what setup reads
// from it.
export function WorkloadIdentityStep({
  projectId,
  projectName,
  hasIdentities,
  canCreate,
  identityName,
  identity,
  roles,
  onChange,
  onCreate,
}: {
  readonly projectId: string;
  readonly projectName: string;
  // The project has at least one workload identity, or one was just created.
  readonly hasIdentities: boolean;
  readonly canCreate: boolean;
  readonly identityName: string;
  readonly identity: WorkloadIdentity | undefined;
  // Undefined when the project's roles cannot be read.
  readonly roles: string[] | undefined;
  readonly onChange: (name: string) => void;
  readonly onCreate: () => void;
}) {
  const { t } = useTranslation();
  const repositoriesText = useRepositoriesText();
  const disabledReason = useCallback(
    (wi: WorkloadIdentity) =>
      workflowProviderOf(wi)
        ? undefined
        : t("gitops.setup.identity.other-provider"),
    [t]
  );
  const describe = useCallback(
    (wi: WorkloadIdentity) =>
      `${getWorkloadIdentityProviderText(
        wi.workloadIdentityConfig?.providerType ??
          WorkloadIdentityConfig_ProviderType.PROVIDER_TYPE_UNSPECIFIED
      )} · ${repositoriesText(wi)}`,
    [repositoriesText]
  );

  const provider = workflowProviderOf(identity);

  const createButton = (appearance: "solid" | "outline") =>
    canCreate && (
      <Button appearance={appearance} onClick={onCreate}>
        {t("gitops.setup.identity.create")}
      </Button>
    );

  return (
    <div className="flex flex-col gap-y-4">
      <StepHeading
        question={t("gitops.setup.identity.question")}
        description={t("gitops.setup.identity.description")}
      />

      {!hasIdentities && !identityName ? (
        <div className="flex flex-col items-start gap-y-3 rounded-sm border border-block-border p-4">
          <p className="text-sm text-control-light">
            {t("gitops.setup.identity.none")}
          </p>
          {createButton("solid")}
        </div>
      ) : (
        <Field label={t("gitops.setup.identity.label")}>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <WorkloadIdentitySelect
              portal
              projectName={projectName}
              value={identityName}
              placeholder={t("gitops.setup.identity.placeholder")}
              describe={describe}
              disabledReason={disabledReason}
              onChange={(name) => onChange(name)}
              className="min-w-0 flex-1"
            />
            <div className="shrink-0">{createButton("outline")}</div>
          </div>
        </Field>
      )}

      {identity && (
        <section className="flex flex-col gap-y-2">
          <h4 className="text-sm font-medium text-main">
            {t("gitops.setup.identity.read-from")}
          </h4>
          <dl className="flex flex-col gap-y-2 rounded-sm border border-block-border bg-control-bg/50 p-3 text-sm">
            <Fact label={t("gitops.setup.identity.git-provider")}>
              <span className="flex items-center gap-x-1.5">
                {provider && <ProviderLogo provider={provider} />}
                {getWorkloadIdentityProviderText(
                  identity.workloadIdentityConfig?.providerType ??
                    WorkloadIdentityConfig_ProviderType.PROVIDER_TYPE_UNSPECIFIED,
                  t("settings.members.workload-identity-generic-oidc")
                )}
              </span>
            </Fact>
            <Fact label={t("gitops.setup.identity.repositories")}>
              {repositoriesText(identity)}
            </Fact>
            <Fact label={t("gitops.setup.identity.role")}>
              {roles === undefined
                ? t("common.unknown")
                : roles.length === 0
                  ? "-"
                  : roles.map(displayRoleTitle).join(", ")}
            </Fact>
          </dl>
        </section>
      )}

      {identity && provider && roles?.length === 0 && (
        <Alert
          variant="warning"
          title={t("gitops.setup.identity.no-role-title")}
          description={t("gitops.setup.identity.no-role-description")}
        >
          <RouterLink
            to={{ name: PROJECT_V1_ROUTE_MEMBERS, params: { projectId } }}
            className="text-sm font-medium text-accent hover:underline"
          >
            {t("gitops.setup.identity.open-members")}
          </RouterLink>
        </Alert>
      )}
    </div>
  );
}

function useRepositoriesText() {
  const { t } = useTranslation();
  return useCallback(
    (identity: WorkloadIdentity) => {
      const { owner, repo } = repositoriesOf(identity);
      if (owner && repo) return `${owner}/${repo}`;
      if (owner) return t("gitops.setup.identity.every-repository", { owner });
      return identity.workloadIdentityConfig?.subjectPattern || "-";
    },
    [t]
  );
}

function Fact({
  label,
  children,
}: {
  readonly label: string;
  readonly children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-x-4 gap-y-1 sm:flex-row">
      <dt className="text-control-light sm:w-56 sm:shrink-0">{label}</dt>
      <dd className="min-w-0 wrap-anywhere text-main">{children}</dd>
    </div>
  );
}
