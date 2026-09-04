import { CheckCircle, Circle, CircleHelp, ListChecks, X } from "lucide-react";
import { useLayoutEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { router, useCurrentRoute } from "@/app/router";
import {
  getHowBytebaseWorksGuideContent,
  HowBytebaseWorksSheet,
} from "@/components/HowBytebaseWorksSheet";
import { SQLEditorButton } from "@/components/SQLEditorButton";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { BlockTooltip, Tooltip } from "@/components/ui/tooltip";
import { useIntroStateByKey } from "@/hooks/useAppState";
import { preCreateIssue } from "@/lib/plan/issue";
import { PRODUCT_INTRO_QUERY_KEY } from "@/lib/productIntro";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import {
  GUIDE_PROGRESS_KEYS,
  guideCompletionAcknowledgedKey,
} from "./progress";
import { resolveGuide } from "./resolve";
import { getGuideJourney } from "./scenarios";
import {
  readGuideWorkspaceUsage,
  readSelectedGuideScenarioId,
} from "./selection";
import { GUIDE_STEP_REGISTRY } from "./steps";
import type { GuideStepId, ResolvedGuideStep } from "./types";
import { useGuideContext } from "./useGuideContext";

export function WorkspaceSetupGuide() {
  const { i18n, t } = useTranslation();
  const currentRoute = useCurrentRoute();
  const scenarioId = readSelectedGuideScenarioId();
  const workspaceUsage = readGuideWorkspaceUsage();
  const journey = useMemo(
    () => getGuideJourney(scenarioId, workspaceUsage),
    [scenarioId, workspaceUsage]
  );
  const dismissed = useIntroStateByKey(GUIDE_PROGRESS_KEYS.dismissed);
  const completionAcknowledged = useIntroStateByKey(
    guideCompletionAcknowledgedKey(journey.id)
  );
  const allowMultipleMembers =
    workspaceUsage === "team" && !completionAcknowledged;
  const guideEnabled = useAppStore((state) =>
    state.workspaceSetupGuideEnabled(allowMultipleMembers)
  );
  const productModelAvailable = !!getHowBytebaseWorksGuideContent(
    i18n.resolvedLanguage ?? "en-US"
  );
  const { context, loading } = useGuideContext({
    enabled: guideEnabled,
    dismissed,
    route: currentRoute,
    scenarioId,
    workspaceUsage,
  });
  const hasContextualProductIntro =
    typeof currentRoute.query?.[PRODUCT_INTRO_QUERY_KEY] === "string";
  const [productModelOpen, setProductModelOpen] = useState(false);
  const [selectedStepId, setSelectedStepId] = useState<GuideStepId>();
  const [stepsOverflow, setStepsOverflow] = useState(false);
  const [completionCompact, setCompletionCompact] = useState(false);
  const guideBarRef = useRef<HTMLDivElement>(null);
  const completionTitleRef = useRef<HTMLDivElement>(null);
  const completionWideWidthRef = useRef(0);
  const stepViewportRef = useRef<HTMLDivElement>(null);
  const stepMeasurementRef = useRef<HTMLDivElement>(null);
  const guide = useMemo(
    () =>
      resolveGuide({
        journey,
        registry: GUIDE_STEP_REGISTRY,
        context,
        selectedStepId,
      }),
    [context, journey, selectedStepId]
  );
  const primaryAction = guide.actionStep?.actions.primary;
  const compactStep =
    guide.highlightedStep ?? guide.actionStep ?? guide.steps[0];
  const compactStepIndex = compactStep
    ? guide.steps.findIndex(
        (step) => step.definition.id === compactStep.definition.id
      )
    : -1;

  useLayoutEffect(() => {
    if (guide.complete) {
      setStepsOverflow(false);
      return;
    }
    const viewport = stepViewportRef.current;
    const measurement = stepMeasurementRef.current;
    if (!viewport || !measurement) return;

    const updateOverflow = () => {
      setStepsOverflow(measurement.scrollWidth > viewport.clientWidth);
    };
    const observer = new ResizeObserver(updateOverflow);
    observer.observe(viewport);
    observer.observe(measurement);
    updateOverflow();
    return () => observer.disconnect();
  }, [guide.complete, guide.steps, i18n.resolvedLanguage]);

  useLayoutEffect(() => {
    completionWideWidthRef.current = 0;
    setCompletionCompact(false);
  }, [i18n.resolvedLanguage, journey.id]);

  useLayoutEffect(() => {
    if (!guide.complete) {
      completionWideWidthRef.current = 0;
      setCompletionCompact(false);
      return;
    }
    const guideBar = guideBarRef.current;
    const completionTitle = completionTitleRef.current;
    if (!guideBar || !completionTitle) return;

    const updateOverflow = () => {
      if (guideBar.clientWidth === 0) return;
      if (completionCompact) {
        if (guideBar.clientWidth >= completionWideWidthRef.current) {
          setCompletionCompact(false);
        }
        return;
      }

      const clippedWidth =
        completionTitle.scrollWidth - completionTitle.clientWidth;
      if (clippedWidth > 0) {
        completionWideWidthRef.current = guideBar.clientWidth + clippedWidth;
        setCompletionCompact(true);
      }
    };
    const observer = new ResizeObserver(updateOverflow);
    observer.observe(guideBar);
    observer.observe(completionTitle);
    updateOverflow();
    return () => observer.disconnect();
  }, [completionCompact, guide.complete, i18n.resolvedLanguage, journey.id]);
  const onSelectStep = (step: ResolvedGuideStep) => {
    setSelectedStepId(step.definition.id);
    const action = step.actions.select;
    if (action?.type === "navigate") void router.push(action.target);
    if (action?.type === "create-change") {
      void preCreateIssue(action.project, [action.database]);
    }
  };

  const handleDismiss = () => {
    useAppStore.getState().saveIntroStateByKey({
      key: guide.complete
        ? guideCompletionAcknowledgedKey(journey.id)
        : GUIDE_PROGRESS_KEYS.dismissed,
      newState: true,
    });
  };

  if (
    dismissed ||
    !guideEnabled ||
    loading ||
    (guide.complete && completionAcknowledged)
  ) {
    return null;
  }

  const hasDatabaseTarget =
    !!context.databaseName && !!context.databaseProjectName;
  const completionPrimaryAction = journey.completionActions[0];

  return (
    <>
      <div
        ref={guideBarRef}
        data-testid="workspace-setup-guide"
        className="flex w-full shrink-0 items-center gap-x-2 border-t border-block-border bg-background px-4 py-3 shadow-[0_-2px_10px_rgba(0,0,0,0.04)] 2xl:gap-x-4 2xl:px-5 2xl:py-4"
      >
        <div className="flex min-w-0 flex-1 items-center gap-x-2 overflow-hidden 2xl:gap-x-4">
          <div className="flex shrink-0 items-center gap-x-1">
            <div className="shrink-0 text-sm font-semibold text-main 2xl:text-base">
              {t("workspace-setup-guide.getting-started")}
            </div>
            {productModelAvailable && (
              <Tooltip content={t("workspace-setup-guide.product-model")}>
                <Button
                  type="button"
                  appearance="secondary"
                  size="sm"
                  data-testid="open-product-model"
                  aria-label={t("workspace-setup-guide.product-model")}
                  onClick={() => setProductModelOpen(true)}
                >
                  <CircleHelp className="size-4" />
                </Button>
              </Tooltip>
            )}
          </div>
          {guide.complete ? (
            <div className="min-w-0 flex-1">
              <div
                ref={completionTitleRef}
                data-testid="completion-title"
                className="truncate text-sm font-medium whitespace-nowrap text-main"
              >
                {t(journey.completionTitleKey)}
              </div>
              {!completionCompact && (
                <div className="truncate text-sm text-control-light">
                  {t(journey.completionDescriptionKey)}
                </div>
              )}
            </div>
          ) : (
            <div
              ref={stepViewportRef}
              data-testid="guide-step-viewport"
              className="relative min-w-0 flex-1 overflow-hidden pr-1 2xl:pr-2"
            >
              <div
                ref={stepMeasurementRef}
                data-testid="guide-step-measurement"
                aria-hidden
                className="pointer-events-none invisible absolute left-0 top-0 flex w-max items-center gap-x-2 2xl:gap-x-3"
              >
                {guide.steps.map((step, index) => (
                  <div
                    key={step.definition.id}
                    className="inline-flex items-center gap-x-2 2xl:gap-x-3"
                  >
                    <Button
                      type="button"
                      appearance="secondary"
                      tabIndex={-1}
                      className="inline-flex h-auto items-center justify-start gap-x-1 rounded-sm px-2.5 py-1.5 text-sm font-medium whitespace-nowrap 2xl:gap-x-2 2xl:px-3 2xl:py-2 2xl:text-base"
                    >
                      {step.done ? (
                        <CheckCircle className="size-4 2xl:size-5" />
                      ) : (
                        <Circle className="size-4 2xl:size-5" />
                      )}
                      <span>{t(step.definition.labelKey)}</span>
                    </Button>
                    {index < guide.steps.length - 1 && (
                      <span className="text-sm 2xl:text-base">&rsaquo;</span>
                    )}
                  </div>
                ))}
              </div>

              {stepsOverflow && compactStep ? (
                <div
                  data-testid="compact-step-navigator"
                  className="flex min-w-0 items-center gap-x-2"
                >
                  <span className="shrink-0 text-xs text-control-light 2xl:text-sm">
                    {t("workspace-setup-guide.step-progress", {
                      current: compactStepIndex + 1,
                      total: guide.steps.length,
                    })}
                  </span>
                  <BlockTooltip
                    content={
                      compactStep.blocked
                        ? t("workspace-setup-guide.previous-step-required")
                        : t(compactStep.definition.descriptionKey)
                    }
                  >
                    <Button
                      type="button"
                      appearance="secondary"
                      data-testid="compact-active-step"
                      className="inline-flex h-auto w-full min-w-0 items-center justify-start gap-x-1 rounded-sm bg-accent/10 px-2.5 py-1.5 text-sm font-medium text-accent 2xl:gap-x-2 2xl:px-3 2xl:py-2 2xl:text-base"
                      disabled={compactStep.blocked}
                      onClick={() => onSelectStep(compactStep)}
                    >
                      {compactStep.done ? (
                        <CheckCircle className="size-4 shrink-0 text-success 2xl:size-5" />
                      ) : (
                        <Circle className="size-4 shrink-0 2xl:size-5" />
                      )}
                      <span className="truncate">
                        {t(compactStep.definition.labelKey)}
                      </span>
                    </Button>
                  </BlockTooltip>
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={
                        <Button
                          type="button"
                          appearance="secondary"
                          size="sm"
                          data-testid="open-step-list"
                          aria-label={t("workspace-setup-guide.view-all-steps")}
                          title={t("workspace-setup-guide.view-all-steps")}
                        >
                          <ListChecks className="size-4" />
                        </Button>
                      }
                    />
                    <DropdownMenuContent align="end" className="w-64">
                      {guide.steps.map((step) => {
                        const highlighted =
                          step.definition.id === compactStep.definition.id;
                        return (
                          <DropdownMenuItem
                            key={step.definition.id}
                            className={cn(
                              "gap-x-2",
                              highlighted && "bg-control-bg"
                            )}
                            disabled={step.blocked}
                            onClick={() => onSelectStep(step)}
                          >
                            {step.done ? (
                              <CheckCircle className="size-4 shrink-0 text-success" />
                            ) : (
                              <Circle className="size-4 shrink-0" />
                            )}
                            <span className="truncate">
                              {t(step.definition.labelKey)}
                            </span>
                          </DropdownMenuItem>
                        );
                      })}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              ) : (
                <div
                  data-testid="guide-step-list"
                  className="flex w-max items-center gap-x-2 2xl:gap-x-3"
                >
                  {guide.steps.map((step, index) => {
                    const highlighted =
                      step.definition.id ===
                      guide.highlightedStep?.definition.id;
                    return (
                      <div
                        key={step.definition.id}
                        className="inline-flex items-center gap-x-2 2xl:gap-x-3"
                      >
                        <Tooltip
                          content={
                            step.blocked
                              ? t(
                                  "workspace-setup-guide.previous-step-required"
                                )
                              : t(step.definition.descriptionKey)
                          }
                        >
                          <Button
                            type="button"
                            appearance="secondary"
                            data-testid={`setup-step-${step.definition.analyticsKey}`}
                            className={cn(
                              "inline-flex h-auto items-center justify-start gap-x-1 rounded-sm px-2.5 py-1.5 text-sm font-medium whitespace-nowrap 2xl:gap-x-2 2xl:px-3 2xl:py-2 2xl:text-base",
                              highlighted
                                ? "bg-accent/10 text-accent"
                                : step.done
                                  ? "text-control-light"
                                  : "text-control"
                            )}
                            disabled={step.blocked}
                            onClick={() => onSelectStep(step)}
                          >
                            {step.done ? (
                              <CheckCircle className="size-4 text-success 2xl:size-5" />
                            ) : (
                              <Circle className="size-4 2xl:size-5" />
                            )}
                            <span>{t(step.definition.labelKey)}</span>
                          </Button>
                        </Tooltip>
                        {index < guide.steps.length - 1 && (
                          <span className="text-sm text-control-light 2xl:text-base">
                            &rsaquo;
                          </span>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>

        <div className="ml-auto flex shrink-0 items-center gap-x-2">
          {guide.complete &&
            hasDatabaseTarget &&
            journey.completionActions.includes("create-change") &&
            (!completionCompact ||
              completionPrimaryAction === "create-change") && (
              <Button
                type="button"
                appearance="secondary"
                size={completionCompact ? "sm" : undefined}
                onClick={() => {
                  void preCreateIssue(context.databaseProjectName, [
                    context.databaseName,
                  ]);
                }}
              >
                {t("workspace-setup-guide.actions.change")}
              </Button>
            )}
          {guide.complete &&
            hasDatabaseTarget &&
            journey.completionActions.includes("open-sql-editor") &&
            (!completionCompact ||
              completionPrimaryAction === "open-sql-editor") && (
              <SQLEditorButton
                database={{
                  name: context.databaseName,
                  project: context.databaseProjectName,
                }}
                openInNewTab
                size={completionCompact ? "sm" : undefined}
                label={t("workspace-setup-guide.actions.query")}
              />
            )}
          {!guide.complete && primaryAction?.type === "open-sql-editor" && (
            <SQLEditorButton
              data-testid="active-action"
              database={primaryAction.database}
              openInNewTab
              size="sm"
              className="2xl:h-9 2xl:gap-1.5 2xl:px-3 2xl:text-sm 2xl:leading-5"
              label={t("workspace-setup-guide.actions.query")}
            />
          )}
          <Button
            type="button"
            data-testid="dismiss-guide"
            aria-label={t("workspace-setup-guide.dismiss")}
            appearance="secondary"
            size="sm"
            className="text-control-light hover:text-control 2xl:h-9"
            onClick={handleDismiss}
          >
            <X className="size-4 2xl:size-5" />
          </Button>
        </div>
      </div>
      {productModelAvailable && (
        <HowBytebaseWorksSheet
          open={productModelOpen && !hasContextualProductIntro}
          onOpenChange={setProductModelOpen}
        />
      )}
    </>
  );
}
