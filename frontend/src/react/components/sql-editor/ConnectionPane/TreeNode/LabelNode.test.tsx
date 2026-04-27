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

let LabelNode: typeof import("./LabelNode").LabelNode;

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

const makeNode = (target: {
  key: string;
  value: string;
}): SQLEditorTreeNode<"label"> =>
  ({
    key: `label/${target.key}/${target.value}`,
    label: target.key,
    meta: { type: "label", target },
  }) as unknown as SQLEditorTreeNode<"label">;

beforeEach(async () => {
  ({ LabelNode } = await import("./LabelNode"));
});

describe("LabelNode", () => {
  test("renders 'key: value' when value is present", () => {
    const { container, render, unmount } = renderIntoContainer(
      <LabelNode node={makeNode({ key: "env", value: "prod" })} keyword="" />
    );
    render();
    expect(container.textContent).toContain("env");
    expect(container.textContent).toContain(":");
    expect(container.textContent).toContain("prod");
    unmount();
  });

  test("falls back to label.empty-label-value when value is empty", () => {
    const { container, render, unmount } = renderIntoContainer(
      <LabelNode node={makeNode({ key: "env", value: "" })} keyword="" />
    );
    render();
    expect(container.textContent).toContain("label.empty-label-value");
    unmount();
  });
});
