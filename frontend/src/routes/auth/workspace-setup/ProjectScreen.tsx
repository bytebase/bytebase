import { useState } from "react";
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
import type { ValidatedMessage } from "@/types";
import type { SetupSource } from "./draft";
import { FinishStatus, type SetupFinishFailure } from "./FinishStatus";
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
  skipping,
  skipFailure,
  validateProjectId,
  onProjectTitleChange,
  onProjectIdChange,
  onProjectIdValidChange,
  onSourceChange,
  onBack,
  onContinue,
  onFinish,
  onFinishWithoutSample,
  finishing,
  failure,
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
  readonly skipping: boolean;
  readonly skipFailure: { detail?: string } | undefined;
  readonly validateProjectId: (id: string) => Promise<ValidatedMessage[]>;
  readonly onProjectTitleChange: (value: string) => void;
  readonly onProjectIdChange: (value: string) => void;
  readonly onProjectIdValidChange: (valid: boolean) => void;
  readonly onSourceChange: (source: SetupSource) => void;
  readonly onBack: () => void;
  readonly onContinue: () => void;
  // Set where this is the last step (self-hosted): Finish replaces Continue.
  readonly onFinish: (() => void) | undefined;
  readonly onFinishWithoutSample: () => void;
  readonly finishing: boolean;
  readonly failure: SetupFinishFailure | undefined;
  // Offered only where setup is optional and nothing has been created yet.
  readonly onSkip: (() => void) | undefined;
}) {
  const { t } = useTranslation();
  const [cleared, setCleared] = useState(false);
  const named = !!projectTitle.trim();
  const ready =
    projectExists ||
    (named && !!projectId && projectIdValid && !projectIdRefusal);
  const showSample = samplePrepared || sample === "offered";
  const projectLabel = named
    ? projectTitle.trim()
    : t("workspace-setup.project.your-project");
  const databaseTitle = t("workspace-setup.project.database-title", {
    project: projectLabel,
  });
  const busy = skipping || finishing;
  const primaryLabel = !onFinish
    ? t("workspace-setup.continue")
    : finishing
      ? t("workspace-setup.team.finishing")
      : failure
        ? t("workspace-setup.team.try-again")
        : t("workspace-setup.team.finish");

  return (
    <SetupScreenFrame
      screen="project"
      title={t("workspace-setup.project.title")}
      description={t("workspace-setup.project.description")}
      onSubmit={() => {
        if (!ready) return;
        if (onFinish) onFinish();
        else onContinue();
      }}
      footer={
        <>
          <Button
            type="button"
            appearance="secondary"
            disabled={busy}
            onClick={onBack}
          >
            {t("common.back")}
          </Button>
          <div className="flex flex-wrap items-center justify-end gap-2">
            {onSkip && (
              <Button
                type="button"
                appearance="secondary"
                disabled={busy}
                onClick={onSkip}
              >
                {t("workspace-setup.project.skip")}
              </Button>
            )}
            {failure?.stage === "sample" && (
              <Button
                type="button"
                appearance="secondary"
                disabled={busy}
                onClick={onFinishWithoutSample}
              >
                {t("workspace-setup.team.finish-without-sample")}
              </Button>
            )}
            <Button type="submit" disabled={!ready || busy}>
              {primaryLabel}
            </Button>
          </div>
        </>
      }
    >
      <FormFieldGroup>
        <FormField>
          <FormLabel htmlFor="workspace-setup-project-title">
            {t("workspace-setup.project.name")}
          </FormLabel>
          <Input
            id="workspace-setup-project-title"
            value={projectTitle}
            autoComplete="off"
            disabled={projectExists || busy}
            placeholder={t("workspace-setup.project.name-placeholder")}
            aria-describedby={
              projectExists
                ? "workspace-setup-project-title-hint"
                : cleared
                  ? "workspace-setup-project-title-note"
                  : undefined
            }
            onChange={(event) => {
              const value = event.target.value;
              // A name the user removed gets its reason; one never typed is
              // asked for by the label alone.
              if (!value.trim() && named) setCleared(true);
              if (value.trim()) setCleared(false);
              onProjectTitleChange(value);
            }}
          />
          {projectExists && (
            <SetupNote id="workspace-setup-project-title-hint">
              {t("workspace-setup.project.exists")}
            </SetupNote>
          )}
          {named ? (
            <ResourceIdField
              suffix
              keepInitialValue
              readonly={projectExists || busy}
              value={projectId}
              autoComplete="off"
              resourceName={t("common.project")}
              resourceTitle={projectTitle}
              validate={validateProjectId}
              onChange={onProjectIdChange}
              onValidationChange={onProjectIdValidChange}
            />
          ) : (
            cleared && (
              <SetupNote id="workspace-setup-project-title-note">
                {t("workspace-setup.project.name-required")}
              </SetupNote>
            )
          )}
          {/* The field shows its own reason when its checks fail. */}
          {named && projectIdRefusal && projectIdValid && (
            <FormError>{projectIdRefusal}</FormError>
          )}
        </FormField>
        <FormField>
          <FormTitle>{databaseTitle}</FormTitle>
          <RadioGroup
            aria-label={databaseTitle}
            value={source}
            disabled={samplePrepared || busy}
            onValueChange={(value) => {
              if (value === "sample" || value === "own") onSourceChange(value);
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
                    : t("workspace-setup.project.sample.description-self-host")}
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
      </FormFieldGroup>
      {skipFailure && (
        <Alert
          variant="error"
          title={
            isSaaS
              ? t("workspace-setup.team.failed.names")
              : t("workspace-setup.team.failed.workspace-name")
          }
          description={skipFailure.detail}
        />
      )}
      {onFinish && (
        <FinishStatus
          finishing={finishing}
          failure={failure}
          projectTitle={projectTitle.trim()}
          source={source}
        />
      )}
    </SetupScreenFrame>
  );
}
