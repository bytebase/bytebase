import { useMemo } from "react";
import { useAppStore } from "@/stores/app";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { getSheetStatement, isSheetContentComplete } from "@/utils/v1/sheet";
import { placementsForSheet } from "../../shared/stores/placementSlice";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";
import {
  completePreviewLineCount,
  countPlacedUnresolvedBySpec,
  countUnresolvedThreads,
  groupThreads,
  sheetNameOfSha256,
  targetSha256OfSpec,
} from "./threadModel";

const NO_COMMENTS: IssueComment[] = [];
const previewEndLinesByContent = new WeakMap<Uint8Array, number>();

const useIssueThreads = (issueName: string | undefined) => {
  const comments = useAppStore((state) =>
    issueName ? state.getIssueComments(issueName) : NO_COMMENTS
  );
  return useMemo(() => groupThreads(comments), [comments]);
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
  specs: Plan_Spec[],
  projectName: string
): ReadonlyMap<string, number> {
  const threads = useIssueThreads(issueName);
  const sheetsByName = useAppStore((s) => s.sheetsByName);
  const placements = usePlanDetailStore((s) => s.placements);
  const placementTargets = usePlanDetailStore((s) => s.placementTargets);
  const previewEndLines = useMemo(() => {
    const lines = new Map<string, number>();
    for (const spec of specs) {
      const sha = targetSha256OfSpec(spec);
      if (!sha) continue;
      const sheet = sheetsByName[sheetNameOfSha256(projectName, sha)];
      if (sheet && !isSheetContentComplete(sheet)) {
        let endLine = previewEndLinesByContent.get(sheet.content);
        if (endLine === undefined) {
          endLine = completePreviewLineCount(getSheetStatement(sheet), true);
          previewEndLinesByContent.set(sheet.content, endLine);
        }
        lines.set(spec.id, endLine);
      }
    }
    return lines;
  }, [projectName, sheetsByName, specs]);
  return useMemo(
    () =>
      countPlacedUnresolvedBySpec(
        threads,
        specs,
        (specId, sha) =>
          placementsForSheet({ placements, placementTargets }, specId, sha),
        (specId) => previewEndLines.get(specId)
      ),
    [placementTargets, placements, previewEndLines, specs, threads]
  );
}
