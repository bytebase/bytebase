import type {
  IssueComment,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { type FindingGroup, inFindingGroup } from "./planGates";

// A reviewer's findings in one group, in change and line order: where a click
// on a count lands first, and the order the walker steps in. A finding
// without an anchor sorts last.
export function findingsInOrder(
  comments: readonly IssueComment[],
  runType: ReviewRun_Type,
  specs: readonly Plan_Spec[],
  group: FindingGroup
): IssueComment[] {
  const specIndex = (comment: IssueComment) => {
    const index = specs.findIndex(
      (spec) => spec.id === comment.statementAnchor?.spec
    );
    return index === -1 ? Number.MAX_SAFE_INTEGER : index;
  };
  const line = (comment: IssueComment) =>
    comment.statementAnchor?.startPosition?.line || Number.MAX_SAFE_INTEGER;
  return comments
    .filter(
      (comment) =>
        comment.reviewMetadata?.runType === runType &&
        inFindingGroup(comment, group)
    )
    .sort(
      (a, b) =>
        specIndex(a) - specIndex(b) ||
        line(a) - line(b) ||
        a.name.localeCompare(b.name)
    );
}

// How many findings share each name, most frequent first.
export function countByName(
  findings: readonly IssueComment[],
  nameOf: (comment: IssueComment) => string
): [string, number][] {
  const counts = new Map<string, number>();
  for (const comment of findings) {
    const name = nameOf(comment);
    counts.set(name, (counts.get(name) ?? 0) + 1);
  }
  return [...counts].sort(([a, x], [b, y]) => y - x || a.localeCompare(b));
}
