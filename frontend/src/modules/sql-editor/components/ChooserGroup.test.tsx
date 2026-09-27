import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("./DatabaseChooser", () => ({
  DatabaseChooser: () => <div data-testid="database-chooser" />,
}));

vi.mock("./SchemaChooser", () => ({
  SchemaChooser: () => <div data-testid="schema-chooser" />,
}));

vi.mock("./ContainerChooser", () => ({
  ContainerChooser: () => <div data-testid="container-chooser" />,
}));

let ChooserGroup: typeof import("./ChooserGroup").ChooserGroup;

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

beforeEach(async () => {
  vi.clearAllMocks();
  ({ ChooserGroup } = await import("./ChooserGroup"));
});

describe("ChooserGroup", () => {
  test("renders all three chooser components inside a flex wrapper", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ChooserGroup />
    );
    render();
    expect(
      container.querySelector("[data-testid='database-chooser']")
    ).not.toBeNull();
    expect(
      container.querySelector("[data-testid='schema-chooser']")
    ).not.toBeNull();
    expect(
      container.querySelector("[data-testid='container-chooser']")
    ).not.toBeNull();
    unmount();
  });

  test("wraps choosers in a flex container", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ChooserGroup />
    );
    render();
    const wrapper = container.firstElementChild as HTMLElement;
    expect(wrapper?.className).toContain("flex");
    unmount();
  });
});
