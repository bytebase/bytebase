import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { DatabaseTableView } from "./DatabaseTableView";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useEnvironmentList: () => [],
  usePlanFeature: () => false,
}));

describe("DatabaseTableView", () => {
  let root: Root | undefined;
  let container: HTMLDivElement | undefined;
  let clientWidthSpy: { mockRestore: () => void } | undefined;

  afterEach(() => {
    act(() => {
      root?.unmount();
    });
    root = undefined;
    container = undefined;
    clientWidthSpy?.mockRestore();
    clientWidthSpy = undefined;
    document.body.innerHTML = "";
  });

  test("centers the empty placeholder within the visible scroll container", async () => {
    clientWidthSpy = vi
      .spyOn(HTMLElement.prototype, "clientWidth", "get")
      .mockImplementation(function (this: HTMLElement) {
        return this.dataset.testid === "database-table-scroll-container"
          ? 640
          : 0;
      });
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    await act(async () => {
      root!.render(
        <DatabaseTableView
          databases={[]}
          mode="PROJECT"
          emptyPlaceholder={<button type="button">connect database</button>}
        />
      );
      await Promise.resolve();
    });

    const placeholder = container.querySelector(
      "[data-testid='database-table-empty-placeholder']"
    ) as HTMLDivElement;

    expect(placeholder).toBeTruthy();
    expect(placeholder.className).toContain("sticky");
    expect(placeholder.style.width).toBe("640px");
  });
});
