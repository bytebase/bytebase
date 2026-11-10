import { ExternalLink } from "lucide-react";
import { useId } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { FormFieldGroup } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/workload_identity_service_pb";
import { Field, InlineCode, StepHeading, TextAction } from "./StepParts";
import { type SubjectFields, subjectEnvironment } from "./subjectPattern";

// GitHub's announcement of ID-based subjects, which explains why a name alone
// can be taken over. The one external link in the dialog.
const GITHUB_IMMUTABLE_SUBJECT_URL =
  "https://github.blog/changelog/2026-04-23-immutable-subject-claims-for-github-actions-oidc-tokens/";

export interface RepositoriesErrors {
  owner?: string;
  ownerId?: string;
  repo?: string;
  subjectPattern?: string;
}

interface RepositoriesStepProps {
  readonly providerType: WorkloadIdentityConfig_ProviderType;
  readonly fields: SubjectFields;
  readonly onFieldsChange: (fields: SubjectFields) => void;
  // The pattern names something the fields cannot read.
  readonly fieldsDisabled: boolean;
  readonly subjectPattern: string;
  readonly onSubjectPatternChange: (value: string) => void;
  readonly patternEditing: boolean;
  readonly onPatternEdit: () => void;
  readonly onPatternReset: () => void;
  readonly errors: RepositoriesErrors;
}

export function RepositoriesStep(props: RepositoriesStepProps) {
  const { t } = useTranslation();
  const ownerId = useId();
  const ownerIdId = useId();
  const repoId = useId();
  const patternId = useId();
  const { providerType, fields, errors } = props;
  const isOIDC = providerType === WorkloadIdentityConfig_ProviderType.OIDC;
  const isGitHub = providerType === WorkloadIdentityConfig_ProviderType.GITHUB;

  if (isOIDC) {
    return (
      <div className="flex flex-col gap-y-4">
        <StepHeading
          question={t("workload-identity.repositories.oidc-question")}
        />
        <Field
          id={patternId}
          label={t("workload-identity.repositories.subject-pattern")}
          hint={
            <Trans
              i18nKey="workload-identity.repositories.oidc-pattern-hint"
              components={{ code: <InlineCode /> }}
            />
          }
          error={errors.subjectPattern}
        >
          <Input
            id={patternId}
            value={props.subjectPattern}
            onChange={(event) =>
              props.onSubjectPatternChange(event.target.value)
            }
            placeholder="system:serviceaccount:ci:deployer"
            maxLength={500}
            autoComplete="off"
            className="font-mono"
            aria-invalid={!!errors.subjectPattern}
          />
        </Field>
      </div>
    );
  }

  const set = (patch: Partial<SubjectFields>) =>
    props.onFieldsChange({ ...fields, ...patch });
  const environment = isGitHub
    ? subjectEnvironment(props.subjectPattern)
    : undefined;

  return (
    <div className="flex flex-col gap-y-4">
      <StepHeading
        question={t("workload-identity.repositories.question")}
        description={t("workload-identity.repositories.description")}
      />

      <FormFieldGroup density="compact">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field
            id={ownerId}
            label={
              isGitHub
                ? t("workload-identity.repositories.owner")
                : t("workload-identity.repositories.group")
            }
            hint={
              isGitHub
                ? t("workload-identity.repositories.owner-hint")
                : t("workload-identity.repositories.group-hint")
            }
            error={errors.owner}
          >
            <Input
              id={ownerId}
              value={fields.owner}
              disabled={props.fieldsDisabled}
              onChange={(event) => set({ owner: event.target.value.trim() })}
              placeholder={isGitHub ? "acme-corp" : "acme"}
              maxLength={200}
              autoComplete="off"
              aria-invalid={!!errors.owner}
            />
          </Field>

          <Field
            id={repoId}
            label={
              isGitHub
                ? t("workload-identity.repositories.repository")
                : t("workload-identity.repositories.project")
            }
            optional
            hint={
              isGitHub
                ? t("workload-identity.repositories.repository-hint")
                : t("workload-identity.repositories.project-hint")
            }
            error={errors.repo}
          >
            <Input
              id={repoId}
              value={fields.repo}
              disabled={props.fieldsDisabled}
              onChange={(event) => set({ repo: event.target.value.trim() })}
              placeholder="deploy"
              maxLength={200}
              autoComplete="off"
              aria-invalid={!!errors.repo}
            />
          </Field>
        </div>

        {isGitHub && (
          <Field
            id={ownerIdId}
            label={t("workload-identity.repositories.owner-id")}
            optional
            hint={
              <Trans
                i18nKey="workload-identity.repositories.owner-id-hint"
                values={{ owner: fields.owner || "OWNER" }}
                components={{ code: <InlineCode /> }}
              />
            }
            error={errors.ownerId}
          >
            <Input
              id={ownerIdId}
              value={fields.ownerId}
              disabled={props.fieldsDisabled}
              onChange={(event) => set({ ownerId: event.target.value.trim() })}
              placeholder="1234567"
              inputMode="numeric"
              maxLength={20}
              autoComplete="off"
              aria-invalid={!!errors.ownerId}
            />
          </Field>
        )}

        {/* Always shown for GitHub with the same text, so typing never
            changes the step's height; only the style follows Owner ID. */}
        {isGitHub && (
          <Alert
            variant={fields.ownerId ? "info" : "warning"}
            title={t("workload-identity.repositories.owner-id-note-title")}
            description={
              <>
                <p>
                  {t("workload-identity.repositories.owner-id-note")}{" "}
                  <a
                    href={GITHUB_IMMUTABLE_SUBJECT_URL}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-x-1 font-medium text-accent hover:underline"
                  >
                    {t("workload-identity.repositories.owner-id-note-link")}
                    <ExternalLink className="size-3" />
                  </a>
                </p>
              </>
            }
          />
        )}

        <Field
          id={patternId}
          label={t("workload-identity.repositories.subject-pattern")}
          action={
            props.patternEditing ? (
              <TextAction onClick={props.onPatternReset}>
                {t("common.reset")}
              </TextAction>
            ) : (
              <TextAction onClick={props.onPatternEdit}>
                {t("common.edit")}
              </TextAction>
            )
          }
          hint={
            props.fieldsDisabled
              ? t("workload-identity.repositories.pattern-unreadable")
              : props.patternEditing
                ? isGitHub
                  ? t(
                      "workload-identity.repositories.github-pattern-editing-hint"
                    )
                  : t(
                      "workload-identity.repositories.gitlab-pattern-editing-hint"
                    )
                : t("workload-identity.repositories.pattern-hint")
          }
          error={errors.subjectPattern}
        >
          <Input
            id={patternId}
            value={props.subjectPattern}
            readOnly={!props.patternEditing}
            onChange={(event) =>
              props.onSubjectPatternChange(event.target.value)
            }
            maxLength={500}
            autoComplete="off"
            className="font-mono"
            aria-invalid={!!errors.subjectPattern}
          />
        </Field>

        {environment !== undefined && (
          <Alert
            variant="warning"
            description={
              environment === "*" ? (
                t("workload-identity.repositories.any-environment-warning")
              ) : (
                <Trans
                  i18nKey="workload-identity.repositories.environment-warning"
                  values={{ environment }}
                  components={{ code: <InlineCode /> }}
                />
              )
            }
          />
        )}
      </FormFieldGroup>
    </div>
  );
}
