import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { emptySearchParams } from "./AdvancedSearch";
import { IssueSearchBar } from "./IssueTable";

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
  });
});
