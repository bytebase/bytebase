import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorTreeNode } from "@/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("./InstanceNode", () => ({
  InstanceNode: () => <div data-testid="InstanceNode" />,
}));
vi.mock("./DatabaseNode", () => ({
  DatabaseNode: () => <div data-testid="DatabaseNode" />,
}));
vi.mock("./LabelNode", () => ({
  LabelNode: () => <div data-testid="LabelNode" />,
}));
vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: () => false,
}));
vi.mock("@/store", () => ({
  featureToRef: () => ({ value: false }),
}));
vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: { FEATURE_ENVIRONMENT_TIERS: "FEATURE_ENVIRONMENT_TIERS" },
}));
vi.mock("@/types", () => ({
  NULL_ENVIRONMENT_NAME: "environments/-",
  UNKNOWN_ENVIRONMENT_NAME: "environments/-1",
}));
vi.mock("@/utils", () => ({
  hexToRgb: () => [128, 128, 128],
}));

let Label: typeof import("./Label").Label;

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
  type: "instance" | "environment" | "database" | "label"
): SQLEditorTreeNode =>
  ({
    key: `${type}/x`,
    meta: {
      type,
      target: {
        name: "environments/dev",
        title: "Dev",
        tags: {},
      },
    },
  }) as unknown as SQLEditorTreeNode;

beforeEach(async () => {
  ({ Label } = await import("./Label"));
});

describe("Label", () => {
  test("dispatches to InstanceNode for type=instance", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Label node={makeNode("instance")} keyword="" checked={false} />
    );
    render();
    expect(
      container.querySelector("[data-testid='InstanceNode']")
    ).not.toBeNull();
    unmount();
  });

  test("dispatches to DatabaseNode for type=database", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Label node={makeNode("database")} keyword="" checked={false} />
    );
    render();
    expect(
      container.querySelector("[data-testid='DatabaseNode']")
    ).not.toBeNull();
    unmount();
  });

  test("dispatches to LabelNode for type=label", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Label node={makeNode("label")} keyword="" checked={false} />
    );
    render();
    expect(container.querySelector("[data-testid='LabelNode']")).not.toBeNull();
    unmount();
  });

  test("renders environment title inline for type=environment", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Label node={makeNode("environment")} keyword="" checked={false} />
    );
    render();
    expect(container.textContent).toContain("Dev");
    unmount();
  });
});
