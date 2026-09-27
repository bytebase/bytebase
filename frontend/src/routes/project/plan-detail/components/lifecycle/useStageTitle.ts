import { useMemo } from "react";
import { useAppStore } from "@/stores/app";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";

// Resolve a stage's display title (its environment title), subscribing to
// environmentList so it updates once environments load. Mirrors DeployStageCard.
export function useStageTitle(stage: Stage): string {
  const environmentList = useAppStore((state) => state.environmentList);
  return useMemo(
    () => useAppStore.getState().getEnvironmentByName(stage.environment).title,
    [environmentList, stage.environment]
  );
}
