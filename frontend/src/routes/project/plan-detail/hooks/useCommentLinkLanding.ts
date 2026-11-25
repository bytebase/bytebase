import { useEffect, useMemo, useRef } from "react";
import { PLAN_DETAIL_PHASE_REVIEW } from "@/app/router/handles";
import { useAppStore } from "@/stores/app";
import type { PlanDetailPhase } from "../shared/stores/types";
import { usePlanDetailStoreApi } from "../shared/stores/usePlanDetailStore";
import { commentNameOfHash } from "../utils/commentLink";

// Opens the comment the page's address names: expands Review and asks the
// Activity timeline to reveal it. It runs once for each address, and waits for
// the issue's comments, so a later refresh of them never moves the page again.
// A comment the loaded ones lack may be newer than them, as when a link is
// pasted into a tab that already shows the plan, so they are listed once more.
// An address that still names no comment opens the plan as it would without
// the fragment. Returns the comment's name for the outline.
export function useCommentLinkLanding({
  expandPhase,
  issueName,
  routeHash,
}: {
  expandPhase: (phase: PlanDetailPhase) => void;
  issueName: string | undefined;
  routeHash: string | undefined;
}): string | undefined {
  const storeApi = usePlanDetailStoreApi();
  const named = useMemo(
    () => commentNameOfHash(routeHash, issueName),
    [issueName, routeHash]
  );
  const exists = useAppStore((state) => {
    if (!named || !issueName) return undefined;
    const comments = state.issueCommentsByIssue[issueName];
    return comments?.some((comment) => comment.name === named);
  });
  // The address already acted on, and the one already listed again for.
  const settled = useRef<string | undefined>(undefined);
  const relisted = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (!named || !issueName) {
      settled.current = undefined;
      relisted.current = undefined;
      return;
    }
    if (exists === undefined || settled.current === named) return;
    if (exists) {
      settled.current = named;
      expandPhase(PLAN_DETAIL_PHASE_REVIEW);
      storeApi.getState().requestActivityFocus(named);
      return;
    }
    if (relisted.current === named) return;
    relisted.current = named;
    const giveUpIfAbsent = () => {
      const comments = useAppStore.getState().issueCommentsByIssue[issueName];
      if (!comments?.some((comment) => comment.name === named)) {
        settled.current = named;
      }
    };
    void useAppStore
      .getState()
      .fetchIssueCommentThreads({ parent: issueName })
      .then(giveUpIfAbsent, giveUpIfAbsent);
  }, [exists, expandPhase, issueName, named, storeApi]);
  return exists ? named : undefined;
}
