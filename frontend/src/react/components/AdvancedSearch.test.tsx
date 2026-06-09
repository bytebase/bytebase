import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { AdvancedSearch, emptySearchParams } from "./AdvancedSearch";

vi.mock("react-i18next", () => ({
  initReactI18next: {
    type: "3rdParty",
    init: vi.fn(),
  },
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}

globalThis.ResizeObserver =
  ResizeObserverMock as unknown as typeof ResizeObserver;

describe("AdvancedSearch", () => {
  test("renders a default placeholder when none is provided", () => {
    render(
      <AdvancedSearch
        params={emptySearchParams()}
        scopeOptions={[]}
        onParamsChange={vi.fn()}
      />
    );

    expect(screen.getByPlaceholderText("common.filter")).toBeInTheDocument();
  });

  test("renders an empty placeholder when scope values have no matches", () => {
    render(
      <AdvancedSearch
        params={emptySearchParams()}
        scopeOptions={[
          {
            id: "state",
            title: "State",
            options: [
              { value: "OPEN", keywords: ["open"] },
              { value: "CLOSED", keywords: ["closed"] },
            ],
          },
        ]}
        onParamsChange={vi.fn()}
      />
    );

    fireEvent.click(screen.getByRole("textbox"));
    fireEvent.click(screen.getByText("state"));
    fireEvent.change(screen.getByRole("textbox"), {
      target: { value: "state:missing" },
    });

    expect(screen.getByText("common.search-no-result")).toBeInTheDocument();
  });
});
