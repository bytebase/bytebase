import { CheckCircle, Circle, CircleHelp, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { createBehaviorMetric } from "@/app/analytics/behavior";
import { behaviorAnalytics } from "@/app/analytics/provider";
import { router, useCurrentRoute } from "@/app/router";
import {
  getHowBytebaseWorksGuideContent,
  HowBytebaseWorksSheet,
} from "@/components/HowBytebaseWorksSheet";
import { SQLEditorButton } from "@/components/SQLEditorButton";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { useIntroStateByKey } from "@/hooks/useAppState";
import { preCreateIssue } from "@/lib/plan/issue";
import { PRODUCT_INTRO_QUERY_KEY } from "@/lib/productIntro";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { resolveGuide } from "./resolve";
import { LEARN_BYTEBASE_BASICS_SCENARIO } from "./scenarios";
import { GUIDE_STEP_REGISTRY } from "./steps";
import type { GuideStepId, ResolvedGuideStep } from "./types";
import { useGuideContext } from "./useGuideContext";

const DISMISSED_KEY = "workspace-setup-guide.dismissed";
const PRODUCT_MODEL_SEEN_KEY = "workspace-setup-guide.product-model-seen";

export function WorkspaceSetupGuide() {
  const { i18n, t } = useTranslation();
  const currentRoute = useCurrentRoute();
  const dismissed = useIntroStateByKey(DISMISSED_KEY);
  const productModelSeen = useIntroStateByKey(PRODUCT_MODEL_SEEN_KEY);
  const guideEnabled = useAppStore((state) =>
    state.workspaceSetupGuideEnabled()
  );
  const workspaceResourceName = useAppStore((state) =>
    state.workspaceResourceName()
  );
  const { context, loading } = useGuideContext({
    enabled: guideEnabled,
    dismissed,
    route: currentRoute,
  });
  const productModelAvailable = !!getHowBytebaseWorksGuideContent(
    i18n.resolvedLanguage ?? "en-US"
  );
  const hasContextualProductIntro =
    typeof currentRoute.query?.[PRODUCT_INTRO_QUERY_KEY] === "string";
  const [productModelOpen, setProductModelOpen] = useState(false);
  const [selectedStepId, setSelectedStepId] = useState<GuideStepId>();
  const productModelAutoOpenedScopeRef = useRef<string | undefined>(undefined);
  const guide = useMemo(
    () =>
      resolveGuide({
        scenario: LEARN_BYTEBASE_BASICS_SCENARIO,
        registry: GUIDE_STEP_REGISTRY,
        context,
        selectedStepId,
      }),
    [context, selectedStepId]
  );
  const secondaryAction = guide.actionStep.actions.secondary;
  const primaryAction = guide.actionStep.actions.primary;

  useEffect(() => {
    setSelectedStepId(undefined);
  }, [currentRoute.name]);

  useEffect(() => {
    if (productModelSeen) {
      productModelAutoOpenedScopeRef.current = undefined;
      return;
    }
    if (
      loading ||
      dismissed ||
      !guideEnabled ||
      !productModelAvailable ||
      hasContextualProductIntro ||
      !workspaceResourceName ||
      productModelAutoOpenedScopeRef.current === workspaceResourceName
    ) {
      return;
    }
    productModelAutoOpenedScopeRef.current = workspaceResourceName;
    setProductModelOpen(true);
  }, [
    dismissed,
    guideEnabled,
    hasContextualProductIntro,
    loading,
    productModelAvailable,
    productModelSeen,
    workspaceResourceName,
  ]);

  const onSelectStep = (step: ResolvedGuideStep) => {
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("setup guide action clicked", {
        properties: { step: step.definition.analyticsKey },
      })
    );
    setSelectedStepId(step.definition.id);
    if (step.actions.select?.type === "navigate") {
      void router.push(step.actions.select.target);
    }
  };

  const handleDismiss = () => {
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("setup guide dismissed", {
        properties: {
          step: guide.actionStep.definition.analyticsKey,
        },
      })
    );
    useAppStore.getState().saveIntroStateByKey({
      key: DISMISSED_KEY,
      newState: true,
    });
  };

  const handleProductModelOpenChange = (open: boolean) => {
    if (open) {
      behaviorAnalytics.captureMetric(
        createBehaviorMetric("setup guide action clicked", {
          properties: {
            action: "product_model_open",
            source: "guide_bar",
          },
        })
      );
    }
    setProductModelOpen(open);
    if (!open && !productModelSeen) {
      useAppStore.getState().saveIntroStateByKey({
        key: PRODUCT_MODEL_SEEN_KEY,
        newState: true,
      });
    }
  };

  if (dismissed || !guideEnabled || loading) {
    return null;
  }

  return (
    <>
      <div className="flex w-full shrink-0 items-center gap-x-2 border-t border-block-border bg-white px-3 py-2 shadow-[0_-2px_10px_rgba(0,0,0,0.04)] 2xl:gap-x-4 2xl:px-5 2xl:py-4">
        <div className="flex min-w-0 flex-1 items-center gap-x-2 overflow-hidden 2xl:gap-x-4">
          <div className="flex shrink-0 items-center gap-x-1">
            <div className="shrink-0 text-sm font-semibold text-main 2xl:text-base">
              {t("workspace-setup-guide.self")}
            </div>
            {productModelAvailable && (
              <Tooltip content={t("workspace-setup-guide.product-model")}>
                <Button
                  type="button"
                  appearance="secondary"
                  size="sm"
                  data-testid="open-product-model"
                  aria-label={t("workspace-setup-guide.product-model")}
                  onClick={() => handleProductModelOpenChange(true)}
                >
                  <CircleHelp className="size-4" />
                </Button>
              </Tooltip>
            )}
          </div>
          <div className="flex min-w-0 flex-1 items-center gap-x-2 overflow-x-auto pr-1 2xl:gap-x-3 2xl:pr-2">
            {guide.steps.map((step, index) => {
              const highlighted =
                step.definition.id === guide.highlightedStep?.definition.id;
              const tooltipContent = step.blocked
                ? t("workspace-setup-guide.previous-step-required")
                : t(step.definition.descriptionKey);
              return (
                <div
                  key={step.definition.id}
                  className="inline-flex items-center gap-x-2 2xl:gap-x-3"
                >
                  <Tooltip content={tooltipContent}>
                    <Button
                      type="button"
                      appearance="secondary"
                      data-testid={`setup-step-${step.definition.analyticsKey}`}
                      className={cn(
                        "inline-flex h-auto items-center justify-start gap-x-1 rounded-sm px-2 py-1 text-sm font-medium whitespace-nowrap 2xl:gap-x-2 2xl:px-3 2xl:py-2 2xl:text-base",
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
        </div>
        <div className="ml-auto flex shrink-0 items-center gap-x-2">
          {secondaryAction?.type === "create-change" && (
            <Button
              type="button"
              data-testid="secondary-action"
              appearance="secondary"
              size="md"
              className="hidden 2xl:inline-flex"
              onClick={() => {
                behaviorAnalytics.captureMetric(
                  createBehaviorMetric("setup guide action clicked", {
                    properties: { step: "createFirstChange" },
                  })
                );
                void preCreateIssue(secondaryAction.project, [
                  secondaryAction.database,
                ]);
              }}
            >
              {t("workspace-setup-guide.actions.change")}
            </Button>
          )}
          {primaryAction?.type === "open-sql-editor" && (
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
            className="text-control-light hover:text-control 2xl:h-9 2xl:gap-1.5 2xl:px-3 2xl:text-sm 2xl:leading-5"
            onClick={handleDismiss}
          >
            <X className="size-4 2xl:size-5" />
          </Button>
        </div>
      </div>
      {productModelAvailable && (
        <HowBytebaseWorksSheet
          open={productModelOpen && !hasContextualProductIntro}
          onOpenChange={handleProductModelOpenChange}
        />
      )}
    </>
  );
}
