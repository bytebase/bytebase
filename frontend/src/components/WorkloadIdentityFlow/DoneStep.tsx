import { Trans, useTranslation } from "react-i18next";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/app/router/handles";
import { RouterLink } from "@/components/RouterLink";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { FormFieldGroup } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { useServerState } from "@/hooks/useAppState";
import { writeTextToClipboard } from "@/lib/clipboard";
import {
  EXTERNAL_URL_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/lib/productIntro";
import { useAppStore } from "@/stores/app";
import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/workload_identity_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { Field, InlineCode, StepHeading } from "./StepParts";
import { type SubjectFields, subjectEnvironment } from "./subjectPattern";
import {
  buildGitHubTestWorkflow,
  buildGitLabTestJob,
  GITHUB_TEST_WORKFLOW_PATH,
} from "./testWorkflow";

// Stands in for the External URL in the test file until one is set; a run with
// it fails at once on the unknown host.
const EXTERNAL_URL_PLACEHOLDER = "https://YOUR-BYTEBASE-URL";

interface DoneStepProps {
  readonly email: string;
  // An existing identity was updated, not created.
  readonly updated?: boolean;
  readonly providerType: WorkloadIdentityConfig_ProviderType;
  readonly fields: SubjectFields;
  readonly subjectPattern: string;
  readonly audience: string;
  // The project's title, or undefined at workspace level.
  readonly projectTitle?: string;
  // Why granting the role failed, when it did.
  readonly roleError?: string;
}

export function DoneStep(props: DoneStepProps) {
  const { t } = useTranslation();
  const { externalUrl, needConfigureExternalUrl } = useServerState();
  const { providerType, fields, projectTitle } = props;
  const isProject = projectTitle !== undefined;
  const isGitHub = providerType === WorkloadIdentityConfig_ProviderType.GITHUB;
  const isGitLab = providerType === WorkloadIdentityConfig_ProviderType.GITLAB;

  const target = fields.repo
    ? `${fields.owner}/${fields.repo}`
    : isGitLab
      ? t("workload-identity.done.target-any-project", { owner: fields.owner })
      : t("workload-identity.done.target-any-repository", {
          owner: fields.owner,
        });
  const isRepository = isGitHub || isGitLab;
  // Literal keys per case, so the i18n check can see each one.
  const description = props.updated
    ? isRepository
      ? isProject
        ? t("workload-identity.done.updated-project", {
            target,
            project: projectTitle,
          })
        : t("workload-identity.done.updated-workspace", { target })
      : isProject
        ? t("workload-identity.done.updated-oidc-project", {
            pattern: props.subjectPattern,
            project: projectTitle,
          })
        : t("workload-identity.done.updated-oidc-workspace", {
            pattern: props.subjectPattern,
          })
    : isRepository
      ? isProject
        ? t("workload-identity.done.description-project", {
            target,
            project: projectTitle,
          })
        : t("workload-identity.done.description-workspace", { target })
      : isProject
        ? t("workload-identity.done.description-oidc-project", {
            pattern: props.subjectPattern,
            project: projectTitle,
          })
        : t("workload-identity.done.description-oidc-workspace", {
            pattern: props.subjectPattern,
          });

  // A pattern pinned to one environment needs the test job to run in it.
  const environment = isGitHub
    ? subjectEnvironment(props.subjectPattern)
    : undefined;
  const testParams = {
    environment: environment === "*" ? undefined : environment,
    externalUrl: needConfigureExternalUrl
      ? EXTERNAL_URL_PLACEHOLDER
      : externalUrl,
    email: props.email,
    audience: props.audience,
  };
  const canConfigureExternalUrl = hasWorkspacePermissionV2(
    "bb.settings.setWorkspaceProfile"
  );

  const copyEmail = async () => {
    const ok = await writeTextToClipboard(props.email);
    useAppStore.getState().notify({
      module: "bytebase",
      style: ok ? "SUCCESS" : "CRITICAL",
      title: ok ? t("common.copied") : t("common.copy-failed"),
    });
  };

  return (
    <div className="flex flex-col gap-y-4">
      <StepHeading
        question={
          props.updated
            ? t("workload-identity.done.title-updated")
            : t("workload-identity.done.title")
        }
        description={description}
      />

      {props.roleError && (
        <Alert
          variant="warning"
          description={t("workload-identity.done.role-not-granted", {
            error: props.roleError,
          })}
        />
      )}

      {/* On edit the banner already shows the email. */}
      {!props.updated && (
        <FormFieldGroup density="compact">
          <Field
            label={t("common.email")}
            hint={
              <Trans
                i18nKey="workload-identity.done.email-hint"
                components={{ code: <InlineCode /> }}
              />
            }
          >
            <div className="flex min-h-9 items-center justify-between gap-x-2 rounded-xs border border-control-border bg-control-bg px-3 py-1.5">
              <span className="font-mono text-sm break-all text-main">
                {props.email}
              </span>
              <Button
                type="button"
                appearance="link"
                size="sm"
                className="h-auto shrink-0 p-0 text-sm font-medium"
                onClick={() => void copyEmail()}
              >
                {t("common.copy")}
              </Button>
            </div>
          </Field>
        </FormFieldGroup>
      )}

      {isRepository && (
        <>
          <Separator className="mt-2" />
          <section className="flex flex-col gap-y-4">
            <StepHeading
              question={t("workload-identity.done.test-title")}
              optional
              description={t("workload-identity.done.test-description")}
            />
            {needConfigureExternalUrl && (
              <Alert
                variant="warning"
                className="py-1.5"
                description={
                  <>
                    {t("workload-identity.done.external-url-placeholder", {
                      placeholder: EXTERNAL_URL_PLACEHOLDER,
                    })}
                    {canConfigureExternalUrl && (
                      <>
                        {" "}
                        <RouterLink
                          to={{
                            name: SETTING_ROUTE_WORKSPACE_GENERAL,
                            hash: "#external-url",
                            query: {
                              [PRODUCT_INTRO_QUERY_KEY]:
                                EXTERNAL_URL_PRODUCT_INTRO,
                            },
                          }}
                          className="font-medium text-accent hover:underline"
                        >
                          {t("common.configure-now")}
                        </RouterLink>
                      </>
                    )}
                  </>
                }
              />
            )}
            <CodeBlock
              fileName={isGitHub ? GITHUB_TEST_WORKFLOW_PATH : ".gitlab-ci.yml"}
              fileHint={
                isGitHub
                  ? undefined
                  : t("workload-identity.done.gitlab-file-hint")
              }
              code={
                isGitHub
                  ? buildGitHubTestWorkflow(testParams)
                  : buildGitLabTestJob(testParams)
              }
            />
            <p className="text-xs text-control-light">
              {isGitHub
                ? t("workload-identity.done.github-caption", {
                    repository: `${fields.owner}/${fields.repo || "REPOSITORY"}`,
                  })
                : t("workload-identity.done.gitlab-caption")}
            </p>
          </section>
        </>
      )}
    </div>
  );
}

// One file, so its name is a plain label in the header, not a tab; it wraps on
// a narrow screen instead of being cut off.
function CodeBlock({
  fileName,
  fileHint,
  code,
}: {
  readonly fileName: string;
  readonly fileHint?: string;
  readonly code: string;
}) {
  return (
    <div className="overflow-hidden rounded-sm border border-block-border">
      <div
        data-slot="code-block-header"
        className="flex items-center justify-between gap-x-2 border-b border-block-border bg-background py-1.5 pr-2 pl-3"
      >
        <span className="flex min-w-0 flex-wrap items-baseline gap-x-1.5">
          <span className="font-mono text-xs wrap-anywhere text-main">
            {fileName}
          </span>
          {fileHint && (
            <span className="text-xs text-control-light">{fileHint}</span>
          )}
        </span>
        <div className="shrink-0">
          <CopyButton content={code} size="sm" />
        </div>
      </div>
      <pre className="overflow-x-auto bg-control-bg p-3 font-mono text-xs whitespace-pre text-main">
        {code}
      </pre>
    </div>
  );
}
