import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test } from "vitest";
import { Input } from "./input";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("Input", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
  });

  test("uses an inset error focus ring when invalid", () => {
    act(() => {
      root.render(<Input aria-invalid="true" />);
    });

    const input = container.querySelector("input");
    expect(input).not.toBeNull();
    expect(input?.className).toContain("aria-invalid:border-error");
    expect(input?.className).toContain("aria-invalid:focus:border-error");
    expect(input?.className).toContain("aria-invalid:focus:ring-1");
    expect(input?.className).toContain("aria-invalid:focus:ring-error/30");
    expect(input?.className).toContain("aria-invalid:focus:ring-inset");
  });
});
