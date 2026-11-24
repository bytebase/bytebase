import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FormField, FormFieldGroup, FormLabel } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { SetupNote, SetupScreenFrame } from "./SetupScreenFrame";

export function WorkspaceScreen({
  name,
  showName,
  workspaceTitle,
  showWorkspaceTitle,
  onNameChange,
  onWorkspaceTitleChange,
  onContinue,
  onSkip,
  skipping,
  skipFailure,
}: {
  readonly name: string;
  // Cloud only: a self-hosted account named itself at sign-up.
  readonly showName: boolean;
  readonly workspaceTitle: string;
  readonly showWorkspaceTitle: boolean;
  readonly onNameChange: (value: string) => void;
  readonly onWorkspaceTitleChange: (value: string) => void;
  readonly onContinue: () => void;
  // Self-hosted only, while nothing has been created yet.
  readonly onSkip: (() => void) | undefined;
  readonly skipping: boolean;
  readonly skipFailure: { detail?: string } | undefined;
}) {
  const { t } = useTranslation();
  const missingWorkspaceTitle = showWorkspaceTitle && !workspaceTitle.trim();
  const missingName = showName && !name.trim();
  const blocked = missingWorkspaceTitle || missingName;

  return (
    <SetupScreenFrame
      screen="workspace"
      title={t("workspace-setup.workspace.title")}
      description={t("workspace-setup.workspace.description")}
      onSubmit={() => {
        if (!blocked && !skipping) onContinue();
      }}
      footer={
        <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
          {onSkip && (
            <Button
              type="button"
              appearance="secondary"
              disabled={skipping}
              onClick={onSkip}
            >
              {t("workspace-setup.workspace.skip")}
            </Button>
          )}
          <Button type="submit" disabled={blocked || skipping}>
            {t("workspace-setup.continue")}
          </Button>
        </div>
      }
    >
      <FormFieldGroup>
        {showName && (
          <FormField>
            <FormLabel htmlFor="workspace-setup-name">
              {t("settings.profile.display-name")}
            </FormLabel>
            <Input
              id="workspace-setup-name"
              value={name}
              autoComplete="off"
              placeholder={t("workspace-setup.workspace.name-placeholder")}
              aria-describedby={
                missingName ? "workspace-setup-name-note" : undefined
              }
              onChange={(event) => onNameChange(event.target.value)}
            />
            {missingName && (
              <SetupNote id="workspace-setup-name-note">
                {t("workspace-setup.workspace.name-required")}
              </SetupNote>
            )}
          </FormField>
        )}
        {showWorkspaceTitle && (
          <FormField>
            <FormLabel htmlFor="workspace-setup-workspace-title">
              {t("workspace-setup.workspace.workspace-name")}
            </FormLabel>
            <Input
              id="workspace-setup-workspace-title"
              value={workspaceTitle}
              autoComplete="off"
              placeholder={t(
                "workspace-setup.workspace.workspace-name-placeholder"
              )}
              aria-describedby={
                missingWorkspaceTitle
                  ? "workspace-setup-workspace-title-note"
                  : undefined
              }
              onChange={(event) => onWorkspaceTitleChange(event.target.value)}
            />
            {missingWorkspaceTitle && (
              <SetupNote id="workspace-setup-workspace-title-note">
                {t("workspace-setup.workspace.workspace-name-required")}
              </SetupNote>
            )}
          </FormField>
        )}
      </FormFieldGroup>
      {skipFailure && (
        <Alert
          variant="error"
          title={
            showName
              ? t("workspace-setup.team.failed.names")
              : t("workspace-setup.team.failed.workspace-name")
          }
          description={skipFailure.detail}
        />
      )}
    </SetupScreenFrame>
  );
}
