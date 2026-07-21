import { act, createElement, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { Checkbox } from "./checkbox";

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
    rerender: (next: ReactElement) => {
      act(() => {
        root.render(next);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

const getCheckbox = (container: HTMLElement) => {
  const cb = container.querySelector('[role="checkbox"]');
  if (!cb) throw new Error("checkbox not found");
  return cb as HTMLElement;
};

describe("Checkbox", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("renders unchecked by default", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(Checkbox, { checked: false, "aria-label": "cb" })
    );
    expect(getCheckbox(container).getAttribute("aria-checked")).toBe("false");
    unmount();
  });

  test("renders checked", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(Checkbox, { checked: true, "aria-label": "cb" })
    );
    expect(getCheckbox(container).getAttribute("aria-checked")).toBe("true");
    unmount();
  });

  test("renders indeterminate", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(Checkbox, { checked: "indeterminate", "aria-label": "cb" })
    );
    expect(getCheckbox(container).getAttribute("aria-checked")).toBe("mixed");
    unmount();
  });

  test("click on unchecked emits true", () => {
    const onCheckedChange = vi.fn();
    const { container, unmount } = renderIntoContainer(
      createElement(Checkbox, {
        checked: false,
        onCheckedChange,
        "aria-label": "cb",
      })
    );
    act(() => {
      getCheckbox(container).dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(onCheckedChange).toHaveBeenCalledWith(true);
    unmount();
  });

  test("click on checked emits false", () => {
    const onCheckedChange = vi.fn();
    const { container, unmount } = renderIntoContainer(
      createElement(Checkbox, {
        checked: true,
        onCheckedChange,
        "aria-label": "cb",
      })
    );
    act(() => {
      getCheckbox(container).dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(onCheckedChange).toHaveBeenCalledWith(false);
    unmount();
  });

  test("click on indeterminate emits true", () => {
    const onCheckedChange = vi.fn();
    const { container, unmount } = renderIntoContainer(
      createElement(Checkbox, {
        checked: "indeterminate",
        onCheckedChange,
        "aria-label": "cb",
      })
    );
    act(() => {
      getCheckbox(container).dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(onCheckedChange).toHaveBeenCalledWith(true);
    unmount();
  });

  test("disabled blocks onCheckedChange", () => {
    const onCheckedChange = vi.fn();
    const { container, unmount } = renderIntoContainer(
      createElement(Checkbox, {
        checked: false,
        disabled: true,
        onCheckedChange,
        "aria-label": "cb",
      })
    );
    act(() => {
      getCheckbox(container).dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(onCheckedChange).not.toHaveBeenCalled();
    unmount();
  });

  test("size md renders size-4 class", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(Checkbox, { checked: false, "aria-label": "cb" })
    );
    expect(getCheckbox(container).className).toContain("size-4");
    unmount();
  });

  test("size sm renders size-3.5 class", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(Checkbox, {
        checked: false,
        size: "sm",
        "aria-label": "cb",
      })
    );
    expect(getCheckbox(container).className).toContain("size-3.5");
    unmount();
  });

  test("onClick prop is forwarded", () => {
    const onClick = vi.fn();
    const { container, unmount } = renderIntoContainer(
      createElement(Checkbox, {
        checked: false,
        onClick,
        "aria-label": "cb",
      })
    );
    act(() => {
      getCheckbox(container).dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(onClick).toHaveBeenCalled();
    unmount();
  });
});
