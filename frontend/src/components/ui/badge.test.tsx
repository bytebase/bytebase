import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import { Badge } from "./badge";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("Badge", () => {
  let root: ReturnType<typeof createRoot> | undefined;

  afterEach(() => {
    act(() => root?.unmount());
    root = undefined;
    document.body.innerHTML = "";
  });

  test("uses the compact label geometry by default", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    act(() => {
      root?.render(<Badge variant="success">Active</Badge>);
    });

    const badge = container.querySelector("span");
    expect(badge?.className).toContain("rounded-xs");
    expect(badge?.className).toContain("px-1.5");
    expect(badge?.className).toContain("py-0.5");
    expect(badge?.className).toContain("text-xs");
    expect(badge?.className).toContain("border-transparent");
  });

  test("supports the roomier form-token density", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    act(() => {
      root?.render(
        <Badge variant="default" size="sm">
          Workspace Admin
        </Badge>
      );
    });

    const badge = container.querySelector("span");
    expect(badge?.className).toContain("h-7");
    expect(badge?.className).toContain("px-2");
    expect(badge?.className).toContain("text-sm");
  });
});
