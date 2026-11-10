import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import { IssueSummary } from "./IssueSummary";

vi.mock("react-i18next", () => ({
  initReactI18next: {
    type: "3rdParty",
    init: vi.fn(),
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

const issue = create(IssueSchema, {
  name: "projects/example/issues/101",
  title: "Example access request",
  creator: "users/creator@example.com",
  status: IssueStatus.DONE,
  createTime: create(TimestampSchema, { seconds: 1720915200n }),
});

describe("IssueSummary", () => {
  test("shows the issue status, title, ID, creation date, and creator", () => {
    render(<IssueSummary issue={issue} />);

    expect(screen.getByText("Example access request")).toBeInTheDocument();
    expect(screen.getByText("common.done")).toHaveClass("sr-only");
    expect(screen.getByText("#101")).toBeInTheDocument();
    expect(screen.getByText("common.created")).toBeInTheDocument();
    expect(screen.getByText(/2024/)).toBeInTheDocument();
    expect(screen.getByText("creator")).toBeInTheDocument();
    expect(
      screen.getByText("Example access request").closest("[data-slot='issue-summary']")
        ?.querySelector("[data-slot='issue-status-icon']")
    ).toBeTruthy();
  });

  test("has no nested controls when rendered in a picker button", () => {
    render(
      <button type="button">
        <IssueSummary issue={issue} />
      </button>
    );

    expect(
      screen.getByText("Example access request").closest("[data-slot='issue-summary']")
        ?.querySelector("a, button, input, [role='button']")
    ).toBeNull();
  });
});
