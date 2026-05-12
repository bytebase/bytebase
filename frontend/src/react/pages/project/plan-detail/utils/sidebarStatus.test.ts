import { describe, expect, test } from "vitest";
import { ApprovalStatus, State } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { getPlanDetailSidebarStatusInfo } from "./sidebarStatus";

const labels: Record<string, string> = {
  "common.approved": "Approved",
  "common.canceled": "Canceled",
  "common.deploying": "Deploying",
  "common.done": "Done",
  "common.draft": "Draft",
  "common.failed": "Failed",
  "common.in-review": "In Review",
  "common.not-started": "Not started",
  "common.rejected": "Rejected",
};

const t = (key: string) => labels[key] ?? key;

const status = ({
  issue,
  rollout,
}: {
  issue?: {
    approvalStatus: ApprovalStatus;
    status: IssueStatus;
  };
  rollout?: Rollout;
}) =>
  getPlanDetailSidebarStatusInfo({
    isCreating: false,
    issue,
    planState: State.ACTIVE,
    rollout,
    t,
  });

const rolloutWith = (...statuses: Task_Status[]): Rollout =>
  ({
    stages: [
      {
        tasks: statuses.map((taskStatus) => ({ status: taskStatus })),
      },
    ],
  }) as unknown as Rollout;

describe("plan detail sidebar status", () => {
  test("maps review statuses like the Vue sidebar", () => {
    expect(status({}).label).toBe("Draft");
    expect(
      status({
        issue: {
          approvalStatus: ApprovalStatus.REJECTED,
          status: IssueStatus.OPEN,
        },
      })
    ).toEqual({ dotClass: "bg-warning", label: "Rejected" });
    expect(
      status({
        issue: {
          approvalStatus: ApprovalStatus.SKIPPED,
          status: IssueStatus.OPEN,
        },
      })
    ).toEqual({ dotClass: "bg-success", label: "Approved" });
  });

  test("maps rollout statuses like the Vue sidebar", () => {
    expect(status({ rollout: rolloutWith(Task_Status.RUNNING) })).toEqual({
      dotClass: "bg-accent",
      label: "Deploying",
    });
    expect(status({ rollout: rolloutWith(Task_Status.DONE) })).toEqual({
      dotClass: "bg-success",
      label: "Done",
    });
    expect(status({ rollout: rolloutWith(Task_Status.FAILED) })).toEqual({
      dotClass: "bg-error",
      label: "Failed",
    });
    expect(status({ rollout: rolloutWith(Task_Status.CANCELED) })).toEqual({
      dotClass: "bg-control-placeholder",
      label: "Canceled",
    });
    expect(status({ rollout: rolloutWith(Task_Status.NOT_STARTED) })).toEqual({
      dotClass: "bg-control-placeholder",
      label: "Not started",
    });
  });
});
