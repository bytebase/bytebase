import { KeyRound } from "lucide-react";
import { useId } from "react";
import { Trans, useTranslation } from "react-i18next";
import githubIcon from "@/assets/git/github.svg";
import gitlabIcon from "@/assets/git/gitlab.svg";
import { FormError, FormFieldGroup } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Separator } from "@/components/ui/separator";
import { TagInput } from "@/components/ui/tag-input";
import { cn } from "@/lib/utils";
import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/workload_identity_service_pb";
import { Field, InlineCode, StepHeading, TextAction } from "./StepParts";
import { isRepositoryProvider } from "./subjectPattern";

export interface TokenSourceErrors {
  provider?: string;
  issuerUrl?: string;
  audiences?: string;
  jwksUrl?: string;
}

interface TokenSourceStepProps {
  readonly providerType?: WorkloadIdentityConfig_ProviderType;
  readonly providerLocked: boolean;
  // Offer GitHub Actions and GitLab CI only.
  readonly gitProvidersOnly?: boolean;
  readonly onProviderChange: (
    providerType: WorkloadIdentityConfig_ProviderType
  ) => void;
  readonly issuerUrl: string;
  readonly onIssuerUrlChange: (value: string) => void;
  readonly issuerEditing: boolean;
  readonly onIssuerEdit: () => void;
  readonly onIssuerReset: () => void;
  readonly audiences: string[];
  readonly onAudiencesChange: (value: string[]) => void;
  readonly audiencesEditing: boolean;
  readonly onAudiencesEdit: () => void;
  readonly onAudiencesReset: () => void;
  readonly jwksUrl: string;
  readonly onJwksUrlChange: (value: string) => void;
  readonly errors: TokenSourceErrors;
}

export function TokenSourceStep(props: TokenSourceStepProps) {
  const { t } = useTranslation();
  const issuerId = useId();
  const audiencesId = useId();
  const jwksId = useId();
  const { providerType, errors } = props;
  const isOIDC = providerType === WorkloadIdentityConfig_ProviderType.OIDC;
  const isGitLab = providerType === WorkloadIdentityConfig_ProviderType.GITLAB;
  // Other OIDC has no preset to show read-only.
  const issuerEditing = isOIDC || props.issuerEditing;
  const audiencesEditing = isOIDC || props.audiencesEditing;
  const providers = [
    {
      value: WorkloadIdentityConfig_ProviderType.GITHUB,
      icon: <img src={githubIcon} alt="" className="size-6" />,
      title: t("workload-identity.token-source.github-title"),
      subtitle: t("workload-identity.token-source.github-subtitle"),
    },
    {
      value: WorkloadIdentityConfig_ProviderType.GITLAB,
      icon: <img src={gitlabIcon} alt="" className="size-6" />,
      title: t("workload-identity.token-source.gitlab-title"),
      subtitle: t("workload-identity.token-source.gitlab-subtitle"),
    },
    {
      value: WorkloadIdentityConfig_ProviderType.OIDC,
      icon: <KeyRound className="size-5 text-main" />,
      title: t("workload-identity.token-source.oidc-title"),
      subtitle: t("workload-identity.token-source.oidc-subtitle"),
    },
  ].filter(
    (provider) =>
      !props.gitProvidersOnly || isRepositoryProvider(provider.value)
  );

  return (
    <div className="flex flex-col gap-y-4">
      <StepHeading
        question={t("workload-identity.token-source.question")}
        description={t("workload-identity.token-source.description")}
      />

      <div className="flex flex-col gap-y-1.5">
        <RadioGroup
          aria-label={t("workload-identity.rail.token-source")}
          value={providerType === undefined ? "" : String(providerType)}
          disabled={props.providerLocked}
          onValueChange={(value) =>
            props.onProviderChange(
              Number(value) as WorkloadIdentityConfig_ProviderType
            )
          }
          className={cn(
            "grid grid-cols-1 items-stretch gap-2",
            providers.length === 2 ? "sm:grid-cols-2" : "sm:grid-cols-3"
          )}
        >
          {providers.map((provider) => {
            const selected = providerType === provider.value;
            return (
              <RadioGroupItem
                key={provider.value}
                value={String(provider.value)}
                // The card is the visible control; the radio keeps the group's
                // semantics and keyboard handling.
                radioClassName="sr-only"
                contentClassName="flex w-full flex-col items-center gap-y-1 text-center"
                className={cn(
                  "justify-center rounded-sm border px-3 py-4",
                  "has-[[role=radio]:focus-visible]:ring-2 has-[[role=radio]:focus-visible]:ring-accent",
                  selected
                    ? "border-accent bg-accent/5 ring-1 ring-accent ring-inset"
                    : "border-control-border hover:bg-control-bg",
                  props.providerLocked && !selected && "opacity-50"
                )}
              >
                <span className="mb-1 flex size-10 items-center justify-center rounded-sm bg-control-bg">
                  {provider.icon}
                </span>
                <span className="font-medium text-main">{provider.title}</span>
                <span className="text-xs text-control-light">
                  {provider.subtitle}
                </span>
              </RadioGroupItem>
            );
          })}
        </RadioGroup>
        {errors.provider && <FormError>{errors.provider}</FormError>}
      </div>

      {providerType !== undefined && (
        <>
          <Separator />
          <FormFieldGroup density="compact">
            {!isOIDC && (
              <p className="text-sm text-main">
                {isGitLab
                  ? t("workload-identity.token-source.gitlab-note")
                  : t("workload-identity.token-source.github-note")}
              </p>
            )}

            <Field
              id={issuerId}
              label={t("workload-identity.token-source.issuer")}
              action={
                isOIDC ? undefined : issuerEditing ? (
                  <TextAction onClick={props.onIssuerReset}>
                    {t("common.reset")}
                  </TextAction>
                ) : (
                  <TextAction onClick={props.onIssuerEdit}>
                    {t("common.edit")}
                  </TextAction>
                )
              }
              hint={
                isOIDC
                  ? undefined
                  : isGitLab
                    ? t("workload-identity.token-source.gitlab-issuer-hint")
                    : t("workload-identity.token-source.github-issuer-hint")
              }
              error={errors.issuerUrl}
            >
              <Input
                id={issuerId}
                value={props.issuerUrl}
                readOnly={!issuerEditing}
                onChange={(event) =>
                  props.onIssuerUrlChange(event.target.value)
                }
                placeholder="https://issuer.example.com"
                maxLength={500}
                autoComplete="off"
                className="font-mono"
                aria-invalid={!!errors.issuerUrl}
              />
            </Field>

            <Field
              id={audiencesEditing ? audiencesId : undefined}
              label={t("workload-identity.token-source.audiences")}
              action={
                isOIDC ? undefined : audiencesEditing ? (
                  <TextAction onClick={props.onAudiencesReset}>
                    {t("common.reset")}
                  </TextAction>
                ) : (
                  <TextAction onClick={props.onAudiencesEdit}>
                    {t("common.edit")}
                  </TextAction>
                )
              }
              hint={
                isOIDC ? (
                  t("workload-identity.token-source.oidc-audiences-hint")
                ) : (
                  <Trans
                    i18nKey="workload-identity.token-source.audiences-hint"
                    components={{ code: <InlineCode /> }}
                  />
                )
              }
              error={errors.audiences}
            >
              <TagInput
                id={audiencesId}
                value={props.audiences}
                onChange={props.onAudiencesChange}
                readOnly={!audiencesEditing}
                placeholder={t(
                  "workload-identity.token-source.audiences-placeholder"
                )}
                maxLength={500}
                aria-invalid={!!errors.audiences}
                removeLabel={(audience) =>
                  t("workload-identity.token-source.remove-audience", {
                    audience,
                  })
                }
              />
            </Field>

            {isOIDC && (
              <Field
                id={jwksId}
                label={t("workload-identity.token-source.jwks-url")}
                optional
                hint={
                  <Trans
                    i18nKey="workload-identity.token-source.jwks-url-hint"
                    components={{ code: <InlineCode /> }}
                  />
                }
                error={errors.jwksUrl}
              >
                <Input
                  id={jwksId}
                  value={props.jwksUrl}
                  onChange={(event) =>
                    props.onJwksUrlChange(event.target.value)
                  }
                  placeholder="https://issuer.example.com/keys"
                  maxLength={500}
                  autoComplete="off"
                  className="font-mono"
                  aria-invalid={!!errors.jwksUrl}
                />
              </Field>
            )}
          </FormFieldGroup>
        </>
      )}
    </div>
  );
}
