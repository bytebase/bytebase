import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { emptySearchParams } from "./AdvancedSearch";
import { IssueSearchBar, PresetButtons } from "./IssueTable";

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
});
