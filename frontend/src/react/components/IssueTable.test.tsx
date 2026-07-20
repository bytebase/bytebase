import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  IssueSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { emptySearchParams } from "./AdvancedSearch";
import { IssueListItem, IssueSearchBar, PresetButtons } from "./IssueTable";

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => ({ email: "reviewer@example.com" }),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: {
    type: "3rdParty",
    init: vi.fn(),
  },
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/react/components/TimeRangePicker", () => ({
  TimeRangePicker: () => <div data-testid="time-range-picker" />,
}));

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}

globalThis.ResizeObserver =
  ResizeObserverMock as unknown as typeof ResizeObserver;

describe("IssueSearchBar", () => {
  test("renders the default advanced search placeholder", () => {
    render(
      <IssueSearchBar
        params={emptySearchParams()}
        onParamsChange={vi.fn()}
        orderBy=""
        onOrderByChange={vi.fn()}
        scopeOptions={[]}
      />
    );

    expect(screen.getByPlaceholderText("common.filter")).toBeInTheDocument();
    expect(screen.getByTestId("time-range-picker").parentElement).toHaveClass(
      "flex-wrap",
      "gap-2"
    );
  });
});

describe("PresetButtons", () => {
  test("uses accessible tabs and updates the selected preset", () => {
    const onParamsChange = vi.fn();
    render(
      <PresetButtons
        params={{ query: "", scopes: [{ id: "status", value: "OPEN" }] }}
        onParamsChange={onParamsChange}
      />
    );

    expect(screen.getByRole("tablist")).toBeInTheDocument();

    const open = screen.getByRole("tab", { name: "issue.table.open" });
    expect(open).toHaveAttribute("aria-selected", "true");

    fireEvent.click(screen.getByRole("tab", { name: "issue.table.closed" }));
    expect(onParamsChange).toHaveBeenCalledWith({
      query: "",
      scopes: [
        { id: "status", value: "DONE", readonly: undefined },
        { id: "status", value: "CANCELED", readonly: undefined },
      ],
    });
  });

  test("keeps every tab unselected when the filter matches no preset", () => {
    const onParamsChange = vi.fn();
    render(
      <PresetButtons
        params={{
          query: "",
          scopes: [
            { id: "status", value: "OPEN" },
            { id: "approval", value: "APPROVED" },
          ],
        }}
        onParamsChange={onParamsChange}
      />
    );

    for (const tab of screen.getAllByRole("tab")) {
      expect(tab).toHaveAttribute("aria-selected", "false");
    }

    // A tab click from the no-match state must still apply its preset.
    fireEvent.click(
      screen.getByRole("tab", { name: "issue.waiting-approval" })
    );
    expect(onParamsChange).toHaveBeenCalled();
  });
});

describe("IssueListItem", () => {
  const issue = create(IssueSchema, {
    name: "projects/foo/issues/1",
    title: "Issue 1",
    creator: "users/creator@example.com",
    status: IssueStatus.OPEN,
  });

  test("marks normal row and title-link navigation for restoration", () => {
    const onOpenIssue = vi.fn();
    const { rerender } = render(
      <IssueListItem
        issue={issue}
        selected={false}
        onToggleSelection={vi.fn()}
        onOpenIssue={onOpenIssue}
      />
    );

    fireEvent.click(screen.getByTestId("issue-list-item"));
    expect(onOpenIssue).toHaveBeenCalledOnce();

    onOpenIssue.mockClear();
    rerender(
      <IssueListItem
        issue={issue}
        selected={false}
        onToggleSelection={vi.fn()}
        onOpenIssue={onOpenIssue}
      />
    );
    fireEvent.click(screen.getByRole("link", { name: "Issue 1" }));
    expect(onOpenIssue).toHaveBeenCalledOnce();
  });

  test("does not mark modified title-link navigation", () => {
    const onOpenIssue = vi.fn();
    render(
      <IssueListItem
        issue={issue}
        selected={false}
        onToggleSelection={vi.fn()}
        onOpenIssue={onOpenIssue}
      />
    );

    fireEvent.click(screen.getByRole("link", { name: "Issue 1" }), {
      ctrlKey: true,
    });
    expect(onOpenIssue).not.toHaveBeenCalled();
  });
});
