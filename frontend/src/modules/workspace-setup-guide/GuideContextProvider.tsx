import { createContext, type ReactNode, useContext, useMemo } from "react";
import { useCurrentRoute } from "@/app/router";
import { useIntroStateByKey } from "@/hooks/useAppState";
import { useAppStore } from "@/stores/app";
import {
  GUIDE_PROGRESS_KEYS,
  guideCompletionAcknowledgedKey,
} from "./progress";
import { getGuideJourney } from "./scenarios";
import {
  readGuideWorkspaceUsage,
  readSelectedGuideScenarioId,
} from "./selection";
import { useGuideContext } from "./useGuideContext";

type SharedGuideContext = ReturnType<typeof useGuideContext>;

const GuideContext = createContext<SharedGuideContext | null>(null);

export function GuideContextProvider({ children }: { children: ReactNode }) {
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
  const value = useGuideContext({
    enabled,
    dismissed,
    route,
    scenarioId,
    workspaceUsage,
  });

  return (
    <GuideContext.Provider value={value}>{children}</GuideContext.Provider>
  );
}

export function useSharedGuideContext(): SharedGuideContext {
  const value = useContext(GuideContext);
  if (!value) {
    throw new Error("GuideContextProvider is required");
  }
  return value;
}
