import { useMemo } from "react";
import {
  type BehaviorMetricName,
  createBehaviorMetric,
} from "@/app/analytics/behavior";
import { behaviorAnalytics } from "@/app/analytics/provider";
import { router, useCurrentRoute } from "@/app/router";
import { useIntroStateByKey } from "@/hooks/useAppState";
import { preCreateIssue } from "@/lib/plan/issue";
import { getGuideAnalyticsProperties } from "@/modules/workspace-setup-guide/analytics";
import {
  GUIDE_PROGRESS_KEYS,
  guideCompletionAcknowledgedKey,
} from "@/modules/workspace-setup-guide/progress";
import { resolveGuide } from "@/modules/workspace-setup-guide/resolve";
import { getGuideJourney } from "@/modules/workspace-setup-guide/scenarios";
import {
  readGuideWorkspaceUsage,
  readSelectedGuideScenarioId,
} from "@/modules/workspace-setup-guide/selection";
import { GUIDE_STEP_DEFINITIONS } from "@/modules/workspace-setup-guide/steps";
import type {
  GuideAction,
  ResolvedGuideStep,
} from "@/modules/workspace-setup-guide/types";
import { useGuideContext } from "@/modules/workspace-setup-guide/useGuideContext";
import { WorkspaceHomeGettingStarted } from "@/modules/workspace-setup-guide/WorkspaceHomeGettingStarted";
import { useAppStore } from "@/stores/app";

export function WorkspaceLandingGettingStarted() {
  const route = useCurrentRoute();
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
  const enabled = useAppStore((state) =>
    state.workspaceSetupGuideEnabled(allowMultipleMembers)
  );
  const { context, contextReady } = useGuideContext({
    enabled,
    dismissed,
    route,
    scenarioId,
    workspaceUsage,
  });
  const guide = useMemo(
    () =>
      resolveGuide({
        journey,
        definitions: GUIDE_STEP_DEFINITIONS,
        context,
      }),
    [context, journey]
  );

  if (
    dismissed ||
    !enabled ||
    !contextReady ||
    (guide.complete && completionAcknowledged)
  ) {
    return null;
  }

  const captureMetric = (
    event: BehaviorMetricName,
    properties: Record<string, unknown> = {}
  ) => {
    behaviorAnalytics.captureMetric(
      createBehaviorMetric(event, {
        properties: {
          ...getGuideAnalyticsProperties({
            guide,
            journey,
            scenarioId,
            workspaceUsage,
          }),
          ...properties,
        },
      })
    );
  };
  const captureStepAction = (step: ResolvedGuideStep, action: GuideAction) => {
    captureMetric("workspace setup guide step action selected", {
      step: step.definition.id,
      action_type: action.type,
    });
  };
  const onSelectStep = (step: ResolvedGuideStep) => {
    const action = step.actions.select;
    if (action) captureStepAction(step, action);
    if (action?.type === "navigate") void router.push(action.target);
    if (action?.type === "create-change") {
      void preCreateIssue(action.project, [action.database]);
    }
  };
  const onDismiss = () => {
    captureMetric("workspace setup guide dismissed");
    useAppStore.getState().saveIntroStateByKey({
      key: guide.complete
        ? guideCompletionAcknowledgedKey(journey.id)
        : GUIDE_PROGRESS_KEYS.dismissed,
      newState: true,
    });
  };

  return (
    <WorkspaceHomeGettingStarted
      guide={guide}
      projectName={context.projectName}
      onSelectStep={onSelectStep}
      onCaptureStepAction={captureStepAction}
      onDismiss={onDismiss}
    />
  );
}
