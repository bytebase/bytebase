import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/router", () => ({
  router: { resolve: vi.fn(() => ({ fullPath: "/x" })) },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL: "project.dbgroup.detail",
}));

vi.mock("@/store", () => ({
  getProjectNameAndDatabaseGroupName: (name: string) => name.split("/"),
}));

let DatabaseGroupDataTable: typeof import("./DatabaseGroupDataTable").DatabaseGroupDataTable;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

const makeGroup = (overrides: Partial<DatabaseGroup>): DatabaseGroup =>
  ({
    name: overrides.name ?? "projects/p/databaseGroups/g",
    title: overrides.title ?? "G",
    databaseExpr: overrides.databaseExpr,
  }) as DatabaseGroup;

beforeEach(async () => {
  ({ DatabaseGroupDataTable } = await import("./DatabaseGroupDataTable"));
});

describe("DatabaseGroupDataTable", () => {
  test("renders a row per group with title + expression", () => {
    const groups = [
      makeGroup({
        name: "projects/p/databaseGroups/a",
        title: "A",
        databaseExpr: {
          expression: "env == 'prod'",
        } as DatabaseGroup["databaseExpr"],
      }),
      makeGroup({
        name: "projects/p/databaseGroups/b",
        title: "B",
        databaseExpr: { expression: "" } as DatabaseGroup["databaseExpr"],
      }),
    ];
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseGroupDataTable databaseGroupList={groups} />
    );
    render();
    expect(container.textContent).toContain("A");
    expect(container.textContent).toContain("env == 'prod'");
    expect(container.textContent).toContain("B");
    // Empty expression → common.empty placeholder
    expect(container.textContent).toContain("common.empty");
    unmount();
  });

  test("shows loading state when loading + empty list", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseGroupDataTable databaseGroupList={[]} loading />
    );
    render();
    expect(container.textContent).toContain("common.loading");
    unmount();
  });

  test("shows no-data when list is empty and not loading", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseGroupDataTable databaseGroupList={[]} />
    );
    render();
    expect(container.textContent).toContain("common.no-data");
    unmount();
  });

  test("checkbox toggles selection (multi-select mode)", () => {
    const groups = [
      makeGroup({ name: "projects/p/databaseGroups/a", title: "A" }),
    ];
    const onChange = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseGroupDataTable
        databaseGroupList={groups}
        showSelection
        onSelectedDatabaseGroupNamesChange={onChange}
      />
    );
    render();
    const cb = container.querySelector(
      "input[type='checkbox']"
    ) as HTMLInputElement;
    act(() => {
      cb.click();
    });
    expect(onChange).toHaveBeenCalledWith(["projects/p/databaseGroups/a"]);
    unmount();
  });

  test("row click fires onRowClick with group", () => {
    const group = makeGroup({
      name: "projects/p/databaseGroups/a",
      title: "A",
    });
    const onRowClick = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseGroupDataTable
        databaseGroupList={[group]}
        onRowClick={onRowClick}
      />
    );
    render();
    const row = container.querySelector("tbody tr") as HTMLElement;
    act(() => {
      row.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(onRowClick).toHaveBeenCalledTimes(1);
    expect(onRowClick.mock.calls[0][1]).toBe(group);
    unmount();
  });

  test("external-link column opens a new tab (does not trigger row click)", () => {
    const onRowClick = vi.fn();
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseGroupDataTable
        databaseGroupList={[
          makeGroup({ name: "projects/p/databaseGroups/a", title: "A" }),
        ]}
        showExternalLink
        onRowClick={onRowClick}
      />
    );
    render();
    const linkBtn = container.querySelector(
      "button[aria-label='common.view-details']"
    ) as HTMLButtonElement;
    act(() => {
      linkBtn.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(openSpy).toHaveBeenCalledWith("/x", "_blank");
    expect(onRowClick).not.toHaveBeenCalled();
    openSpy.mockRestore();
    unmount();
  });
});
