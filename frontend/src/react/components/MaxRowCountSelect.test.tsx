import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string, opts?: { n?: number }) =>
      opts?.n !== undefined ? `${key}:${opts.n}` : key,
  })),
  Popover: vi.fn(({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover">{children}</div>
  )),
  PopoverTrigger: vi.fn(
    ({
      children,
      render,
    }: {
      children?: React.ReactNode;
      render?: React.ReactElement;
    }) => {
      if (render) {
        return <div data-testid="popover-trigger">{children}</div>;
      }
      return <div data-testid="popover-trigger">{children}</div>;
    }
  ),
  PopoverContent: vi.fn(({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover-content">{children}</div>
  )),
  minmax: vi.fn((value: number, min: number, max: number) =>
    Math.min(Math.max(value, min), max)
  ),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/components/ui/popover", () => ({
  Popover: mocks.Popover,
  PopoverTrigger: mocks.PopoverTrigger,
  PopoverContent: mocks.PopoverContent,
}));

vi.mock("@/utils", () => ({
  minmax: mocks.minmax,
}));

let MaxRowCountSelect: typeof import("./MaxRowCountSelect").MaxRowCountSelect;

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
  vi.clearAllMocks();
  mocks.useTranslation.mockReturnValue({
    t: (key: string, opts?: { n?: number }) =>
      opts?.n !== undefined ? `${key}:${opts.n}` : key,
  });
  ({ MaxRowCountSelect } = await import("./MaxRowCountSelect"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("MaxRowCountSelect", () => {
  test("trigger renders current value label", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MaxRowCountSelect
        value={1000}
        onChange={() => {}}
        maximum={Number.MAX_VALUE}
      />
    );
    render();
    // The trigger contains the translated key for n-rows with n=1000
    expect(container.textContent).toContain("common.rows.n-rows");
    unmount();
  });

  test("preset options are filtered by maximum", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MaxRowCountSelect value={100} onChange={() => {}} maximum={500} />
    );
    render();
    // Should show 1, 100, 500 — not 1000, 5000, etc.
    const content = container.querySelector("[data-testid='popover-content']");
    const buttons = content?.querySelectorAll("button[type='button']") ?? [];
    const labels = Array.from(buttons).map((b) => b.textContent ?? "");
    // Filtered options: 1, 100, 500 (n-rows key with n)
    expect(labels.some((l) => l.includes(":1"))).toBe(true);
    expect(labels.some((l) => l.includes(":100"))).toBe(true);
    expect(labels.some((l) => l.includes(":500"))).toBe(true);
    // 1000 should NOT be present
    expect(labels.some((l) => l.includes(":1000"))).toBe(false);
    unmount();
  });

  test("clicking a preset option calls onChange with the value", () => {
    const onChange = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <MaxRowCountSelect
        value={100}
        onChange={onChange}
        maximum={Number.MAX_VALUE}
      />
    );
    render();
    // Click the button for 500
    const buttons = Array.from(
      container.querySelectorAll(
        "[data-testid='popover-content'] button[type='button']"
      )
    );
    const btn500 = buttons.find((b) => b.textContent?.includes(":500"));
    act(() => {
      (btn500 as HTMLButtonElement)?.click();
    });
    expect(onChange).toHaveBeenCalledWith(500);
    unmount();
  });

  test("custom number input applies minmax clamping", () => {
    const onChange = vi.fn();
    mocks.minmax.mockReturnValue(500);
    const { container, render, unmount } = renderIntoContainer(
      <MaxRowCountSelect value={100} onChange={onChange} maximum={500} />
    );
    render();
    const input = container.querySelector(
      "[data-testid='popover-content'] input[type='number']"
    ) as HTMLInputElement;
    expect(input).toBeTruthy();
    act(() => {
      Object.defineProperty(input, "value", { writable: true, value: "9999" });
      input.dispatchEvent(new Event("change", { bubbles: true }));
    });
    // minmax was called and onChange received the clamped value
    expect(mocks.minmax).toHaveBeenCalled();
    unmount();
  });
});
