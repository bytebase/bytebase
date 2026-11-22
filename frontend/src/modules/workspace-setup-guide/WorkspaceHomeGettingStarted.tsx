import { CheckCircle, ListChecks, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { SQLEditorButton } from "@/components/SQLEditorButton";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import type { GuideAction, ResolvedGuide, ResolvedGuideStep } from "./types";

export function WorkspaceHomeGettingStarted({
  guide,
  projectName,
  onSelectStep,
  onCaptureStepAction,
  onDismiss,
}: {
  guide: ResolvedGuide;
  projectName: string;
  onSelectStep: (step: ResolvedGuideStep) => void;
  onCaptureStepAction: (step: ResolvedGuideStep, action: GuideAction) => void;
  onDismiss: () => void;
}) {
  const { t } = useTranslation();
  const actionStep = guide.activeStep;
  const action = actionStep?.actions.primary;
  const actionLabel = actionStep
    ? actionStep.definition.id === "connect-instance"
      ? t("landing.onboarding.connect-action")
      : t(actionStep.definition.labelKey)
    : undefined;
  const projectTitle = useAppStore(
    (state) => state.projectsByName?.[projectName]?.title
  );
  const completedStepCount = guide.steps.filter((step) => step.done).length;

  return (
    <section data-testid="workspace-home-getting-started" className="min-w-0">
      <div className="flex flex-wrap items-center justify-between gap-2 pb-4">
        <h2 className="text-base font-semibold text-main">
          {t("workspace-setup-guide.getting-started")}
        </h2>
        <span className="text-xs text-control-light">
          {t("landing.onboarding.completed-progress", {
            completed: completedStepCount,
            total: guide.steps.length,
          })}
        </span>
      </div>

      <div className="flex min-w-0 flex-wrap items-center gap-4 border-y border-block-border py-5">
        <span className="flex size-8 shrink-0 items-center justify-center rounded-full border border-accent text-accent">
          {guide.complete ? (
            <CheckCircle className="size-4" />
          ) : (
            <ListChecks className="size-4" />
          )}
        </span>
        <div className="min-w-0 flex-1">
          <div className="text-sm font-semibold text-main">
            {actionLabel ?? t("workspace-setup-guide.all-steps-completed")}
          </div>
          {actionStep && (
            <p className="mt-1 text-xs text-control-light">
              {actionStep.definition.id === "connect-instance" && projectTitle
                ? t("landing.onboarding.connect-description", {
                    project: projectTitle,
                  })
                : t(actionStep.definition.descriptionKey)}
            </p>
          )}
        </div>
        {actionStep &&
          actionStep.actions.select &&
          (action?.type === "open-sql-editor" ? (
            <span
              onClickCapture={() => onCaptureStepAction(actionStep, action)}
            >
              <SQLEditorButton
                data-testid="workspace-home-next-action"
                database={action.database}
                query={action.query}
                openInNewTab
                size="sm"
                label={actionLabel}
              />
            </span>
          ) : (
            <Button
              type="button"
              size="sm"
              className="shrink-0"
              data-testid="workspace-home-next-action"
              onClick={() => onSelectStep(actionStep)}
            >
              {actionLabel}
            </Button>
          ))}
        {guide.complete && (
          <Tooltip content={t("workspace-setup-guide.dismiss")}>
            <Button
              type="button"
              size="sm"
              appearance="secondary"
              aria-label={t("workspace-setup-guide.dismiss")}
              onClick={onDismiss}
            >
              <X className="size-4" />
            </Button>
          </Tooltip>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 pt-3">
        {guide.steps.map((step, index) => (
          <div
            key={step.definition.id}
            className="inline-flex items-center gap-x-2"
          >
            {index > 0 && <span className="text-xs text-control-light">/</span>}
            <span
              className={cn(
                "inline-flex items-center gap-x-1 text-xs",
                step.done
                  ? "text-success"
                  : step.definition.id === actionStep?.definition.id
                    ? "font-semibold text-main"
                    : "text-control-light"
              )}
            >
              {step.done && <CheckCircle className="size-3" />}
              {t(step.definition.labelKey)}
            </span>
          </div>
        ))}
      </div>
    </section>
  );
}
