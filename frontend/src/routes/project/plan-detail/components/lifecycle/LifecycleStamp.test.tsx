import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import { LifecycleStamp } from "./LifecycleStamp";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("LifecycleStamp", () => {
  let root: ReturnType<typeof createRoot> | undefined;

  afterEach(() => {
    act(() => root?.unmount());
    root = undefined;
    document.body.innerHTML = "";
  });

  test("uses a pill shape for non-interactive lifecycle status", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    act(() => {
      root?.render(<LifecycleStamp tone="success">Deployed</LifecycleStamp>);
    });

    const stamp = container.querySelector("span");
    expect(stamp?.className).toContain("rounded-full");
  });
});
