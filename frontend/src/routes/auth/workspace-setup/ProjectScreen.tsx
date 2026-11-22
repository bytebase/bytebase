import { useTranslation } from "react-i18next";
import { ResourceIdField } from "@/components/ResourceIdField";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  FormError,
  FormField,
  FormFieldGroup,
  FormLabel,
  FormTitle,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { cn } from "@/lib/utils";
import type { ValidatedMessage } from "@/types";
import type { SetupSource } from "./draft";
import {
  choiceCardClassName,
  SetupNote,
  SetupScreenFrame,
} from "./SetupScreenFrame";

// The sample is single-use per workspace, and a server may not offer one.
export type SampleOffer = "offered" | "used" | "unavailable";

export function ProjectScreen({
  projectTitle,
  projectId,
  projectIdValid,
  projectIdRefusal,
  projectExists,
  source,
  sample,
  samplePrepared,
  isSaaS,
  projectOptional,
  skipping,
  skipFailure,
  validateProjectId,
  onProjectTitleChange,
  onProjectIdChange,
  onProjectIdValidChange,
  onSourceChange,
  onBack,
  onContinue,
  onSkip,
}: {
  readonly projectTitle: string;
  readonly projectId: string;
  readonly projectIdValid: boolean;
  readonly projectIdRefusal: string | undefined;
  // A Finish attempt created the project, so its name and ID are fixed here.
  readonly projectExists: boolean;
  readonly source: SetupSource;
  readonly sample: SampleOffer;
  readonly samplePrepared: boolean;
  readonly isSaaS: boolean;
  // Left empty, no project is made, and so no database is offered (D12).
  readonly projectOptional: boolean;
  readonly skipping: boolean;
  readonly skipFailure: { detail?: string } | undefined;
  readonly validateProjectId: (id: string) => Promise<ValidatedMessage[]>;
  readonly onProjectTitleChange: (value: string) => void;
  readonly onProjectIdChange: (value: string) => void;
  readonly onProjectIdValidChange: (valid: boolean) => void;
  readonly onSourceChange: (source: SetupSource) => void;
  readonly onBack: () => void;
  readonly onContinue: () => void;
  // Offered only where setup is optional and nothing has been created yet.
  readonly onSkip: (() => void) | undefined;
}) {
  const { t } = useTranslation();
  const named = !!projectTitle.trim();
  const ready =
    projectExists ||
    (named
      ? !!projectId && projectIdValid && !projectIdRefusal
      : projectOptional);
  const showSample = samplePrepared || sample === "offered";
  const projectLabel = named
    ? projectTitle.trim()
    : t("workspace-setup.project.your-project");
  const databaseTitle = t("workspace-setup.project.database-title", {
    project: projectLabel,
  });

  return (
    <SetupScreenFrame
      screen="project"
      title={t("workspace-setup.project.title")}
      description={t("workspace-setup.project.description")}
      onSubmit={() => {
        if (ready) onContinue();
      }}
      footer={
        <>
          <Button
            type="button"
            appearance="secondary"
            disabled={skipping}
            onClick={onBack}
          >
            {t("common.back")}
          </Button>
          <div className="flex flex-wrap items-center justify-end gap-2">
            {onSkip && (
              <Button
                type="button"
                appearance="secondary"
                disabled={skipping}
                onClick={onSkip}
              >
                {t("workspace-setup.project.skip")}
              </Button>
            )}
            <Button type="submit" disabled={!ready || skipping}>
              {t("workspace-setup.continue")}
            </Button>
          </div>
        </>
      }
    >
      <FormFieldGroup>
        <FormField>
          <FormLabel htmlFor="workspace-setup-project-title">
            {t("workspace-setup.project.name")}
            {projectOptional && (
              <span className="ml-1 font-normal text-control-light">
                {t("workspace-setup.optional")}
              </span>
            )}
          </FormLabel>
          <Input
            id="workspace-setup-project-title"
            value={projectTitle}
            autoComplete="off"
            disabled={projectExists}
            placeholder={t("workspace-setup.project.name-placeholder")}
            aria-describedby={cn(
              "workspace-setup-project-title-hint",
              !named && "workspace-setup-project-title-note"
            )}
            onChange={(event) => onProjectTitleChange(event.target.value)}
          />
          <SetupNote id="workspace-setup-project-title-hint">
            {projectExists
              ? t("workspace-setup.project.exists")
              : t("workspace-setup.project.name-hint")}
          </SetupNote>
          {named ? (
            <ResourceIdField
              suffix
              keepInitialValue
              readonly={projectExists}
              value={projectId}
              autoComplete="off"
              resourceName={t("common.project")}
              resourceTitle={projectTitle}
              validate={validateProjectId}
              onChange={onProjectIdChange}
              onValidationChange={onProjectIdValidChange}
            />
          ) : (
            <SetupNote id="workspace-setup-project-title-note">
              {projectOptional
                ? t("workspace-setup.project.name-optional-note")
                : t("workspace-setup.project.name-required")}
            </SetupNote>
          )}
          {/* The field shows its own reason when its checks fail. */}
          {named && projectIdRefusal && projectIdValid && (
            <FormError>{projectIdRefusal}</FormError>
          )}
        </FormField>
        {(named || !projectOptional) && (
          <FormField>
            <FormTitle>{databaseTitle}</FormTitle>
            <RadioGroup
              aria-label={databaseTitle}
              value={source}
              disabled={samplePrepared}
              onValueChange={(value) => {
                if (value === "sample" || value === "own")
                  onSourceChange(value);
              }}
              className="flex-col items-stretch gap-y-2"
            >
              {showSample && (
                <RadioGroupItem
                  value="sample"
                  className={choiceCardClassName(source === "sample")}
                  radioClassName="mt-0.5"
                  contentClassName="min-w-0"
                >
                  <div className="flex flex-wrap items-center gap-x-2 font-medium text-main">
                    {t("workspace-setup.project.sample.title")}
                    <Badge variant="secondary">
                      {t("workspace-setup.project.sample.recommended")}
                    </Badge>
                  </div>
                  <div className="mt-1 text-control-light">
                    {isSaaS
                      ? t("workspace-setup.project.sample.description")
                      : t(
                          "workspace-setup.project.sample.description-self-host"
                        )}
                  </div>
                </RadioGroupItem>
              )}
              <RadioGroupItem
                value="own"
                className={choiceCardClassName(source === "own")}
                radioClassName="mt-0.5"
                contentClassName="min-w-0"
              >
                <div className="font-medium text-main">
                  {t("workspace-setup.project.own.title")}
                </div>
                <div className="mt-1 text-control-light">
                  {t("workspace-setup.project.own.description", {
                    project: projectLabel,
                  })}
                </div>
              </RadioGroupItem>
            </RadioGroup>
            {samplePrepared ? (
              <SetupNote>
                {t("workspace-setup.project.sample.prepared")}
              </SetupNote>
            ) : (
              sample !== "offered" && (
                <SetupNote>
                  {sample === "used"
                    ? t("workspace-setup.project.sample.used")
                    : t("workspace-setup.project.sample.unavailable")}
                </SetupNote>
              )
            )}
          </FormField>
        )}
      </FormFieldGroup>
      {skipFailure && (
        <Alert
          variant="error"
          title={t("workspace-setup.team.failed.names")}
          description={skipFailure.detail}
        />
      )}
    </SetupScreenFrame>
  );
}
