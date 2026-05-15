import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorTreeNode } from "@/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/utils", () => ({
  instanceV1Name: (inst: { title: string }) => inst.title,
}));

vi.mock("@/react/components/instance/constants", () => ({
  EngineIconPath: {
    MYSQL: "/icon/mysql.svg",
    UNSUPPORTED: "",
  },
}));

let InstanceNode: typeof import("./InstanceNode").InstanceNode;

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
  overrides?: Partial<{ engine: string; title: string; name: string }>
): SQLEditorTreeNode<"instance"> =>
  ({
    key: "instances/prod",
    label: overrides?.title ?? "Prod",
    meta: {
      type: "instance",
      target: {
        name: overrides?.name ?? "instances/prod",
        title: overrides?.title ?? "Prod",
        engine: overrides?.engine ?? "MYSQL",
      },
    },
  }) as unknown as SQLEditorTreeNode<"instance">;

beforeEach(async () => {
  ({ InstanceNode } = await import("./InstanceNode"));
});

describe("InstanceNode", () => {
  test("renders instance title", () => {
    const { container, render, unmount } = renderIntoContainer(
      <InstanceNode node={makeNode({ title: "Production" })} keyword="" />
    );
    render();
    expect(container.textContent).toContain("Production");
    unmount();
  });

  test("renders engine icon when icon path exists", () => {
    const { container, render, unmount } = renderIntoContainer(
      <InstanceNode node={makeNode({ engine: "MYSQL" })} keyword="" />
    );
    render();
    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    expect(img?.getAttribute("src")).toBe("/icon/mysql.svg");
    unmount();
  });

  test("omits icon when engine has no icon path", () => {
    const { container, render, unmount } = renderIntoContainer(
      <InstanceNode node={makeNode({ engine: "UNSUPPORTED" })} keyword="" />
    );
    render();
    expect(container.querySelector("img")).toBeNull();
    unmount();
  });
});
