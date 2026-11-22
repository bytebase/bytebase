import type { TFunction } from "i18next";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { IssueComment_ReviewMetadata_Priority } from "@/types/proto-es/v1/issue_service_pb";
import type { FindingPriority, ReviewerFindings } from "../../utils/planGates";
import { priorityStyle } from "../threads/ReviewResult";

const PRIORITY: Record<FindingPriority, IssueComment_ReviewMetadata_Priority> =
  {
    p0: IssueComment_ReviewMetadata_Priority.P0,
    p1: IssueComment_ReviewMetadata_Priority.P1,
    p2: IssueComment_ReviewMetadata_Priority.P2,
  };

export const FINDING_PRIORITIES: readonly FindingPriority[] = [
  "p0",
  "p1",
  "p2",
];

export const findingPriorityStyle = (t: TFunction, priority: FindingPriority) =>
  priorityStyle(t, PRIORITY[priority]);

// The priority badge each finding carries, then how many are open. The count
// stays outside the badge and takes the color of the text around it.
export function PriorityCount({
  priority,
  count,
  className,
  countClassName,
}: {
  priority: FindingPriority;
  count: number;
  className?: string;
  countClassName?: string;
}) {
  const { t } = useTranslation();
  return (
    <span className={cn("inline-flex items-center gap-x-1", className)}>
      <Badge variant={findingPriorityStyle(t, priority)?.variant}>
        {priority.toUpperCase()}
      </Badge>
      <span className={countClassName}>{count}</span>
    </span>
  );
}

// A reviewer's open findings by priority, P0 first.
export function PriorityCounts({ findings }: { findings: ReviewerFindings }) {
  return (
    <span
      className="inline-flex items-center gap-x-3 align-middle font-normal"
      data-testid="priority-counts"
    >
      {FINDING_PRIORITIES.filter((priority) => findings[priority] > 0).map(
        (priority) => (
          <PriorityCount
            count={findings[priority]}
            key={priority}
            priority={priority}
          />
        )
      )}
    </span>
  );
}
