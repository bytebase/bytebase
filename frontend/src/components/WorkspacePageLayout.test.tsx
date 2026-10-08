import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test } from "vitest";
import { WorkspacePageLayout } from "./WorkspacePageLayout";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  container.remove();
});

describe("WorkspacePageLayout", () => {
  test("only exposes horizontal overflow when a page opts in", () => {
    act(() => {
      root.render(
        <>
          <WorkspacePageLayout />
          <WorkspacePageLayout allowHorizontalOverflow />
        </>
      );
    });

    const pages = container.querySelectorAll("[data-slot='workspace-page-layout']");
    expect(pages).toHaveLength(2);
    expect(pages[0]?.className).not.toBe(pages[1]?.className);
  });
});
