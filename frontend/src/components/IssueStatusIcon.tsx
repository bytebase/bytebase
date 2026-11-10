import { Check, Minus } from "lucide-react";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";

export function IssueStatusIcon({ status }: { status: IssueStatus }) {
  switch (status) {
    case IssueStatus.OPEN:
      return (
        <span
          data-slot="issue-status-icon"
          className="flex items-center justify-center rounded-full size-5 bg-background border-2 border-info text-info shrink-0"
        >
          <span className="h-1.5 w-1.5 bg-info rounded-full" />
        </span>
      );
    case IssueStatus.DONE:
      return (
        <span
          data-slot="issue-status-icon"
          className="flex items-center justify-center rounded-full size-5 bg-success text-accent-text shrink-0"
        >
          <Check className="size-4" aria-hidden="true" />
        </span>
      );
    case IssueStatus.CANCELED:
      return (
        <span
          data-slot="issue-status-icon"
          className="flex items-center justify-center rounded-full size-5 bg-background border-2 text-control-placeholder border-control-placeholder shrink-0"
        >
          <Minus className="size-4" aria-hidden="true" />
        </span>
      );
    default:
      return null;
  }
}
