import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { FormField, FormFieldGroup, FormLabel } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { SetupNote, SetupScreenFrame } from "./SetupScreenFrame";

export function WorkspaceScreen({
  name,
  workspaceTitle,
  showWorkspaceTitle,
  onNameChange,
  onWorkspaceTitleChange,
  onContinue,
}: {
  readonly name: string;
  readonly workspaceTitle: string;
  readonly showWorkspaceTitle: boolean;
  readonly onNameChange: (value: string) => void;
  readonly onWorkspaceTitleChange: (value: string) => void;
  readonly onContinue: () => void;
}) {
  const { t } = useTranslation();
  const missingWorkspaceTitle = showWorkspaceTitle && !workspaceTitle.trim();

  return (
    <SetupScreenFrame
      screen="workspace"
      title={t("workspace-setup.workspace.title")}
      description={t("workspace-setup.workspace.description")}
      onSubmit={() => {
        if (!missingWorkspaceTitle) onContinue();
      }}
      footer={
        <Button
          type="submit"
          className="ml-auto"
          disabled={missingWorkspaceTitle}
        >
          {t("workspace-setup.continue")}
        </Button>
      }
    >
      <FormFieldGroup>
        <FormField>
          <FormLabel htmlFor="workspace-setup-name">
            {t("settings.profile.display-name")}
            <span className="ml-1 font-normal text-control-light">
              {t("workspace-setup.optional")}
            </span>
          </FormLabel>
          <Input
            id="workspace-setup-name"
            value={name}
            autoComplete="off"
            placeholder={t("workspace-setup.workspace.name-placeholder")}
            onChange={(event) => onNameChange(event.target.value)}
          />
        </FormField>
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
              aria-describedby="workspace-setup-workspace-title-note"
              onChange={(event) => onWorkspaceTitleChange(event.target.value)}
            />
            <SetupNote id="workspace-setup-workspace-title-note">
              {missingWorkspaceTitle
                ? t("workspace-setup.workspace.workspace-name-required")
                : t("workspace-setup.workspace.workspace-name-hint")}
            </SetupNote>
          </FormField>
        )}
      </FormFieldGroup>
    </SetupScreenFrame>
  );
}
