import { act, createElement, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import { RadioGroup, RadioGroupItem } from "./radio-group";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => {
    root.render(element);
  });
  return {
    container,
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

const getRadio = (container: HTMLElement) => {
  const radio = container.querySelector('[role="radio"]');
  if (!radio) throw new Error("radio not found");
  return radio as HTMLElement;
};

describe("RadioGroupItem", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("prevents the radio control from shrinking in long option labels", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        RadioGroup,
        { value: "workspace", onValueChange: () => undefined },
        createElement(
          RadioGroupItem,
          { value: "workspace" },
          createElement(
            "span",
            null,
            "Use issue to request, review, rollout, and version database changes"
          )
        )
      )
    );

    expect(getRadio(container).className).toContain("size-4");
    expect(getRadio(container).className).toContain("shrink-0");

    unmount();
  });
});
