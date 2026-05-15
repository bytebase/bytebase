import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorTreeNode } from "@/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useSQLEditorTabStore: vi.fn(() => ({ supportBatchMode: true })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/store", () => ({}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/react/components/instance/constants", () => ({
  EngineIconPath: { MYSQL: "/icon/mysql.svg" },
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: (name: string) => ({
    databaseName: name.split("/").pop() ?? name,
  }),
  getInstanceResource: () => ({
    name: "instances/prod",
    title: "Prod",
    engine: "MYSQL",
  }),
  instanceV1Name: (inst: { title: string }) => inst.title,
  isDatabaseV1Queryable: () => true,
}));

vi.mock("@/react/components/sql-editor/RequestQueryButton", () => ({
  RequestQueryButton: () => (
    <div data-testid="request-query-button">request</div>
  ),
}));

vi.mock("@bufbuild/protobuf", () => ({
  create: vi.fn((_schema, data) => data),
}));

vi.mock("@/types/proto-es/v1/common_pb", () => ({
  PermissionDeniedDetailSchema: { typeName: "PermissionDeniedDetailSchema" },
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

let DatabaseNode: typeof import("./DatabaseNode").DatabaseNode;

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

const makeNode = (
  overrides?: Partial<{ name: string }>
): SQLEditorTreeNode<"database"> =>
  ({
    key: "databases/bb",
    meta: {
      type: "database",
      target: {
        name: overrides?.name ?? "instances/prod/databases/bb",
        effectiveEnvironment: "environments/dev",
        labels: {},
      },
    },
  }) as unknown as SQLEditorTreeNode<"database">;

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useSQLEditorTabStore.mockReturnValue({ supportBatchMode: true });
  mocks.useVueState.mockImplementation((getter) => getter());
  ({ DatabaseNode } = await import("./DatabaseNode"));
});

describe("DatabaseNode", () => {
  test("renders instance + database name breadcrumb", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseNode node={makeNode()} keyword="" />
    );
    render();
    expect(container.textContent).toContain("Prod");
    expect(container.textContent).toContain("bb");
    unmount();
  });

  test("breadcrumb clicks bubble to the parent (no internal stopPropagation)", () => {
    const onParentClick = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <div onClick={onParentClick}>
        <DatabaseNode node={makeNode()} keyword="" />
      </div>
    );
    render();
    const bread = container.querySelector(".tree-node-database") as HTMLElement;
    act(() => {
      bread.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(onParentClick).toHaveBeenCalledTimes(1);
    unmount();
  });

  test("renders checkbox only when tabStore.supportBatchMode is true", () => {
    mocks.useVueState.mockImplementation(() => false);
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseNode node={makeNode()} keyword="" />
    );
    render();
    expect(container.querySelector("input[type='checkbox']")).toBeNull();
    unmount();
  });

  test("checkbox click propagation is stopped", () => {
    mocks.useVueState.mockImplementation(() => true);
    const onClick = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <div onClick={onClick}>
        <DatabaseNode node={makeNode()} keyword="" />
      </div>
    );
    render();
    const cb = container.querySelector(
      "input[type='checkbox']"
    ) as HTMLInputElement;
    act(() => {
      cb.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(onClick).not.toHaveBeenCalled();
    unmount();
  });
});
