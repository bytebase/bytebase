import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

let WelcomeButton: typeof import("./WelcomeButton").WelcomeButton;

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
  ({ WelcomeButton } = await import("./WelcomeButton"));
});

describe("WelcomeButton", () => {
  test("matches the workspace landing quick-link style", () => {
    const { container, render, unmount } = renderIntoContainer(
      <WelcomeButton icon={<span data-testid="icon">I</span>}>
        Hello
      </WelcomeButton>
    );
    render();
    const button = container.querySelector("button");
    expect(button?.textContent).toContain("Hello");
    expect(container.querySelector('[data-testid="icon"]')).not.toBeNull();
    expect(button?.className).toContain("rounded-sm");
    expect(button?.className).toContain("bg-background");
    expect(button?.className).toContain("hover:bg-control-bg");
    expect(button?.className).toContain("gap-x-2");
    unmount();
  });

  test("invokes onClick when clicked", () => {
    const handler = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <WelcomeButton icon={<span>I</span>} onClick={handler}>
        Click
      </WelcomeButton>
    );
    render();
    const button = container.querySelector("button");
    act(() => {
      button?.click();
    });
    expect(handler).toHaveBeenCalledTimes(1);
    unmount();
  });

});
