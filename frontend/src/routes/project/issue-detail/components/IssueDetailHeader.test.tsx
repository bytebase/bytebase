import { create } from "@bufbuild/protobuf";
import { render, screen } from "@testing-library/react";
import type { ButtonHTMLAttributes } from "react";
import { beforeEach, expect, test, vi } from "vitest";
import { ApprovalStatus, IssueStatus, State } from "@/types/proto-es/v1/common_pb";
import { RolloutSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { IssueDetailPageState } from "../hooks/useIssueDetailPage";
import { IssueDetailHeader } from "./IssueDetailHeader";

const mocks = vi.hoisted(() => ({
  page: undefined as unknown as IssueDetailPageState,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/ui/button", () => ({
  Button: ({ children, size, appearance, variant, ...props }: ButtonHTMLAttributes<HTMLButtonElement> & {
    size?: string;
    appearance?: string;
    variant?: string;
  }) => (
    <button
      {...props}
      data-button-size={size}
      data-button-appearance={appearance}
      data-button-variant={variant}
    >
      {children}
    </button>
  ),
}));
vi.mock("../context/IssueDetailContext", () => ({
  useIssueDetailContext: () => mocks.page,
}));
vi.mock("./IssueDetailActionBar", () => ({ IssueDetailActionBar: () => null }));
vi.mock("./IssueDetailTitleInput", () => ({ IssueDetailTitleInput: () => null }));

beforeEach(() => {
  vi.clearAllMocks();
  mocks.page = {
    issue: {
      approvalStatus: ApprovalStatus.PENDING,
      draft: false,
      name: "projects/example/issues/42",
      plan: "projects/example/plans/9",
      status: IssueStatus.OPEN,
      title: "Review database migration",
    },
    plan: {
      hasRollout: false,
      name: "projects/example/plans/9",
      state: State.ACTIVE,
      title: "Add audit table",
    },
    sidebarMode: "NONE",
    setMobileSidebarOpen: vi.fn(),
  } as unknown as IssueDetailPageState;
});

test("does not offer generic Agent review for a pending Issue", () => {
  render(<IssueDetailHeader />);

  expect(
    screen.queryByRole("button", { name: "agent.issue-assistance.review" })
  ).toBeNull();
});

test.each([
  { draft: true },
  { approvalStatus: ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED },
  { approvalStatus: ApprovalStatus.CHECKING },
  { approvalStatus: ApprovalStatus.APPROVED },
  { approvalStatus: ApprovalStatus.REJECTED },
  { approvalStatus: ApprovalStatus.SKIPPED },
  { status: IssueStatus.CANCELED },
  { status: IssueStatus.DONE },
])("does not offer Agent review outside pending review: %j", (state) => {
  mocks.page.issue = { ...mocks.page.issue!, ...state };
  render(<IssueDetailHeader />);

  expect(
    screen.queryByRole("button", { name: "agent.issue-assistance.review" })
  ).toBeNull();
});

test("does not reintroduce generic Agent review when approval completes", () => {
  const { rerender } = render(<IssueDetailHeader />);
  expect(
    screen.queryByRole("button", { name: "agent.issue-assistance.review" })
  ).toBeNull();

  mocks.page.issue = {
    ...mocks.page.issue!,
    approvalStatus: ApprovalStatus.APPROVED,
  };
  rerender(<IssueDetailHeader />);
  expect(
    screen.queryByRole("button", { name: "agent.issue-assistance.review" })
  ).toBeNull();
});

test("does not offer generic Agent review without a linked Plan", () => {
  mocks.page.issue = { ...mocks.page.issue!, plan: "" };
  mocks.page.plan = { ...mocks.page.plan!, state: State.STATE_UNSPECIFIED };
  render(<IssueDetailHeader />);

  expect(
    screen.queryByRole("button", { name: "agent.issue-assistance.review" })
  ).toBeNull();
});

test.each([{ state: State.DELETED }, { hasRollout: true }])(
  "does not offer Agent review for a non-reviewable linked Plan: %j",
  (state) => {
    mocks.page.plan = { ...mocks.page.plan!, ...state };
    render(<IssueDetailHeader />);

    expect(
      screen.queryByRole("button", { name: "agent.issue-assistance.review" })
    ).toBeNull();
  }
);

test("does not offer Agent review when a rollout is already loaded", () => {
  mocks.page.rollout = create(RolloutSchema);
  render(<IssueDetailHeader />);

  expect(
    screen.queryByRole("button", { name: "agent.issue-assistance.review" })
  ).toBeNull();
});
