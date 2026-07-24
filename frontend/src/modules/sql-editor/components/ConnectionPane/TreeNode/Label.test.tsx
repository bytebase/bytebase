import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorTreeNode } from "@/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  EnvironmentLabel: vi.fn(
    ({ environment }: { environment: { title: string } }) => (
      <span data-testid="SharedEnvironmentLabel">{environment.title}</span>
    )
  ),
}));

vi.mock("./InstanceNode", () => ({
  InstanceNode: () => <div data-testid="InstanceNode" />,
}));
vi.mock("./DatabaseNode", () => ({
  DatabaseNode: () => <div data-testid="DatabaseNode" />,
}));
vi.mock("./LabelNode", () => ({
  LabelNode: () => <div data-testid="LabelNode" />,
}));
vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: mocks.EnvironmentLabel,
}));
vi.mock("@/modules/sql-editor/components/theme/SQLEditorThemeScope", () => ({
  useSQLEditorTheme: () => ({
    tokens: {
      "--color-background": "#1e1e1e",
      "--color-accent-hover": "#818cf8",
    },
  }),
}));
vi.mock("@/modules/sql-editor/components/theme/derive", () => ({
  isDarkTheme: () => true,
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
  vi.clearAllMocks();
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

  test("uses shared EnvironmentLabel for type=environment", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Label node={makeNode("environment")} keyword="Dev" checked={false} />
    );
    render();
    expect(
      container.querySelector("[data-testid='SharedEnvironmentLabel']")
    ).not.toBeNull();
    expect(mocks.EnvironmentLabel).toHaveBeenCalledWith(
      expect.objectContaining({
        environment: expect.objectContaining({ title: "Dev" }),
        link: false,
        styleOptions: {
          defaultColorTextColor: "#818cf8",
          backgroundAlpha: 0.18,
        },
      }),
      undefined
    );
    expect(mocks.EnvironmentLabel.mock.calls[0][0]).not.toHaveProperty(
      "keyword"
    );
    unmount();
  });
});
