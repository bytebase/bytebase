import { fireEvent, render, screen } from "@testing-library/react";
import type { ButtonHTMLAttributes } from "react";
import { beforeEach, expect, test, vi } from "vitest";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import { IssueDetailHeader } from "./IssueDetailHeader";

const mocks = vi.hoisted(() => ({
  openAgentWithPrompt: vi.fn(),
  page: undefined as unknown,
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
vi.mock("@/modules/agent/logic/instanceConnectionHelp", () => ({
  openAgentWithPrompt: mocks.openAgentWithPrompt,
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
      approvalStatus: 0,
      name: "projects/example/issues/42",
      plan: "projects/example/plans/9",
      status: IssueStatus.OPEN,
      title: "Review database migration",
    },
    plan: {
      name: "projects/example/plans/9",
      title: "Add audit table",
    },
    sidebarMode: "NONE",
    setMobileSidebarOpen: vi.fn(),
  };
});

test("opens the Agent with this Issue as the selected resource for review", () => {
  render(<IssueDetailHeader />);

  const reviewButton = screen.getByRole("button", {
    name: "agent.issue-assistance.review",
  });
  expect(reviewButton).toHaveAttribute("data-button-size", "md");
  expect(reviewButton).toHaveAttribute("data-button-appearance", "ghost");
  expect(reviewButton.querySelector(".lucide-bot")).not.toBeNull();
  expect(reviewButton).not.toHaveClass("whitespace-normal", "h-auto");
  expect(reviewButton.parentElement).toHaveClass(
    "w-full",
    "flex-wrap",
    "gap-y-2"
  );
  expect(reviewButton.parentElement?.parentElement).toHaveClass("flex-wrap");
  fireEvent.click(reviewButton);

  expect(mocks.openAgentWithPrompt).toHaveBeenCalledWith(
    expect.stringContaining("Review this Issue"),
    [
      {
        kind: "issue",
        name: "projects/example/issues/42",
        project: "projects/example",
        label: "Review database migration",
      },
    ]
  );
});
