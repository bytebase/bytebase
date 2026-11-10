import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test } from "vitest";
import { Textarea } from "./textarea";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("Textarea", () => {
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
      root.render(<Textarea aria-invalid="true" />);
    });

    const textarea = container.querySelector("textarea");
    expect(textarea).not.toBeNull();
    expect(textarea?.className).toContain("aria-invalid:border-error");
    expect(textarea?.className).toContain("aria-invalid:focus:border-error");
    expect(textarea?.className).toContain("aria-invalid:focus:ring-1");
    expect(textarea?.className).toContain("aria-invalid:focus:ring-error/30");
    expect(textarea?.className).toContain("aria-invalid:focus:ring-inset");
  });
});
