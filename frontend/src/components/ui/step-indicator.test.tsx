import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test } from "vitest";
import { StepIndicator } from "./step-indicator";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  act(() => {
    root.render(element);
  });

  return {
    container,
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

describe("StepIndicator", () => {
  test("marks reached steps with accent and future steps as muted", () => {
    const { container, unmount } = renderIntoContainer(
      <StepIndicator
        currentIndex={1}
        steps={[
          { title: "First" },
          { title: "Second" },
          { title: "Third" },
        ]}
      />
    );

    const labels = [...container.querySelectorAll("li span")];
    expect(labels.map((label) => label.textContent)).toEqual([
      "First",
      "Second",
      "Third",
    ]);
    expect(labels[0].className).toContain("text-accent");
    expect(labels[1].className).toContain("text-accent");
    expect(labels[2].className).toContain("text-control-light");
    expect(container.querySelector("svg")).toBeTruthy();
    expect(container.textContent).not.toContain("1");
    expect(container.textContent).toContain("2");
    expect(container.textContent).toContain("3");
    expect(container.querySelectorAll(".bg-control-border")).toHaveLength(2);

    unmount();
  });
});
