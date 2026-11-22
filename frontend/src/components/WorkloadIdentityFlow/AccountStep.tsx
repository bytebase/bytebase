import { useId } from "react";
import { useTranslation } from "react-i18next";
import { RoleSelect } from "@/components/RoleSelect";
import { Alert } from "@/components/ui/alert";
import { FormFieldGroup } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { displayRoleTitle } from "@/utils/role";
import { Field, StepHeading, TextAction } from "./StepParts";

export interface AccountErrors {
  email?: string;
  role?: string;
}

interface AccountStepProps {
  readonly isEditMode: boolean;
  // The stored email, shown locked in edit mode.
  readonly email?: string;
  readonly emailPrefix: string;
  readonly onEmailPrefixChange: (value: string) => void;
  readonly emailSuffix: string;
  readonly title: string;
  readonly onTitleChange: (value: string) => void;
  // The role to grant, or "" when cleared after Edit.
  readonly role: string;
  readonly onRoleChange: (role: string) => void;
  readonly roleEditing: boolean;
  readonly onRoleEdit: () => void;
  readonly onRoleReset: () => void;
  readonly canSetRoles: boolean;
  // The project's title, or undefined at workspace level.
  readonly projectTitle?: string;
  readonly errors: AccountErrors;
}

export function AccountStep(props: AccountStepProps) {
  const { t } = useTranslation();
  const emailId = useId();
  const titleId = useId();
  const roleId = useId();
  const { errors, projectTitle } = props;
  const isProject = projectTitle !== undefined;

  return (
    <div className="flex flex-col gap-y-4">
      <StepHeading
        question={t("workload-identity.account.question")}
        description={
          isProject
            ? t("workload-identity.account.description-project", {
                project: projectTitle,
              })
            : t("workload-identity.account.description-workspace")
        }
      />

      <FormFieldGroup density="compact">
        <Field
          id={emailId}
          label={t("common.email")}
          hint={t("workload-identity.account.email-hint")}
          error={errors.email}
        >
          {props.isEditMode ? (
            <Input
              id={emailId}
              value={props.email ?? ""}
              readOnly
              className="font-mono"
            />
          ) : (
            <div className="flex flex-col sm:flex-row sm:items-center">
              <Input
                id={emailId}
                value={props.emailPrefix}
                onChange={(event) =>
                  props.onEmailPrefixChange(event.target.value.trim())
                }
                placeholder="deploy-ci"
                maxLength={100}
                autoComplete="off"
                aria-invalid={!!errors.email}
                className="rounded-b-none sm:rounded-r-none sm:rounded-bl-xs"
              />
              <span className="flex items-center rounded-b-xs border border-t-0 border-control-border bg-control-bg px-3 py-1.5 text-xs break-all text-control-light sm:min-h-9 sm:rounded-l-none sm:rounded-tr-xs sm:border-t sm:border-l-0 sm:py-0 sm:whitespace-nowrap">
                @{props.emailSuffix}
              </span>
            </div>
          )}
        </Field>

        <Field
          id={titleId}
          label={t("workload-identity.account.display-name")}
          optional
          hint={t("workload-identity.account.display-name-hint")}
        >
          <Input
            id={titleId}
            value={props.title}
            onChange={(event) => props.onTitleChange(event.target.value)}
            placeholder={props.emailPrefix || undefined}
            maxLength={200}
            autoComplete="off"
          />
        </Field>

        {!props.isEditMode &&
          (props.canSetRoles ? (
            <Field
              id={props.roleEditing ? undefined : roleId}
              label={t("common.role.self")}
              action={
                props.roleEditing ? (
                  <TextAction onClick={props.onRoleReset}>
                    {t("common.reset")}
                  </TextAction>
                ) : (
                  <TextAction onClick={props.onRoleEdit}>
                    {t("common.edit")}
                  </TextAction>
                )
              }
              hint={
                isProject
                  ? t("workload-identity.account.role-hint-project")
                  : t("workload-identity.account.role-hint-workspace")
              }
              error={errors.role}
            >
              {props.roleEditing ? (
                <RoleSelect
                  multiple={false}
                  clearable={false}
                  value={props.role ? [props.role] : []}
                  onChange={(roles) => props.onRoleChange(roles[0] ?? "")}
                  scope={isProject ? "project" : undefined}
                />
              ) : (
                <Input
                  id={roleId}
                  value={displayRoleTitle(props.role)}
                  readOnly
                />
              )}
            </Field>
          ) : (
            <Field label={t("common.role.self")}>
              <Alert
                variant="warning"
                description={
                  isProject
                    ? t("workload-identity.account.no-permission-project")
                    : t("workload-identity.account.no-permission-workspace")
                }
              />
            </Field>
          ))}
      </FormFieldGroup>
    </div>
  );
}
