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

  test("scrolls the pending value option into view when navigating by keyboard", () => {
    const scrollIntoView = vi.fn();
    const originalScrollIntoView = HTMLElement.prototype.scrollIntoView;
    let scrolledText = "";
    HTMLElement.prototype.scrollIntoView = function (
      options?: ScrollIntoViewOptions | boolean
    ) {
      scrolledText = this.textContent ?? "";
      scrollIntoView(options);
    };

    try {
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
                { value: "DONE", keywords: ["done"] },
              ],
            },
          ]}
          onParamsChange={vi.fn()}
        />
      );

      fireEvent.click(screen.getByRole("textbox"));
      fireEvent.click(screen.getByText("state"));

      expect(scrollIntoView).not.toHaveBeenCalled();

      fireEvent.keyDown(screen.getByRole("textbox"), { key: "ArrowDown" });

      expect(scrollIntoView).toHaveBeenCalledWith({
        block: "nearest",
        inline: "nearest",
      });
      expect(scrolledText).toContain("CLOSED");
    } finally {
      HTMLElement.prototype.scrollIntoView = originalScrollIntoView;
    }
  });

  test("renders selected scope values with fixed height and centered custom content", () => {
    render(
      <AdvancedSearch
        params={{
          query: "",
          scopes: [
            { id: "project", value: "unassigned" },
            { id: "engine", value: "COSMOSDB" },
          ],
        }}
        scopeOptions={[
          {
            id: "project",
            title: "Project",
            options: [
              {
                value: "unassigned",
                keywords: ["unassigned"],
                custom: true,
                render: () => <i>Unassigned</i>,
              },
            ],
          },
          {
            id: "engine",
            title: "Engine",
            options: [
              {
                value: "COSMOSDB",
                keywords: ["cosmosdb"],
                custom: true,
                render: () => (
                  <span>
                    <span aria-hidden>🛸</span>
                    CosmosDB
                  </span>
                ),
              },
            ],
          },
        ]}
        onParamsChange={vi.fn()}
      />
    );

    const projectTag = document.querySelector(
      '[data-search-scope-id="project"]'
    );
    const engineTag = document.querySelector(
      '[data-search-scope-id="engine"]'
    );
    expect(projectTag?.className).toContain("h-6");
    expect(engineTag?.className).toContain("h-6");

    const projectValue = projectTag?.children.item(1);
    const engineValue = engineTag?.children.item(1);
    expect(projectValue?.className).toContain("inline-flex");
    expect(projectValue?.className).toContain("items-center");
    expect(engineValue?.className).toContain("inline-flex");
    expect(engineValue?.className).toContain("items-center");
  });
});
