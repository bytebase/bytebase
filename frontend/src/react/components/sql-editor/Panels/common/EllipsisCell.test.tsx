import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import { EllipsisCell } from "./EllipsisCell";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  act(() => {
    root.render(element);
  });
  return {
    container,
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

afterEach(() => {
  document.body.innerHTML = "";
});

describe("EllipsisCell", () => {
  test("renders content when no keyword", () => {
    const { container, unmount } = renderIntoContainer(
      <EllipsisCell content="hello world" />
    );
    expect(container.textContent).toBe("hello world");
    expect(container.querySelector("b")).toBeNull();
    unmount();
  });

  test("highlights the keyword inside the content", () => {
    const { container, unmount } = renderIntoContainer(
      <EllipsisCell content="public.users" keyword="users" />
    );
    const highlight = container.querySelector("b");
    expect(highlight).not.toBeNull();
    expect(highlight?.textContent).toBe("users");
    expect(highlight?.className).toContain("text-accent");
    unmount();
  });

  test("does not show the tooltip when content fits", () => {
    const { container, unmount } = renderIntoContainer(
      <EllipsisCell content="short" />
    );
    const span = container.querySelector("span") as HTMLSpanElement;
    act(() => {
      span.dispatchEvent(new MouseEvent("mouseenter", { bubbles: true }));
    });
    // scrollWidth === clientWidth in jsdom, so the tooltip is suppressed.
    expect(document.querySelector("[role='tooltip']")).toBeNull();
    unmount();
  });
});
