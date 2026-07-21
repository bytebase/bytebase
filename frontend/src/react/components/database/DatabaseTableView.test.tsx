import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
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

const makeDatabase = (name: string): Database =>
  ({
    name,
    effectiveEnvironment: "environments/test",
    labels: {},
  }) as Database;

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

  test("selects a database when a selectable row is clicked", async () => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    const database = makeDatabase("instances/i/databases/db1");
    const onSelectedNamesChange = vi.fn();

    await act(async () => {
      root!.render(
        <DatabaseTableView
          databases={[database]}
          mode="PROJECT"
          selectedNames={new Set()}
          onSelectedNamesChange={onSelectedNamesChange}
          selectOnRowClick
        />
      );
      await Promise.resolve();
    });

    const row = container.querySelector("tbody tr") as HTMLTableRowElement;
    act(() => row.click());

    expect(onSelectedNamesChange).toHaveBeenCalledTimes(1);
    expect([...onSelectedNamesChange.mock.calls[0][0]]).toEqual([
      database.name,
    ]);
  });

  test("select all checks database names instead of selection size", async () => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    const databases = [
      makeDatabase("instances/i/databases/db1"),
      makeDatabase("instances/i/databases/db2"),
    ];
    const onSelectedNamesChange = vi.fn();

    await act(async () => {
      root!.render(
        <DatabaseTableView
          databases={databases}
          mode="PROJECT"
          selectedNames={new Set(["old-1", "old-2"])}
          onSelectedNamesChange={onSelectedNamesChange}
        />
      );
      await Promise.resolve();
    });

    const selectAllCell = container.querySelector(
      "thead th"
    ) as HTMLTableCellElement;
    act(() => selectAllCell.click());

    expect([...onSelectedNamesChange.mock.calls[0][0]]).toEqual(
      databases.map((database) => database.name)
    );
  });

  test("can expose the selection column as a product intro target", async () => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    await act(async () => {
      root!.render(
        <DatabaseTableView
          databases={[makeDatabase("instances/i/databases/db1")]}
          mode="PROJECT"
          selectedNames={new Set()}
          onSelectedNamesChange={vi.fn()}
          selectionColumnIntroTarget="prepare-database"
        />
      );
      await Promise.resolve();
    });

    const target = container.querySelector(
      "[data-product-intro-target='prepare-database']"
    );

    expect(target).toBeTruthy();
    expect(target?.getAttribute("data-product-intro-preserve-position")).toBe(
      "true"
    );
    expect(target?.className).toContain("w-12");
    expect(target?.className).toContain("bottom-0");
    expect(target?.className).toContain("pointer-events-none");
    expect(target?.className).not.toContain("h-full");
  });
});
