import { useMemo } from "react";
import { useAppStore } from "@/stores/app";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { inlineThreadsEnabled } from "@/utils/featureGates";
import { placementsForSheet } from "../../shared/stores/placementSlice";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";
import {
  countPlacedUnresolvedBySpec,
  countUnresolvedThreads,
  groupThreads,
} from "./threadModel";

const NO_COMMENTS: IssueComment[] = [];

const useIssueThreads = (issueName: string | undefined) => {
  const comments = useAppStore((state) =>
    issueName ? state.getIssueComments(issueName) : NO_COMMENTS
  );
  return useMemo(
    () => (inlineThreadsEnabled() ? groupThreads(comments) : []),
    [comments]
  );
};

// Every unresolved thread on the issue; zero before an issue exists.
export function useUnresolvedThreadTotal(issueName: string | undefined) {
  const threads = useIssueThreads(issueName);
  return useMemo(() => countUnresolvedThreads(threads), [threads]);
}

// Unresolved threads the statement editor shows for each of `specs`, so a
// change tab and its editor walker agree.
export function usePlacedUnresolvedThreadCounts(
  issueName: string | undefined,
  specs: Plan_Spec[]
): ReadonlyMap<string, number> {
  const threads = useIssueThreads(issueName);
  const placements = usePlanDetailStore((s) => s.placements);
  const placementTargets = usePlanDetailStore((s) => s.placementTargets);
  return useMemo(
    () =>
      countPlacedUnresolvedBySpec(threads, specs, (specId, sha) =>
        placementsForSheet({ placements, placementTargets }, specId, sha)
      ),
    [placementTargets, placements, specs, threads]
  );
}
