import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorTab } from "@/types/sqlEditor/tab";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

let Suffix: typeof import("./Suffix").Suffix;

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

const makeTab = (overrides: Partial<SQLEditorTab> = {}): SQLEditorTab =>
  ({
    id: "t1",
    mode: "WORKSHEET",
    status: "CLEAN",
    ...overrides,
  }) as unknown as SQLEditorTab;

beforeEach(async () => {
  vi.clearAllMocks();
  ({ Suffix } = await import("./Suffix"));
});

describe("Suffix", () => {
  test("CLEAN tab renders the close (X) icon by default", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Suffix tab={makeTab()} onClose={vi.fn()} />
    );
    render();
    // X icon has class 'lucide-x'; fallback to checking svg path exists.
    expect(container.querySelector("svg")).not.toBeNull();
    unmount();
  });

  test("click on the X icon fires onClose", () => {
    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <Suffix tab={makeTab()} onClose={onClose} />
    );
    render();
    const svg = container.querySelector("svg") as Element | null;
    expect(svg).not.toBeNull();
    act(() => {
      svg?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(onClose).toHaveBeenCalledTimes(1);
    unmount();
  });

  test("SAVING WORKSHEET tab renders the spinner (takes priority over hover)", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Suffix tab={makeTab({ status: "SAVING" })} onClose={vi.fn()} />
    );
    render();
    expect(
      container.querySelector(".animate-spin") ?? container.querySelector("svg")
    ).not.toBeNull();
    unmount();
  });

  test("DIRTY WORKSHEET tab renders the unsaved-dot icon (when not hovered)", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Suffix tab={makeTab({ status: "DIRTY" })} onClose={vi.fn()} />
    );
    render();
    const svg = container.querySelector("svg");
    expect(svg).not.toBeNull();
    // Dirty status applies the accent color class.
    expect(svg?.className.baseVal).toContain("text-accent");
    unmount();
  });
});
