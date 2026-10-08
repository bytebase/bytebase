import { useEffect, useMemo } from "react";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { sheetNameOfSpec } from "@/utils/v1/issue/plan";
import { usePlanDetailStore } from "../shared/stores/usePlanDetailStore";

const EMPTY_COMMENTS: IssueComment[] = [];

// Recomputes comment placements whenever the issue's comments or any spec's
// sheet changes. Mounted once per plan page so the statement editor and the
// Review Activity timeline read the same result.
export function usePlacementSync(page: {
  issueName: string | undefined;
  projectId: string;
  ready: boolean;
  specs: readonly Plan_Spec[];
}) {
  const { issueName, projectId, ready, specs } = page;
  const comments = useAppStore((state) =>
    issueName ? state.getIssueComments(issueName) : EMPTY_COMMENTS
  );
  const computePlacements = usePlanDetailStore((s) => s.computePlacements);

  // Polling replaces the plan object on every tick; only the spec ids and
  // their sheets matter here, so key the effect on those.
  const sheetKey = useMemo(
    () => specs.map((spec) => `${spec.id}=${sheetNameOfSpec(spec)}`).join("\n"),
    [specs]
  );

  useEffect(() => {
    if (!ready || !issueName) return;
    void computePlacements({
      comments,
      projectName: `${projectNamePrefix}${projectId}`,
      specs,
    });
  }, [comments, computePlacements, issueName, projectId, ready, sheetKey]);
}
