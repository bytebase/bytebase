import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { PanelSearchBox } from "./PanelSearchBox";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("PanelSearchBox", () => {
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

  test("keeps the placeholder clear of the search icon", () => {
    act(() => {
      root.render(<PanelSearchBox value="" onChange={() => {}} />);
    });

    const input = container.querySelector("input") as HTMLInputElement;
    expect(input.style.paddingInlineStart).toBe("1.75rem");
  });
});
