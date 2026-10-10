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
          <svg className="size-4" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
              clipRule="evenodd"
            />
          </svg>
        </span>
      );
    case IssueStatus.CANCELED:
      return (
        <span
          data-slot="issue-status-icon"
          className="flex items-center justify-center rounded-full size-5 bg-background border-2 text-control-placeholder border-control-placeholder shrink-0"
        >
          <svg className="size-5" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
              clipRule="evenodd"
            />
          </svg>
        </span>
      );
    default:
      return null;
  }
}
